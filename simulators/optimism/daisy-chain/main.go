package main

import (
	"context"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/hive/hivesim"
	"github.com/ethereum/hive/optimism"
	"github.com/stretchr/testify/require"
)

// Test constants
var (
	// GoerliSequencerRPC is the RPC endpoint of the Goerli sequencer.
	GoerliSequencerRPC = "https://goerli-sequencer.optimism.io"

	// HistoricalSequencerRPC is the RPC endpoint of the historical sequencer on Goerli.
	HistoricalSequencerRPC = "https://goerli-historical-0.optimism.io"

	// MockAddr is a mock address used in the tests.
	MockAddr = common.HexToAddress("0x00000000000000000000000000000000000badc0de")

	// IDPrecompile is the address of the identity precompile. Used for gas estimation.
	IDPrecompile = common.HexToAddress("0x0000000000000000000000000000000000000004")
)

var tests = []*optimism.TestSpec{
	{Name: "daisy-chain-debug_traceBlockByNumber", Run: debugTraceBlockByNumberTest},
	{Name: "daisy-chain-debug_traceBlockByHash", Run: debugTraceBlockByHashTest},
	{Name: "daisy-chain-debug_traceTransaction", Run: debugTraceTransactionTest},
	{Name: "daisy-chain-debug_traceCall", Run: debugTraceCallTest},
	{Name: "daisy-chain-eth_call", Run: ethCallTest},
	{Name: "daisy-chain-eth_estimateGas", Run: ethEstimateGasTest},
}

func main() {
	sim := hivesim.New()
	for _, forkName := range optimism.AllOptimismForkConfigs {
		forkName := forkName
		suite := hivesim.Suite{
			Name:        "optimism daisy-chain - " + forkName,
			Description: "Tests the daisy-chain functionality of op-geth.",
		}
		suite.Add(&hivesim.TestSpec{
			Name:        "daisy-chain",
			Description: "Tests the daisy chain.",
			Run:         runAllTests(tests, forkName),
		})
		hivesim.MustRunSuite(sim, suite)
	}
}

func runAllTests(tests []*optimism.TestSpec, fork string) func(t *hivesim.T) {
	return func(t *hivesim.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		// Spin up an op-geth node that has the daisy-chain flag enabled.
		d := optimism.NewDevnet(t)
		d.InitChain(120, 120, 30, nil, fork)
		d.AddOpL2(hivesim.Params{
			"HIVE_OP_GETH_USE_GOERLI_DATADIR": "true",
			"HIVE_OP_GETH_SEQUENCER_HTTP":     GoerliSequencerRPC,
			"HIVE_OP_GETH_HISTORICAL_RPC":     HistoricalSequencerRPC,
		})
		d.WaitUpOpL2Engine(0, time.Second*10)

		// Seed the random number generator.
		rand.Seed(time.Now().UnixNano())

		optimism.RunTests(ctx, t, &optimism.RunTestsParams{
			Devnet:      d,
			Tests:       tests,
			Concurrency: 40,
		})
	}
}

// txTraceResult is the result of a single transaction trace.
type txTraceResult struct {
	Result interface{} `json:"result,omitempty"` // Trace results produced by the tracer
	Error  string      `json:"error,omitempty"`  // Trace failure produced by the tracer
}

// blockTraceResult represents the results of tracing a single block when an entire
// chain is being traced.
type blockTraceResult struct {
	Block  hexutil.Uint64   `json:"block"`  // Block number corresponding to this trace
	Hash   common.Hash      `json:"hash"`   // Block hash corresponding to this trace
	Traces []*txTraceResult `json:"traces"` // Trace results produced by the task
}

// Tests that a daisy-chained debug_traceBlockByNumber call to `op-geth` works as expected.
func debugTraceBlockByNumberTest(t *hivesim.T, env *optimism.TestEnv) {
	// Grab a random historical block
	blockNumber := getRandomHistoricalBlockHex()

	// Grab the result of the trace call from op-geth. This request should be daisy-chained.
	opGeth := env.Devnet.GetOpL2Engine(0).RPC()
	var opGethRes []*txTraceResult
	err := opGeth.CallContext(env.Ctx(), &opGethRes, "debug_traceBlockByNumber", blockNumber)
	require.NoError(t, err, "failed to call debug_traceBlockByNumber on op-geth")

	// Grab the result of the trace call from the historical sequencer endpoint. The result
	// should be the same.
	histSeq, err := rpc.DialHTTP(HistoricalSequencerRPC)
	require.NoError(t, err, "failed to dial historical sequencer RPC")
	var histSeqRes []*txTraceResult
	err = histSeq.CallContext(env.Ctx(), &histSeqRes, "debug_traceBlockByNumber", blockNumber)
	require.NoError(t, err, "failed to call debug_traceBlockByNumber on historical sequencer")

	// Compare the results.
	require.Equal(t, histSeqRes, opGethRes, "results from historical sequencer and op-geth do not match")
}

