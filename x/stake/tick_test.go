package stake

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var r = rand.New(rand.NewSource(502))

func TestGetInflation(t *testing.T) {
	ctx, _, keeper := createTestInput(t, false, 0)
	pool := keeper.GetPool(ctx)
	params := keeper.GetParams(ctx)

	// Governing Mechanism:
	//    bondedRatio = BondedPool / TotalSupply
	//    inflationRateChangePerYear = (1- bondedRatio/ GoalBonded) * MaxInflationRateChange

	tests := []struct {
		name                          string
		setBondedPool, setTotalSupply int64
		setInflation, expectedChange  sdk.Rat
	}{
		// with 0% bonded atom supply the inflation should increase by InflationRateChange
		{"test 1", 0, 0, sdk.NewRat(7, 100), params.InflationRateChange.Quo(hrsPerYrRat).Round(precision)},

		// 100% bonded, starting at 20% inflation and being reduced
		// (1 - (1/0.67))*(0.13/8667)
		{"test 2", 1, 1, sdk.NewRat(20, 100),
			sdk.OneRat().Sub(sdk.OneRat().Quo(params.GoalBonded)).Mul(params.InflationRateChange).Quo(hrsPerYrRat).Round(precision)},

		// 50% bonded, starting at 10% inflation and being increased
		{"test 3", 1, 2, sdk.NewRat(10, 100),
			sdk.OneRat().Sub(sdk.NewRat(1, 2).Quo(params.GoalBonded)).Mul(params.InflationRateChange).Quo(hrsPerYrRat).Round(precision)},

		// test 7% minimum stop (testing with 100% bonded)
		{"test 4", 1, 1, sdk.NewRat(7, 100), sdk.ZeroRat()},
		{"test 5", 1, 1, sdk.NewRat(70001, 1000000), sdk.NewRat(-1, 1000000).Round(precision)},

		// test 20% maximum stop (testing with 0% bonded)
		{"test 6", 0, 0, sdk.NewRat(20, 100), sdk.ZeroRat()},
		{"test 7", 0, 0, sdk.NewRat(199999, 1000000), sdk.NewRat(1, 1000000).Round(precision)},

		// perfect balance shouldn't change inflation
		{"test 8", 67, 100, sdk.NewRat(15, 100), sdk.ZeroRat()},
	}
	for _, tc := range tests {
		pool.BondedPool, pool.TotalSupply = tc.setBondedPool, tc.setTotalSupply
		pool.Inflation = tc.setInflation
		keeper.setPool(ctx, pool)

		inflation := keeper.nextInflation(ctx)
		diffInflation := inflation.Sub(tc.setInflation)

		assert.True(t, diffInflation.Equal(tc.expectedChange),
			"Name: %v\nDiff:  %v\nExpected: %v\n", tc.name, diffInflation, tc.expectedChange)
	}
}

func TestProcessProvisions(t *testing.T) {
	ctx, _, keeper := createTestInput(t, false, 0)
	params := defaultParams()
	keeper.setParams(ctx, params)
	pool := keeper.GetPool(ctx)

	// create some candidates some bonded, some unbonded
	pool = setupCandidates(pool, keeper, ctx, 10, 0, 5)

	// test setUpCandidates returned the token values by passing these vars into checkCandidateSetup()
	var (
		initialTotalTokens    int64 = 550000000
		initialBondedTokens   int64 = 150000000
		initialUnbondedTokens int64 = 400000000
		cumulativeExpProvs    int64
		bondedShares          = sdk.NewRat(150000000, 1)
		unbondedShares        = sdk.NewRat(400000000, 1)
	)
	checkCandidateSetup(t, pool, initialTotalTokens, initialBondedTokens, initialUnbondedTokens)

	// process the provisions a year
	for hr := 0; hr < 8766; hr++ {
		pool := keeper.GetPool(ctx)
		_, expProvisions, _ := checkAndProcessProvisions(t, keeper, pool, ctx, hr)
		cumulativeExpProvs = cumulativeExpProvs + expProvisions
	}

	// Final check that the pool equals initial values + provisions and adjustments we recorded
	pool = keeper.GetPool(ctx)
	checkFinalPoolValues(t, pool, initialTotalTokens,
		initialUnbondedTokens, cumulativeExpProvs,
		0, 0, bondedShares, unbondedShares)

}

