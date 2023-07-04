package types

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestPhaseInflation(t *testing.T) {
	minter := DefaultInitialMinter()

	tests := []struct {
		phase        uint64
		expInflation sdk.Dec
	}{
		// phase 1, inflation: 100% // december 2022 - december 2023
		{1, sdk.NewDecWithPrec(100, 2)},
		// phase 2, inflation: 50% // december 2023 - june 2024
		{2, sdk.NewDecWithPrec(50, 2)},
		// phase 3, inflation: 25% // june 2024 - december 2024
		{3, sdk.NewDecWithPrec(25, 2)},
		// phase 4, inflation: 12.5% // december 2024 - june 2025
		{4, sdk.NewDecWithPrec(125, 3)},
		// phase 5, inflation: 6.125% // june 2025 - december 2025
		{5, sdk.NewDecWithPrec(6125, 5)},
		// phase 6, inflation: 3.0625% // december 2025 - june 2026
		{6, sdk.NewDecWithPrec(30625, 6)},
		// phase 7, inflation: 1.53125% // june 2026 - december 2026
		{7, sdk.NewDecWithPrec(153125, 7)},
		// phase 8, inflation: 0% // INFLATION STOP IN DECEMBER 2026
		{8, sdk.NewDec(0)},
		// phase 9, inflation: still 0%
		{9, sdk.NewDec(0)},
	}
	for i, tc := range tests {
		inflation := minter.PhaseInflationRate(tc.phase)

		require.True(t, inflation.Equal(tc.expInflation),
			"Test Index: %v\nInflation:  %v\nExpected: %v\n", i, inflation, tc.expInflation)
	}
}

func TestNextPhase(t *testing.T) {
	minter := DefaultInitialMinter()
	params := DefaultParams()

	blocksPerYear := uint64(6311520)
	tests := []struct {
		currentBlock, currentPhase, startPhaseBlock, blocksYear, expPhase uint64
	}{
		// genesis
		{1, 0, 0, blocksPerYear, 1},
		// half a year after genesis (no halving for a year so works)
		{3155762, 1, 1, blocksPerYear, 1},
		// last block of phase 1 
		{6311520, 1, 1, blocksPerYear, 1},
		// now we are in phase 2
		{6311521, 1, 1, blocksPerYear, 2},
		// last block of phase 2
		{9467280, 2, 6311521, blocksPerYear, 2},
		// now we are in phase 3 (halving every 6 months)
		{9467281, 2, 6311521, blocksPerYear, 3},
		// last block of phase 3
		{12623040, 3, 9467281, blocksPerYear, 3},
		// now we are in phase 4
		{12623041, 3, 9467281, blocksPerYear, 4},

		//  3155760.5 phase length except 1

		// 1 phase 1
		// 6311520 phase 2
		// 9467280 phase 3
		// 12623041 phase 4
	}
	for i, tc := range tests {
		minter.Phase = tc.currentPhase
		minter.StartPhaseBlock = tc.startPhaseBlock
		params.BlocksPerYear = tc.blocksYear

		phase := minter.NextPhase(params, tc.currentBlock)

		require.True(t, phase == tc.expPhase,
			"Test Index: %v\nPhase:  %v\nExpected: %v\n", i, phase, tc.expPhase)
	}
}

func TestBlockProvision(t *testing.T) {
	minter := InitialMinter(sdk.NewDecWithPrec(1, 1))
	params := DefaultParams()

	secondsPerYear := int64(60 * 60 * 8766)

	tests := []struct {
		annualProvisions int64
		expProvisions    int64
	}{
		{secondsPerYear / 5, 1},
		{secondsPerYear/5 + 1, 1},
		{(secondsPerYear / 5) * 2, 2},
		{(secondsPerYear / 5) / 2, 0},
	}
	for i, tc := range tests {
		minter.AnnualProvisions = sdk.NewDec(tc.annualProvisions)
		provisions := minter.BlockProvision(params)

		expProvisions := sdk.NewCoin(params.MintDenom,
			sdk.NewInt(tc.expProvisions))

		require.True(t, expProvisions.IsEqual(provisions),
			"test: %v\n\tExp: %v\n\tGot: %v\n",
			i, tc.expProvisions, provisions)
	}
}

// Benchmarking :)
// previously using sdk.Int operations:
// BenchmarkBlockProvision-4 5000000 220 ns/op
//
// using sdk.Dec operations: (current implementation)
// BenchmarkBlockProvision-4 3000000 429 ns/op
func BenchmarkBlockProvision(b *testing.B) {
	b.ReportAllocs()
	minter := InitialMinter(sdk.NewDecWithPrec(1, 1))
	params := DefaultParams()

	s1 := rand.NewSource(100)
	r1 := rand.New(s1)
	minter.AnnualProvisions = sdk.NewDec(r1.Int63n(1000000))

	// run the BlockProvision function b.N times
	for n := 0; n < b.N; n++ {
		minter.BlockProvision(params)
	}
}

// Next inflation benchmarking
// BenchmarkPhaseInflation-4 1000000 1828 ns/op
func BenchmarkPhaseInflation(b *testing.B) {
	b.ReportAllocs()
	minter := InitialMinter(sdk.NewDecWithPrec(1, 1))
	phase := uint64(4)

	// run the PhaseInflationRate function b.N times
	for n := 0; n < b.N; n++ {
		minter.PhaseInflationRate(phase)
	}
}

// Next annual provisions benchmarking
// BenchmarkNextAnnualProvisions-4 5000000 251 ns/op
func BenchmarkNextAnnualProvisions(b *testing.B) {
	b.ReportAllocs()
	minter := InitialMinter(sdk.NewDecWithPrec(1, 1))
	params := DefaultParams()
	totalSupply := sdk.NewInt(100000000000000)

	// run the NextAnnualProvisions function b.N times
	for n := 0; n < b.N; n++ {
		minter.NextAnnualProvisions(params, totalSupply)
	}
}