// Tests that a daisy-chained debug_traceBlockByHash call to `op-geth` works as expected.
func debugTraceBlockByHashTest(t *hivesim.T, env *optimism.TestEnv) {
	// Grab a blockhash at a random block that is within the historical window.
	block, err := env.Devnet.GetOpL2Engine(0).EthClient().BlockByNumber(env.Ctx(), getRandomHistoricalBlockNr())
	require.NoError(t, err, "failed to get block by number")
	blockHash := block.Hash()

	// Grab the result of the trace call from op-geth. This request should be daisy-chained.
	opGeth := env.Devnet.GetOpL2Engine(0).RPC()
	var opGethRes []*txTraceResult
	err = opGeth.CallContext(env.Ctx(), &opGethRes, "debug_traceBlockByHash", blockHash.String())
	require.NoError(t, err, "failed to call debug_traceBlockByHash on op-geth")

	// Grab the result of the trace call from the historical sequencer endpoint. The result
	// should be the same.
	histSeq, err := rpc.DialHTTP(HistoricalSequencerRPC)
	require.NoError(t, err, "failed to dial historical sequencer RPC")
	var histSeqRes []*txTraceResult
	err = histSeq.CallContext(env.Ctx(), &histSeqRes, "debug_traceBlockByHash", blockHash.String())
	require.NoError(t, err, "failed to call debug_traceBlockByHash on historical sequencer")

	// Compare the results.
	require.Equal(t, histSeqRes, opGethRes, "results from historical sequencer and op-geth do not match")
}

// Tests that a daisy-chained debug_traceTransaction call to `op-geth` works as expected.
func debugTraceTransactionTest(t *hivesim.T, env *optimism.TestEnv) {
	// Grab a transaction hash at a random block that is within the historical window.
	block, err := env.Devnet.GetOpL2Engine(0).EthClient().BlockByNumber(env.Ctx(), getRandomHistoricalBlockNr())
	require.NoError(t, err, "failed to get block by number")
	txHash := block.Transactions()[0].Hash()

	// Grab the result of the trace call from op-geth. This request should be daisy-chained.
	opGeth := env.Devnet.GetOpL2Engine(0).RPC()
	var opGethRes *txTraceResult
	err = opGeth.CallContext(env.Ctx(), &opGethRes, "debug_traceTransaction", txHash.String())
	require.NoError(t, err, "failed to call debug_traceTransaction on op-geth")

	// Grab the result of the trace call from the historical sequencer endpoint. The result
	// should be the same.
	histSeq, err := rpc.DialHTTP(HistoricalSequencerRPC)
	require.NoError(t, err, "failed to dial historical sequencer RPC")
	var histSeqRes *txTraceResult
	err = histSeq.CallContext(env.Ctx(), &histSeqRes, "debug_traceTransaction", txHash.String())
	require.NoError(t, err, "failed to call debug_traceTransaction on historical sequencer")

	// Compare the results.
	require.Equal(t, histSeqRes, opGethRes, "results from historical sequencer and op-geth do not match")
}

// Tests that a daisy-chained debug_traceCall call to `op-geth` fails as expected.
func debugTraceCallTest(t *hivesim.T, env *optimism.TestEnv) {
	// Grab the result of the trace call from op-geth. This request should be daisy-chained.
	opGeth := env.Devnet.GetOpL2Engine(0).RPC()
	err := opGeth.CallContext(env.Ctx(), nil, "debug_traceCall", make(map[string]interface{}), getRandomHistoricalBlockHex())

	// The debug_traceCall method should not be implemented in op-geth's RPC.
	require.Error(t, err, "debug_traceCall should not be implemented in op-geth")
	require.Equal(t, err.Error(), "l2geth does not have a debug_traceCall method", "debug_traceCall should not be implemented in op-geth")
}

