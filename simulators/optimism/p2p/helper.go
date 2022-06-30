package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/hive/hivesim"
	"github.com/kr/pretty"
)

// default timeout for RPC calls
var rpcTimeout = 10 * time.Second

// TestClient is the environment of a single test.
type TestEnv struct {
	*hivesim.T
	Eth *ethclient.Client

	// This holds most recent context created by the Ctx method.
	// Every time Ctx is called, it creates a new context with the default
	// timeout and cancels the previous one.
	lastCtx    context.Context
	lastCancel context.CancelFunc
}

// Ctx returns a context with the default timeout.
// For subsequent calls to Ctx, it also cancels the previous context.
func (t *TestEnv) Ctx() context.Context {
	if t.lastCtx != nil {
		t.lastCancel()
	}
	t.lastCtx, t.lastCancel = context.WithTimeout(context.Background(), rpcTimeout)
	return t.lastCtx
}

func waitSynced(c *rpc.Client) (err error) {
	var (
		timeout     = 20 * time.Second
		end         = time.Now().Add(timeout)
		ctx, cancel = context.WithDeadline(context.Background(), end)
	)
	defer func() {
		cancel()
		if err == context.DeadlineExceeded {
			err = fmt.Errorf("didn't sync within timeout of %v", 20*time.Second)
		}
	}()

	ec := ethclient.NewClient(c)
	for {
		progress, err := ec.SyncProgress(ctx)
		if err != nil {
			return err
		}
		head, err := ec.BlockNumber(ctx)
		if err != nil {
			return err
		}
		if progress == nil && head > 0 {
			return nil // success!
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// Naive generic function that works in all situations.
// A better solution is to use logs to wait for confirmations.
func waitForTxConfirmations(t *TestEnv, txHash common.Hash, n uint64) (*types.Receipt, error) {
	var (
		receipt    *types.Receipt
		startBlock *types.Block
		err        error
	)

	for i := 0; i < 90; i++ {
		receipt, err = t.Eth.TransactionReceipt(t.Ctx(), txHash)
		if err != nil && err != ethereum.NotFound {
			return nil, err
		}
		if receipt != nil {
			break
		}
		time.Sleep(time.Second)
	}
	if receipt == nil {
		return nil, ethereum.NotFound
	}

	if startBlock, err = t.Eth.BlockByNumber(t.Ctx(), nil); err != nil {
		return nil, err
	}

	for i := 0; i < 90; i++ {
		currentBlock, err := t.Eth.BlockByNumber(t.Ctx(), nil)
		if err != nil {
			return nil, err
		}

		if startBlock.NumberU64()+n >= currentBlock.NumberU64() {
			if checkReceipt, err := t.Eth.TransactionReceipt(t.Ctx(), txHash); checkReceipt != nil {
				if bytes.Compare(receipt.PostState, checkReceipt.PostState) == 0 {
					return receipt, nil
				} else { // chain reorg
					waitForTxConfirmations(t, txHash, n)
				}
			} else {
				return nil, err
			}
		}

		time.Sleep(time.Second)
	}

	return nil, ethereum.NotFound
}

// loggingRoundTrip writes requests and responses to the test log.
type loggingRoundTrip struct {
	t     *hivesim.T
	inner http.RoundTripper
}

func (rt *loggingRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	// Read and log the request body.
	reqBytes, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return nil, err
	}
	rt.t.Logf(">>  %s", bytes.TrimSpace(reqBytes))
	reqCopy := *req
	reqCopy.Body = ioutil.NopCloser(bytes.NewReader(reqBytes))

	// Do the round trip.
	resp, err := rt.inner.RoundTrip(&reqCopy)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read and log the response bytes.
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	respCopy := *resp
	respCopy.Body = ioutil.NopCloser(bytes.NewReader(respBytes))
	rt.t.Logf("<<  %s", bytes.TrimSpace(respBytes))
	return &respCopy, nil
}

// diff checks whether x and y are deeply equal, returning a description
// of their differences if they are not equal.
func diff(x, y interface{}) (d string) {
	for _, l := range pretty.Diff(x, y) {
		d += l + "\n"
	}
	return d
}
