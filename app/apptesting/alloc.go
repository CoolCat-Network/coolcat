package apptesting

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func (suite *KeeperTestHelper) TestDistribution() {
	suite.Setup()

	denom := suite.App.StakingKeeper.BondDenom(suite.Ctx)
	allocKeeper := suite.App.AllocKeeper
	params := suite.App.AllocKeeper.GetParams(suite.Ctx)
	params.DistributionProportions.CommunityPool = sdk.NewDecWithPrec(10, 2)

	suite.App.AllocKeeper.SetParams(suite.Ctx, params)

	feePool := suite.App.DistrKeeper.GetFeePool(suite.Ctx)
	feeCollector := suite.App.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName)
	suite.Equal(
		"0",
		suite.App.BankKeeper.GetAllBalances(suite.Ctx, feeCollector).AmountOf(denom).String())
	suite.Equal(
		sdk.NewDec(0),
		feePool.CommunityPool.AmountOf(denom))

	mintCoin := sdk.NewCoin(denom, sdk.NewInt(100_000))
	mintCoins := sdk.Coins{mintCoin}
	feeCollectorAccount := suite.App.AccountKeeper.GetModuleAccount(suite.Ctx, authtypes.FeeCollectorName)
	suite.Require().NotNil(feeCollectorAccount)

	suite.Require().NoError(suite.FundModuleAccount(suite.Ctx, feeCollectorAccount.GetName(), mintCoins))

	feeCollector = suite.App.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName)
	suite.Equal(
		mintCoin.Amount.String(),
		suite.App.BankKeeper.GetAllBalances(suite.Ctx, feeCollector).AmountOf(denom).String())

	suite.Equal(
		sdk.NewDec(0),
		feePool.CommunityPool.AmountOf(denom))

	err := allocKeeper.DistributeInflation(suite.Ctx)
	suite.Require().NoError(err)

	feeCollector = suite.App.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName)
	modulePortion := params.DistributionProportions.CommunityPool // 10%

	// remaining going to next module should be 100% - 10% = 90%
	suite.Equal(
		sdk.NewDec(mintCoin.Amount.Int64()).Mul(sdk.NewDecWithPrec(100, 2).Sub(modulePortion)).RoundInt().String(),
		suite.App.BankKeeper.GetAllBalances(suite.Ctx, feeCollector).AmountOf(denom).String())

	// since the NFT incentives are not setup yet, funds go into the communtiy pool
	feePool = suite.App.DistrKeeper.GetFeePool(suite.Ctx)
	suite.Equal(
		sdk.NewDec(mintCoin.Amount.Int64()).Mul(params.DistributionProportions.CommunityPool),
		feePool.CommunityPool.AmountOf(denom))
}