//Tests that the hourly rate of change will be positve, negative, or zero, depending on bonded ratio and inflation rate
func TestHourlyRateOfChange(t *testing.T) {
	ctx, _, keeper := createTestInput(t, false, 0)
	params := defaultParams()
	keeper.setParams(ctx, params)
	pool := keeper.GetPool(ctx)

	// create some candidates some bonded, some unbonded
	pool = setupCandidates(pool, keeper, ctx, 10, 0, 5)

	// test setUpCandidates returned the token values by passing these vars into checkCandidateSetup()
	var (
		initialTotalTokens    int64 = 550000000
		initialBondedTokens   int64 = 150000000
		initialUnbondedTokens int64 = 400000000
		cumulativeExpProvs    int64
		bondedShares          = sdk.NewRat(150000000, 1)
		unbondedShares        = sdk.NewRat(400000000, 1)
	)
	checkCandidateSetup(t, pool, initialTotalTokens, initialBondedTokens, initialUnbondedTokens)

	// ~11.4 years to go from 7%, up to 20%, back down to 7%
	for hr := 0; hr < 100000; hr++ {
		pool := keeper.GetPool(ctx)

		previousInflation := pool.Inflation
		expInflation, expProvisions, pool := checkAndProcessProvisions(t, keeper, pool, ctx, hr)
		cumulativeExpProvs = cumulativeExpProvs + expProvisions

		updatedInflation := pool.Inflation
		inflationChange := updatedInflation.Sub(previousInflation)

		//Rate of change positive and increasing, while we are between 7% and 20% inflation
		if pool.bondedRatio().LT(sdk.NewRat(67, 100)) && expInflation.LT(sdk.NewRat(20, 100)) {
			assert.Equal(t, true, inflationChange.GT(sdk.ZeroRat()), strconv.Itoa(hr))
		}

		//Rate of change should be 0 while it holds at 20% a year, until we reach 67%
		if pool.bondedRatio().LT(sdk.NewRat(67, 100)) && expInflation.Equal(sdk.NewRat(20, 100)) {
			if previousInflation.Equal(sdk.NewRat(20, 100)) {
				assert.Equal(t, true, inflationChange.IsZero(), strconv.Itoa(hr))
				//This else covers the one off case where we first hit 20%, but we still needed a positive ROC to get there
			} else {
				assert.Equal(t, true, inflationChange.GT(sdk.ZeroRat()), strconv.Itoa(hr))
			}
		}

		//Rate of change should be negative while the bond is above 67, and should stay negative until we reach inflation of 7%
		if pool.bondedRatio().GT(sdk.NewRat(67, 100)) && expInflation.LT(sdk.NewRat(20, 100)) && expInflation.GT(sdk.NewRat(7, 100)) {
			assert.Equal(t, true, inflationChange.LT(sdk.ZeroRat()), strconv.Itoa(hr))
		}

		//Rate of change should be 0 while we hold at 7%.
		if pool.bondedRatio().GT(sdk.NewRat(67, 100)) && expInflation.Equal(sdk.NewRat(7, 100)) {
			if previousInflation.Equal(sdk.NewRat(7, 100)) {
				assert.Equal(t, true, inflationChange.IsZero(), strconv.Itoa(hr))
				//This else covers the one off case where we first hit 7%, but we still needed a negative ROC to get there
			} else {
				assert.Equal(t, true, inflationChange.LT(sdk.ZeroRat()), strconv.Itoa(hr))
			}
		}
	}

	// Final check that the pool equals initial values + provisions and adjustments we recorded
	pool = keeper.GetPool(ctx)
	checkFinalPoolValues(t, pool, initialTotalTokens,
		initialUnbondedTokens, cumulativeExpProvs,
		0, 0, bondedShares, unbondedShares)
}