// Tests that a daisy-chaned eth_call to `op-geth` works as expected.
func ethCallTest(t *hivesim.T, env *optimism.TestEnv) {
	// Craft the payload to send to eth_call.
	tx := types.NewTransaction(0, MockAddr, big.NewInt(0), 100_000, big.NewInt(0), []byte{})
	blockNumber := getRandomHistoricalBlockHex()
	stateOverride := make(map[string]interface{})
	// Store a simple contract @ 0xdeadbeef that returns its own balance (1 ether).
	stateOverride[MockAddr.String()] =
		struct {
			Balance *hexutil.Big   `json:"balance"`
			Nonce   hexutil.Uint64 `json:"nonce"`
			Code    hexutil.Bytes  `json:"code"`
		}{
			Balance: (*hexutil.Big)(big.NewInt(1e18)),
			Nonce:   hexutil.Uint64(0),
			// SELFBALANCE
			// PUSH1 0x00
			// MSTORE
			// PUSH1 0x20
			// PUSH1 0x00
			// RETURN
			Code: []byte{0x47, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xF3},
		}

	// Grab the result of the call from op-geth. This request should be daisy-chained.
	opGeth := env.Devnet.GetOpL2Engine(0).RPC()
	var opGethRes hexutil.Bytes
	err := opGeth.CallContext(env.Ctx(), &opGethRes, "eth_call", tx, blockNumber, stateOverride)
	require.NoError(t, err, "failed to call eth_call on op-geth")

	// Grab the result of the call from the historical sequencer endpoint. The result
	// should be the same.
	histSeq, err := rpc.DialHTTP(HistoricalSequencerRPC)
	require.NoError(t, err, "failed to dial historical sequencer RPC")
	var histSeqRes hexutil.Bytes
	err = histSeq.CallContext(env.Ctx(), &histSeqRes, "eth_call", tx, blockNumber, stateOverride)
	require.NoError(t, err, "failed to call eth_call on historical sequencer")

	// Compare the results.
	// The expected result is 1e18 in hex (0x0de0b6b3a7640000).
	expectedRes := hexutil.Bytes([]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xd, 0xe0, 0xb6, 0xb3, 0xa7, 0x64, 0x0, 0x0})
	require.Equal(t, histSeqRes, expectedRes, "results from op-geth do not match")
	require.Equal(t, histSeqRes, opGethRes, "results from historical sequencer and op-geth do not match")
}

// Tests that a daisy-chained eth_estimateGas to `op-geth` works as expected.
func ethEstimateGasTest(t *hivesim.T, env *optimism.TestEnv) {
	// Generate a random payload for the identity precompile
	payload := make([]byte, rand.Intn(128))
	_, err := rand.Read(payload)
	require.NoError(t, err, "failed to generate random payload")
	tx := types.NewTransaction(0, IDPrecompile, big.NewInt(0), 100_000, big.NewInt(0), payload)
	blockNumber := getRandomHistoricalBlockHex()

	// Grab the result of the call from op-geth. This request should be daisy-chained.
	opGeth := env.Devnet.GetOpL2Engine(0).RPC()
	var opGethRes hexutil.Uint64
	err = opGeth.CallContext(env.Ctx(), &opGethRes, "eth_estimateGas", tx, blockNumber)
	require.NoError(t, err, "failed to call eth_estimateGas on op-geth")

	// Grab the result of the call from the historical sequencer endpoint. The result
	// should be the same.
	histSeq, err := rpc.DialHTTP(HistoricalSequencerRPC)
	require.NoError(t, err, "failed to dial historical sequencer RPC")
	var histSeqRes hexutil.Uint64
	err = histSeq.CallContext(env.Ctx(), &histSeqRes, "eth_estimateGas", tx, blockNumber)
	require.NoError(t, err, "failed to call eth_estimateGas on historical sequencer")

	// Compare the results.
	require.Greater(t, opGethRes, hexutil.Uint64(0), "gas estimate from op-geth should be greater than 0")
	require.Equal(t, histSeqRes, opGethRes, "results from historical sequencer and op-geth do not match")
}

// getRandomHistoricalBlockHex returns a random historical block number.
// Historical blocks on goerli exist in the range [1, 4061224)
func getRandomHistoricalBlockNr() *big.Int {
	return new(big.Int).SetUint64(uint64(rand.Intn(4061223) + 1))
}

// getRandomHistoricalBlockHex returns a random historical block number in hex.
// Historical blocks on goerli exist in the range [1, 4061224)
func getRandomHistoricalBlockHex() string {
	return hexutil.EncodeBig(getRandomHistoricalBlockNr())
}
