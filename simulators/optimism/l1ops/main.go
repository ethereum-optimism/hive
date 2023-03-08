package main

import (
	"context"
	"time"

	"github.com/ethereum/hive/hivesim"
	"github.com/ethereum/hive/optimism"
	"github.com/stretchr/testify/require"
)

var tests = []*optimism.TestSpec{
	{Name: "deposit simple tx through the portal", Run: simplePortalDepositTest},
	{Name: "deposit contract creation through the portal", Run: contractPortalDepositTest},
	{Name: "erc20 roundtrip through the bridge", Run: erc20RoundtripTest},
	{Name: "simple withdrawal", Run: simpleWithdrawalTest},
	{Name: "failing deposit with mint", Run: failingDepositWithMintTest},
}

func main() {
	sim := hivesim.New()
	for _, forkName := range optimism.AllOptimismForkConfigs {
		forkName := forkName
		suite := hivesim.Suite{
			Name: "optimism l1ops - " + forkName,
			Description: `
Tests deposits, withdrawals, and other L1-related operations against a running node.
`[1:],
		}
		suite.Add(&hivesim.TestSpec{
			Name:        "l1ops",
			Description: "Tests L1 operations.",
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
		}))

		optimism.RunTests(ctx, t, &optimism.RunTestsParams{
			Devnet:      d,
			Tests:       tests,
			Concurrency: 40,
		})
	}
}
