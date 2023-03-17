package main

import (
	"context"
	"time"

	"github.com/ethereum/hive/hivesim"
	"github.com/ethereum/hive/optimism"
	"github.com/stretchr/testify/require"
)

var tests = []*optimism.TestSpec{
	{Name: "daisy-chain", Run: daisyChainTest},
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

		d := optimism.NewDevnet(t)
		require.NoError(t, optimism.StartSequencerDevnet(ctx, d, &optimism.SequencerDevnetParams{
			MaxSeqDrift:   120,
			SeqWindowSize: 120,
			ChanTimeout:   30,
			Fork:          fork,
			IncludeL2Geth: true,
		}))

		optimism.RunTests(ctx, t, &optimism.RunTestsParams{
			Devnet:      d,
			Tests:       tests,
			Concurrency: 40,
		})
	}
}

// TEMP: This is a placeholder test that does nothing.
func daisyChainTest(t *hivesim.T, env *optimism.TestEnv) {
	// TODO
	l2 := env.Devnet.GetOpL2Engine(1)
	err := l2.RPC().Call(nil, "miner_stop")
	if err != nil {
		t.Log(err)
	}
}