//Test that a large unbonding will significantly lower the bonded ratio
func TestLargeUnbond(t *testing.T) {
	ctx, _, keeper := createTestInput(t, false, 0)
	params := defaultParams()
	keeper.setParams(ctx, params)
	pool := keeper.GetPool(ctx)

	// Candidates unbonded (0-4), bonded (5-9),
	// candidate 9 will be unbonded, bringing us from ~73% to ~55%
	pool = setupCandidates(pool, keeper, ctx, 10, 5, 10)

	// test setUpCandidates returned the token values by passing these vars into checkCandidateSetup()
	var (
		initialTotalTokens    int64 = 550000000
		initialBondedTokens   int64 = 400000000
		initialUnbondedTokens int64 = 150000000
		cand9UnbondedTokens   int64
		bondedShares          = sdk.NewRat(400000000, 1)
		unbondedShares        = sdk.NewRat(150000000, 1)
		bondSharesCand9       = sdk.NewRat(100000000, 1)
	)
	checkCandidateSetup(t, pool, initialTotalTokens, initialBondedTokens, initialUnbondedTokens)

	pool = keeper.GetPool(ctx)
	candidate, found := keeper.GetCandidate(ctx, addrs[9])
	assert.True(t, found)

	// before values that we can use to compare to the new values after the unbond
	beforeBondedRatio := pool.bondedRatio()
	expInflationBefore := keeper.nextInflation(ctx)

	// This func will unbond 100,000,000 tokens that were previously bonded
	pool, candidate, _, _ = OpBondOrUnbond(r, pool, candidate)
	keeper.setPool(ctx, pool)

	// process provisions after the bonding, to compare the difference in expProvisions and expInflation
	expInflationAfter, expProvisionsAfter, pool := checkAndProcessProvisions(t, keeper, pool, ctx, 0)

	bondedShares = bondedShares.Sub(bondSharesCand9)
	cand9UnbondedTokens = pool.unbondedShareExRate().Mul(candidate.Assets).Evaluate()
	unbondedShares = unbondedShares.Add(sdk.NewRat(cand9UnbondedTokens, 1).Mul(pool.unbondedShareExRate()))

	// inflation should be lower than before. Because we unbonded, we are further from 67%, and expInlfationAfter is higher
	assert.True(t, expInflationBefore.LT(expInflationAfter))
	// unbonded shares should increase
	assert.True(t, unbondedShares.GT(sdk.NewRat(150000000, 1)))
	// Ensure that new bonded ratio is less than old bonded ratio , because before they were increasing (i.e. 55 < 72)
	assert.True(t, (pool.bondedRatio().LT(beforeBondedRatio)))

	// Final check that the pool equals initial values + provisions and adjustments we recorded
	pool = keeper.GetPool(ctx)
	checkFinalPoolValues(t, pool, initialTotalTokens,
		initialUnbondedTokens, expProvisionsAfter,
		-cand9UnbondedTokens, cand9UnbondedTokens, bondedShares, unbondedShares)

}

//Test that a large bonding will cause inflation to go down, and lower the bondedRatio
func TestLargeBond(t *testing.T) {
	ctx, _, keeper := createTestInput(t, false, 0)
	params := defaultParams()
	keeper.setParams(ctx, params)
	pool := keeper.GetPool(ctx)

	// Candidates unbonded (0-4), bonded (5-8), candidate 9 left unbonded, so it can be bonded later in the test
	// bonded candidate 9 brings us from ~55% to ~73 bondedRatio
	pool = setupCandidates(pool, keeper, ctx, 10, 5, 9)

	// test setUpCandidates returned the token values by passing these vars into checkCandidateSetup()
	var (
		initialTotalTokens    int64 = 550000000
		initialBondedTokens   int64 = 300000000
		initialUnbondedTokens int64 = 250000000
		cand9unbondedTokens   int64 = 100000000
		cand9bondedTokens     int64
		bondedShares          = sdk.NewRat(300000000, 1)
		unbondedShares        = sdk.NewRat(250000000, 1)
		unbondSharesCand9     = sdk.NewRat(100000000, 1)
	)
	checkCandidateSetup(t, pool, initialTotalTokens, initialBondedTokens, initialUnbondedTokens)

	pool = keeper.GetPool(ctx)
	candidate, found := keeper.GetCandidate(ctx, addrs[9])
	assert.True(t, found)

	// before values that we can use to compare to the new values after the Bond
	beforeBondedRatio := pool.bondedRatio()
	expInflationBefore := keeper.nextInflation(ctx)

	// This func will bond 100,000,000 tokens that were previously unbonded
	pool, candidate, _, _ = OpBondOrUnbond(r, pool, candidate)
	keeper.setPool(ctx, pool)

	// process provisions after the bonding, to compare the difference in expProvisions and expInflation
	expInflationAfter, expProvisionsAfter, pool := checkAndProcessProvisions(t, keeper, pool, ctx, 0)

	unbondedShares = unbondedShares.Sub(unbondSharesCand9)
	cand9bondedTokens = cand9unbondedTokens
	cand9unbondedTokens = 0
	bondedTokens := initialBondedTokens + cand9bondedTokens + expProvisionsAfter
	bondedShares = sdk.NewRat(bondedTokens, 1).Quo(pool.bondedShareExRate())

	// inflation should be larger before. Because we bonded, we are closer to 67%, and expInlfationAfter is lower
	assert.True(t, expInflationBefore.GT(expInflationAfter))
	// bonded shares should increase
	assert.True(t, bondedShares.GT(sdk.NewRat(300000000, 1)))
	//Ensure that new bonded ratio is greater than old bonded ratio, since we just added 100,000 bonded
	assert.True(t, (pool.bondedRatio().GT(beforeBondedRatio)))

	// Final check that the pool equals initial values + provisions and adjustments we recorded
	checkFinalPoolValues(t, pool, initialTotalTokens,
		initialUnbondedTokens, expProvisionsAfter,
		cand9bondedTokens, -cand9bondedTokens, bondedShares, unbondedShares)

}

//Tests that inflation works as expected when we get a randomly updating sample of candidates
func TestInflationWithRandomOperations(t *testing.T) {
	ctx, _, keeper := createTestInput(t, false, 0)
	params := defaultParams()
	keeper.setParams(ctx, params)
	numCandidates := 20 //max 40 right now since var addrs only goes up to addrs[39]

	//start off by randomly creating 20 candidates
	pool, candidates := randomSetup(r, numCandidates)
	require.Equal(t, numCandidates, len(candidates))

	for i := 0; i < len(candidates); i++ {
		keeper.setCandidate(ctx, candidates[i])
	}

	keeper.setPool(ctx, pool)

	//Every two weeks (336 hours) we do a random operation on the set of 20 candidates.
	twoWeekCounter := 336

	//We count up to the 20 total candidates, and do a random operation on each candidate
	candidateCounter := 0

	// One year of provisions, with 20 random operations from the 20 candidates setup above from randomSetup()
	for hr := 0; hr < 8766; hr++ {
		pool := keeper.GetPool(ctx)

		// This if statement will randomly bond, unbond, remove shares or add tokens to the candidates every two weeks, for 40 weeks
		// every other hour it will just add normal provisions
		if twoWeekCounter == hr && candidateCounter < 20 {

			//Get values before randomOperation
			expInflationBefore := keeper.nextInflation(ctx)
			initialBondedPool := pool.BondedPool
			initialUnbondedTokens := pool.UnbondedPool

			//Random operation, and recording how candidates are modified
			poolMod, candidateMod, tokens, msg := randomOperation(r)(r, pool, candidates[candidateCounter])
			candidatesMod := make([]Candidate, len(candidates))
			copy(candidatesMod[:], candidates[:])
			require.Equal(t, numCandidates, len(candidates), "i %v", candidateCounter)
			require.Equal(t, numCandidates, len(candidatesMod), "i %v", candidateCounter)
			candidatesMod[candidateCounter] = candidateMod

			assertInvariants(t, msg,
				pool, candidates,
				poolMod, candidatesMod, tokens)

			//set pool and candidates after the random operation
			pool = poolMod
			keeper.setPool(ctx, pool)
			candidates = candidatesMod

			//Get values after randomOperation
			expInflationAfter := keeper.nextInflation(ctx)
			afterBondedPool := pool.BondedPool
			afterUnbondedPool := pool.UnbondedPool

			//Process provisions after random operation
			pool = keeper.processProvisions(ctx)
			keeper.setPool(ctx, pool)

			//Check the inflation has changed as expected, based on the difference between tokens before and after the operation
			checkInflation(t, expInflationAfter, expInflationBefore, msg,
				afterBondedPool, initialBondedPool, afterUnbondedPool, initialUnbondedTokens)

			twoWeekCounter += 336
			candidateCounter++

		} else {

			// If we are not doing a random operation, just check that normal provisions are working for each hour
			checkAndProcessProvisions(t, keeper, pool, ctx, hr)

		}
	}
}

// Final check on the global pool values against what each test added up hour by hour.
// bondedAdjustment and unbondedAdjustment are the calculated changes
// that the test calling this function accumlated (i.e. if three unbonds happened, their total value passed as unbondedAdjustment)
func checkFinalPoolValues(t *testing.T, pool Pool, initialTotalTokens, initialUnbondedTokens,
	cumulativeExpProvs, bondedAdjustment, unbondedAdjustment int64, bondedShares, unbondedShares sdk.Rat) {

	initialBonded := initialTotalTokens - initialUnbondedTokens
	calculatedTotalTokens := initialTotalTokens + cumulativeExpProvs
	calculatedBondedTokens := initialBonded + cumulativeExpProvs + bondedAdjustment
	calculatedUnbondedTokens := initialUnbondedTokens + unbondedAdjustment

	// test that the bonded ratio the pool has is equal to what we calculated for tokens
	assert.True(t, pool.bondedRatio().Equal(sdk.NewRat(calculatedBondedTokens, calculatedTotalTokens)), "%v", pool.bondedRatio())

	// test global supply
	assert.Equal(t, calculatedTotalTokens, pool.TotalSupply)
	assert.Equal(t, calculatedBondedTokens, pool.BondedPool)
	assert.Equal(t, calculatedUnbondedTokens, pool.UnbondedPool)

	// test the value of candidate shares
	assert.True(t, pool.bondedShareExRate().Mul(bondedShares).Equal(sdk.NewRat(calculatedBondedTokens)), "%v", pool.bondedShareExRate())
	assert.True(t, pool.unbondedShareExRate().Mul(unbondedShares).Equal(sdk.NewRat(calculatedUnbondedTokens)), "%v", pool.unbondedShareExRate())
}

// Checks provisions are added to the pool correctly every hour
// Returns expected Provisions, expected Inflation, and pool, to help with cumulative calculations back in main Tests
func checkAndProcessProvisions(t *testing.T, keeper Keeper, pool Pool, ctx sdk.Context, hr int) (sdk.Rat, int64, Pool) {

	//If we are not doing a random operation, just check that normal provisions are working for each hour
	expInflation := keeper.nextInflation(ctx)
	expProvisions := (expInflation.Mul(sdk.NewRat(pool.TotalSupply)).Quo(hrsPerYrRat)).Evaluate()
	// expPRoTemp := (expInflation.Mul(sdk.NewRat(pool.TotalSupply)).Quo(hrsPerYrRat))
	startBondedPool := pool.BondedPool
	startTotalSupply := pool.TotalSupply
	pool = keeper.processProvisions(ctx)
	keeper.setPool(ctx, pool)
	// fmt.Println("expprotemp: ", expPRoTemp)

	//check provisions were added to pool
	require.Equal(t, startBondedPool+expProvisions, pool.BondedPool, "hr %v", hr)
	require.Equal(t, startTotalSupply+expProvisions, pool.TotalSupply)

	return expInflation, expProvisions, pool
}

// Deterministic setup of candidates
// Allows you to decide how many candidates to setup, and which ones you want bonded
// Tokens allocated to each candidate increase by 10000000 for each candidate
func setupCandidates(pool Pool, keeper Keeper, ctx sdk.Context, numCands, indexBondedGT, indexBondedLT int) Pool {

	candidates := make([]Candidate, numCands)
	for i := 0; i < numCands; i++ {
		c := Candidate{
			Status:      Unbonded,
			PubKey:      pks[i],
			Address:     addrs[i],
			Assets:      sdk.NewRat(0),
			Liabilities: sdk.NewRat(0),
		}
		if i >= indexBondedGT && i < indexBondedLT {
			c.Status = Bonded
		}
		mintedTokens := int64((i + 1) * 10000000)
		pool.TotalSupply += mintedTokens
		pool, c, _ = pool.candidateAddTokens(c, mintedTokens)

		keeper.setCandidate(ctx, c)
		candidates[i] = c
	}
	keeper.setPool(ctx, pool)
	pool = keeper.GetPool(ctx)
	return pool
}

// Checks that the deterministic candidate setup you wanted matches the values in the pool
func checkCandidateSetup(t *testing.T, pool Pool, initialTotalTokens, initialBondedTokens, initialUnbondedTokens int64) {

	assert.Equal(t, initialTotalTokens, pool.TotalSupply)
	assert.Equal(t, initialBondedTokens, pool.BondedPool)
	assert.Equal(t, initialUnbondedTokens, pool.UnbondedPool)

	// test initial bonded ratio
	assert.True(t, pool.bondedRatio().Equal(sdk.NewRat(initialBondedTokens, initialTotalTokens)), "%v", pool.bondedRatio())
	// test the value of candidate shares
	assert.True(t, pool.bondedShareExRate().Equal(sdk.OneRat()), "%v", pool.bondedShareExRate())
}

// Pass this function expected inflations and bonded and unbonded token amount, both before and after an operation
// The function verifies that the yearly inflation changes as expected, based on how the pools tokens have changed
func checkInflation(t *testing.T, expInflationAfter, expInflationBefore sdk.Rat, msg string,
	afterBondedPool, initialBondedPool, afterUnbondedPool, initialUnbondedTokens int64) {

	//Comparing expected inflation before and after the randomOperation()
	inflationIncreased := expInflationAfter.GT(expInflationBefore)
	inflationEqual := expInflationAfter.Equal(expInflationBefore)

	switch {
	//Inflation will NOT increase, because we are adding bonded tokens to the pool
	//CASE: Happens when we randomly add tokens to a bonded candidate
	//CASE: Happens when we randomly bond a candidate from unbonded
	case afterBondedPool > initialBondedPool:

		//for the off case where we are bonded so low (i.e. 15%) , that we add bonded tokens and inflation is unchanged at 20%
		//for the off case where we are bonded so high (i.e. 90%), that we add bonded tokens and inflation is unchanged at 7%
		if expInflationBefore.Equal(sdk.NewRat(20, 100)) {
			assert.True(t, !inflationIncreased || inflationEqual, msg)
		} else if expInflationBefore.Equal(sdk.NewRat(7, 100)) {
			assert.True(t, !inflationIncreased || inflationEqual, msg)
		} else {
			assert.True(t, !inflationIncreased, msg)
		}

	//Inflation WILL increase, because we are removing bonded tokens from the pool
	//CASE: Happens when we randomly remove bonded Shares
	//CASE: Happens when we randomly unbond a candidate that was bonded
	case afterBondedPool < initialBondedPool:

		//For the off case where we are bonded so low (i.e. 15%),  and more tokens are unbonded, inflation is unchanged at 20%
		//For the off case where we are bonded so high (i.e. 90%) and we unbond a small amount (maybe 2%), inflation  is unchanged at 7%
		if expInflationBefore.Equal(sdk.NewRat(20, 100)) {
			assert.True(t, inflationIncreased || inflationEqual, msg)
		} else if expInflationBefore.Equal(sdk.NewRat(7, 100)) {
			assert.True(t, inflationIncreased || inflationEqual, msg)
		} else {
			assert.True(t, inflationIncreased, msg)
		}

	//Inflation will STAY THE SAME.
	//Inflation is dependant only on bondedRatio. Sure, a validator can add unbonded tokens, but it doesn't change bondedRatio (totalBondedTokens / globalTotalTokens)
	//CASE: Happens when we randomly add unbonded tokens
	case afterUnbondedPool > initialUnbondedTokens:
		assert.True(t, inflationEqual, msg)

	//Inflation will STAY THE SAME.
	//Inflation is dependant only on bondedRatio. Sure, a validator can add unbonded tokens, but it doesn't change bondedRatio (totalBondedTokens / globalTotalTokens)
	//CASE: Happens when we randomly remove shares from an unbonded candidate
	case afterUnbondedPool < initialUnbondedTokens:
		assert.True(t, inflationEqual, msg)

	default:
		panic(fmt.Sprintf("pool.UnbondedPool and pool.BondedPool are unchanged. All operations calling this function should change either the unbondedPool or bondedPool amounts."))
	}

}
