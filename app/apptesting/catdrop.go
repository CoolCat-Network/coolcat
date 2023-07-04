package apptesting

import (
	"fmt"
	"time"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/coolcat-network/coolcat/v1/x/catdrop/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (s *KeeperTestHelper) TestExportGenesis() {
	app, ctx := s.App, s.Ctx
	app.CatdropKeeper.InitGenesis(ctx, *types.DefaultGenesis())
	// app.CatdropKeeper.SetParams(ctx, types.DefaultParams())
	exported := app.CatdropKeeper.ExportGenesis(ctx)
	params := types.DefaultParams()
	params.AirdropStartTime = ctx.BlockTime()
	s.Require().Equal(params.AllowedClaimers, exported.Params.AllowedClaimers)
}

func (suite *KeeperTestHelper) TestModuleAccountCreated() {
	app, ctx := suite.App, suite.Ctx
	moduleAddress := app.AccountKeeper.GetModuleAddress(types.ModuleName)
	balance := app.BankKeeper.GetBalance(ctx, moduleAddress, types.DefaultClaimDenom)
	suite.Require().Equal(fmt.Sprintf("10000000%s", types.DefaultClaimDenom), balance.String())
}

func (suite *KeeperTestHelper) TestClaimFor() {
	pub1 := secp256k1.GenPrivKey().PubKey()
	pub2 := secp256k1.GenPrivKey().PubKey()
	addr1 := sdk.AccAddress(pub1.Address())
	addr2 := sdk.AccAddress(pub2.Address())

	contractAddress1 := wasmkeeper.BuildContractAddressClassic(1, 1)
	contractAddress2 := wasmkeeper.BuildContractAddressClassic(1, 2)
	claimRecords := []types.ClaimRecord{
		{
			Address:                addr1.String(),
			InitialClaimableAmount: sdk.NewCoins(sdk.NewInt64Coin(types.DefaultClaimDenom, 2000)),
			ActionCompleted:        []bool{false, false, false, false},
		},
		{
			Address:                addr2.String(),
			InitialClaimableAmount: sdk.NewCoins(sdk.NewInt64Coin(types.DefaultClaimDenom, 2000)),
			ActionCompleted:        []bool{false, false, false, false},
		},
	}

	suite.App.AccountKeeper.SetAccount(suite.Ctx, authtypes.NewBaseAccount(addr1, nil, 0, 0))
	suite.App.AccountKeeper.SetAccount(suite.Ctx, authtypes.NewBaseAccount(addr2, nil, 0, 0))

	suite.App.CatdropKeeper.SetParams(suite.Ctx, types.Params{
		AirdropEnabled:     false,
		AirdropStartTime:   time.Now().Add(time.Hour * -1),
		ClaimDenom:         types.DefaultClaimDenom,
		DurationUntilDecay: time.Hour,
		DurationOfDecay:    time.Hour * 4,
		AllowedClaimers:    make([]types.ClaimAuthorization, 0),
	})
	err := suite.App.CatdropKeeper.SetClaimRecords(suite.Ctx, claimRecords)
	suite.Require().NoError(err)
	msgClaimFor := types.NewMsgClaimFor(contractAddress1.String(), addr1.String(), types.ActionCreateProfile)
	ctx := sdk.WrapSDKContext(suite.Ctx)
	_, err = suite.msgSrvr.ClaimFor(ctx, msgClaimFor)
	suite.Error(err)
	suite.Contains(err.Error(), "airdrop not enabled")
	suite.App.CatdropKeeper.SetParams(suite.Ctx, types.Params{
		AirdropEnabled:     true,
		AirdropStartTime:   time.Now().Add(time.Hour * -1),
		ClaimDenom:         types.DefaultClaimDenom,
		DurationUntilDecay: time.Hour,
		DurationOfDecay:    time.Hour * 4,
		AllowedClaimers: []types.ClaimAuthorization{
			{
				ContractAddress: contractAddress1.String(),
				Action:          types.ActionCreateProfile,
			},
			{
				ContractAddress: contractAddress2.String(),
				Action:          types.ActionUseClowder,
			},
		},
	})

	// unauthorized
	msgClaimFor = types.NewMsgClaimFor(wasmkeeper.BuildContractAddressClassic(2, 1).String(), addr1.String(), types.ActionCreateProfile)
	_, err = suite.msgSrvr.ClaimFor(ctx, msgClaimFor)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "address is not allowed to claim")

	// unauthorized to claim another action
	msgClaimFor = types.NewMsgClaimFor(contractAddress1.String(), addr1.String(), types.ActionCreateProfile)
	_, err = suite.msgSrvr.ClaimFor(ctx, msgClaimFor)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "address is not allowed to claim")

	// claim
	msgClaimFor = types.NewMsgClaimFor(contractAddress1.String(), addr1.String(), types.ActionCreateProfile)
	_, err = suite.msgSrvr.ClaimFor(ctx, msgClaimFor)
	suite.Require().NoError(err)

	claimedCoins := suite.App.BankKeeper.GetAllBalances(suite.Ctx, addr1)
	suite.Require().Equal(claimedCoins.AmountOf(types.DefaultClaimDenom), claimRecords[0].InitialClaimableAmount.AmountOf(types.DefaultClaimDenom).Quo(sdk.NewInt(4)))

	record, err := suite.App.CatdropKeeper.GetClaimRecord(suite.Ctx, addr1)
	suite.Require().NoError(err)
	suite.Require().True(record.ActionCompleted[1])
	suite.Require().Equal([]bool{false, true, false, false}, record.ActionCompleted)

	// claim 2
	msgClaimFor = types.NewMsgClaimFor(contractAddress2.String(), addr1.String(), types.ActionCreateProfile)
	_, err = suite.msgSrvr.ClaimFor(ctx, msgClaimFor)
	suite.Require().NoError(err)

	claimedCoins = suite.App.BankKeeper.GetAllBalances(suite.Ctx, addr1)
	suite.Require().Equal(
		claimedCoins.AmountOf(types.DefaultClaimDenom).String(),
		claimRecords[0].InitialClaimableAmount.AmountOf(types.DefaultClaimDenom).Quo(sdk.NewInt(4)).Mul(sdk.NewInt(2)).String(), // 2 actions claimed
	)

	record, err = suite.App.CatdropKeeper.GetClaimRecord(suite.Ctx, addr1)
	suite.Require().NoError(err)
	suite.Require().True(record.ActionCompleted[1])
	suite.Require().True(record.ActionCompleted[2])
	suite.Require().Equal([]bool{false, true, true, false}, record.ActionCompleted)

	// claim second address
	msgClaimFor = types.NewMsgClaimFor(contractAddress2.String(), addr2.String(), types.ActionCreateProfile)
	_, err = suite.msgSrvr.ClaimFor(ctx, msgClaimFor)
	suite.Require().NoError(err)

	claimedCoins = suite.App.BankKeeper.GetAllBalances(suite.Ctx, addr2)
	suite.Require().Equal(
		claimedCoins.AmountOf(types.DefaultClaimDenom).String(),
		claimRecords[0].InitialClaimableAmount.AmountOf(types.DefaultClaimDenom).Quo(sdk.NewInt(4)).String(), // 1 action claimed
	)

	record, err = suite.App.CatdropKeeper.GetClaimRecord(suite.Ctx, addr2)
	suite.Require().NoError(err)
	suite.Require().False(record.ActionCompleted[1])
	suite.Require().True(record.ActionCompleted[2])
	suite.Require().Equal([]bool{false, false, true, false}, record.ActionCompleted)
}

func (suite *KeeperTestHelper) TestHookOfUnclaimableAccount() {
	pub1 := secp256k1.GenPrivKey().PubKey()
	addr1 := sdk.AccAddress(pub1.Address())
	suite.App.AccountKeeper.SetAccount(suite.Ctx, authtypes.NewBaseAccount(addr1, nil, 0, 0))

	claim, err := suite.App.CatdropKeeper.GetClaimRecord(suite.Ctx, addr1)
	suite.NoError(err)
	suite.Equal(types.ClaimRecord{}, claim)

	suite.App.CatdropKeeper.AfterDelegationModified(suite.Ctx, addr1, sdk.ValAddress(addr1))

	balances := suite.App.BankKeeper.GetAllBalances(suite.Ctx, addr1)
	suite.Equal(sdk.Coins{}, balances)
}

func (suite *KeeperTestHelper) TestHookBeforeAirdropStart() {
	suite.Setup()

	airdropStartTime := time.Now().Add(time.Hour)

	suite.App.CatdropKeeper.SetParams(suite.Ctx, types.Params{
		AirdropEnabled:     true,
		ClaimDenom:         types.DefaultClaimDenom,
		AirdropStartTime:   airdropStartTime,
		DurationUntilDecay: time.Hour,
		DurationOfDecay:    time.Hour * 4,
	})

	pub1 := secp256k1.GenPrivKey().PubKey()
	addr1 := sdk.AccAddress(pub1.Address())

	claimRecords := []types.ClaimRecord{
		{
			Address:                addr1.String(),
			InitialClaimableAmount: sdk.NewCoins(sdk.NewInt64Coin(types.DefaultClaimDenom, 1000)),
			ActionCompleted:        []bool{false, false, false, false},
		},
	}
	suite.App.AccountKeeper.SetAccount(suite.Ctx, authtypes.NewBaseAccount(addr1, nil, 0, 0))

	err := suite.App.CatdropKeeper.SetClaimRecords(suite.Ctx, claimRecords)
	suite.Require().NoError(err)

	coins, err := suite.App.CatdropKeeper.GetUserTotalClaimable(suite.Ctx, addr1)
	suite.NoError(err)
	// Now, it is before starting air drop, so this value should return the empty coins
	suite.True(coins.Empty())

	coins, err = suite.App.CatdropKeeper.GetClaimableAmountForAction(suite.Ctx, addr1, types.ActionDelegateStake)
	suite.NoError(err)
	// Now, it is before starting air drop, so this value should return the empty coins
	suite.True(coins.Empty())

	suite.App.CatdropKeeper.AfterDelegationModified(suite.Ctx, addr1, sdk.ValAddress(addr1))
	balances := suite.App.BankKeeper.GetAllBalances(suite.Ctx, addr1)
	// Now, it is before starting air drop, so claim module should not send the balances to the user after delegate.
	suite.True(balances.Empty())

	suite.App.CatdropKeeper.AfterDelegationModified(suite.Ctx.WithBlockTime(airdropStartTime), addr1, sdk.ValAddress(addr1))
	balances = suite.App.BankKeeper.GetAllBalances(suite.Ctx, addr1)
	// Now, it is the time for air drop, so claim module should send the balances to the user after delegate.
	suite.Equal(claimRecords[0].InitialClaimableAmount.AmountOf(types.DefaultClaimDenom).Quo(sdk.NewInt(int64(len(types.Action_value)))), balances.AmountOf(types.DefaultClaimDenom))
}

func (suite *KeeperTestHelper) TestAirdropDisabled() {
	suite.Setup()

	airdropStartTime := time.Now().Add(time.Hour)

	suite.App.CatdropKeeper.SetParams(suite.Ctx, types.Params{
		AirdropEnabled:     false,
		ClaimDenom:         types.DefaultClaimDenom,
		DurationUntilDecay: time.Hour,
		DurationOfDecay:    time.Hour * 4,
	})

	pub1 := secp256k1.GenPrivKey().PubKey()
	addr1 := sdk.AccAddress(pub1.Address())

	claimRecords := []types.ClaimRecord{
		{
			Address:                addr1.String(),
			InitialClaimableAmount: sdk.NewCoins(sdk.NewInt64Coin(types.DefaultClaimDenom, 1000)),
			ActionCompleted:        []bool{false, false, false, false},
		},
	}
	suite.App.AccountKeeper.SetAccount(suite.Ctx, authtypes.NewBaseAccount(addr1, nil, 0, 0))

	err := suite.App.CatdropKeeper.SetClaimRecords(suite.Ctx, claimRecords)
	suite.Require().NoError(err)

	coins, err := suite.App.CatdropKeeper.GetUserTotalClaimable(suite.Ctx, addr1)
	suite.NoError(err)
	// Now, it is before starting air drop, so this value should return the empty coins
	suite.True(coins.Empty())

	coins, err = suite.App.CatdropKeeper.GetClaimableAmountForAction(suite.Ctx, addr1, types.ActionDelegateStake)
	suite.NoError(err)
	// Now, it is before starting air drop, so this value should return the empty coins
	suite.True(coins.Empty())

	suite.App.CatdropKeeper.AfterDelegationModified(suite.Ctx, addr1, sdk.ValAddress(addr1))
	balances := suite.App.BankKeeper.GetAllBalances(suite.Ctx, addr1)
	// Now, it is before starting air drop, so claim module should not send the balances to the user after delegate.
	suite.True(balances.Empty())

	suite.App.CatdropKeeper.AfterProposalVote(suite.Ctx, 1, addr1)
	balances = suite.App.BankKeeper.GetAllBalances(suite.Ctx, addr1)
	// Now, it is before starting air drop, so claim module should not send the balances to the user after vote.
	suite.True(balances.Empty())

	// set airdrop enabled but with invalid date
	suite.App.CatdropKeeper.SetParams(suite.Ctx, types.Params{
		AirdropEnabled:     true,
		ClaimDenom:         types.DefaultClaimDenom,
		DurationUntilDecay: time.Hour,
		DurationOfDecay:    time.Hour * 4,
	})

	suite.App.CatdropKeeper.AfterDelegationModified(suite.Ctx, addr1, sdk.ValAddress(addr1))
	balances = suite.App.BankKeeper.GetAllBalances(suite.Ctx, addr1)
	// Now airdrop is enabled but a potential misconfiguraion on start time
	suite.True(balances.Empty())

	suite.App.CatdropKeeper.AfterProposalVote(suite.Ctx, 1, addr1)
	balances = suite.App.BankKeeper.GetAllBalances(suite.Ctx, addr1)
	// Now airdrop is enabled but a potential misconfiguraion on start time, so claim module should not send the balances to the user after vote.
	suite.True(balances.Empty())

	// set airdrop enabled but with date in the future
	suite.App.CatdropKeeper.SetParams(suite.Ctx, types.Params{
		AirdropEnabled:     true,
		AirdropStartTime:   airdropStartTime.Add(time.Hour),
		ClaimDenom:         types.DefaultClaimDenom,
		DurationUntilDecay: time.Hour,
		DurationOfDecay:    time.Hour * 4,
	})

	suite.App.CatdropKeeper.AfterDelegationModified(suite.Ctx, addr1, sdk.ValAddress(addr1))
	balances = suite.App.BankKeeper.GetAllBalances(suite.Ctx, addr1)
	// Now airdrop is enabled  and date is not empty but block time still behid
	suite.True(balances.Empty())

	suite.App.CatdropKeeper.AfterProposalVote(suite.Ctx, 1, addr1)
	balances = suite.App.BankKeeper.GetAllBalances(suite.Ctx, addr1)
	// Now airdrop is enabled  and date is not empty but block time still behid
	suite.True(balances.Empty())

	// add extra 2 hours
	suite.App.CatdropKeeper.AfterDelegationModified(suite.Ctx.WithBlockTime(airdropStartTime.Add(time.Hour*2)), addr1, sdk.ValAddress(addr1))
	balances = suite.App.BankKeeper.GetAllBalances(suite.Ctx, addr1)
	// Now, it is the time for air drop, so claim module should send the balances to the user after delegate.
	suite.Equal(claimRecords[0].InitialClaimableAmount.AmountOf(types.DefaultClaimDenom).Quo(sdk.NewInt(int64(len(types.Action_value)))), balances.AmountOf(types.DefaultClaimDenom))
}

func (suite *KeeperTestHelper) TestDuplicatedActionNotWithdrawRepeatedly() {
	suite.Setup()

	pub1 := secp256k1.GenPrivKey().PubKey()
	addr1 := sdk.AccAddress(pub1.Address())

	claimRecords := []types.ClaimRecord{
		{
			Address:                addr1.String(),
			InitialClaimableAmount: sdk.NewCoins(sdk.NewInt64Coin(types.DefaultClaimDenom, 2000)),
			ActionCompleted:        []bool{false, false, false, false},
		},
	}
	suite.App.AccountKeeper.SetAccount(suite.Ctx, authtypes.NewBaseAccount(addr1, nil, 0, 0))

	err := suite.App.CatdropKeeper.SetClaimRecords(suite.Ctx, claimRecords)
	suite.Require().NoError(err)

	coins1, err := suite.App.CatdropKeeper.GetUserTotalClaimable(suite.Ctx, addr1)
	suite.Require().NoError(err)
	suite.Require().Equal(coins1, claimRecords[0].InitialClaimableAmount)

	suite.App.CatdropKeeper.AfterDelegationModified(suite.Ctx, addr1, sdk.ValAddress(addr1))
	claim, err := suite.App.CatdropKeeper.GetClaimRecord(suite.Ctx, addr1)
	suite.NoError(err)
	suite.True(claim.ActionCompleted[types.ActionDelegateStake])

	claimedCoins := suite.App.BankKeeper.GetAllBalances(suite.Ctx, addr1)
	suite.Require().Equal(claimedCoins.AmountOf(types.DefaultClaimDenom), claimRecords[0].InitialClaimableAmount.AmountOf(types.DefaultClaimDenom).Quo(sdk.NewInt(5)))

	suite.App.CatdropKeeper.AfterDelegationModified(suite.Ctx, addr1, sdk.ValAddress(addr1))
	claim, err = suite.App.CatdropKeeper.GetClaimRecord(suite.Ctx, addr1)
	suite.NoError(err)
	suite.True(claim.ActionCompleted[types.ActionDelegateStake])

	claimedCoins = suite.App.BankKeeper.GetAllBalances(suite.Ctx, addr1)
	suite.Require().Equal(claimedCoins.AmountOf(types.DefaultClaimDenom), claimRecords[0].InitialClaimableAmount.AmountOf(types.DefaultClaimDenom).Quo(sdk.NewInt(5)))
}

func (suite *KeeperTestHelper) TestNotRunningGenesisBlock() {
	suite.Ctx = suite.Ctx.WithBlockHeight(1)
	suite.App.CatdropKeeper.CreateModuleAccount(suite.Ctx, sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10000000)))
	// set airdrop enabled but with date in the future
	suite.App.CatdropKeeper.SetParams(suite.Ctx, types.Params{
		AirdropEnabled:     true,
		AirdropStartTime:   time.Now().Add(time.Hour * -1),
		ClaimDenom:         sdk.DefaultBondDenom,
		DurationUntilDecay: time.Hour,
		DurationOfDecay:    time.Hour * 4,
		AllowedClaimers:    make([]types.ClaimAuthorization, 0),
	})

	pub1 := secp256k1.GenPrivKey().PubKey()
	addr1 := sdk.AccAddress(pub1.Address())

	claimRecords := []types.ClaimRecord{
		{
			Address:                addr1.String(),
			InitialClaimableAmount: sdk.NewCoins(sdk.NewInt64Coin(types.DefaultClaimDenom, 2000)),
			ActionCompleted:        []bool{false, false, false, false},
		},
	}
	suite.App.AccountKeeper.SetAccount(suite.Ctx, authtypes.NewBaseAccount(addr1, nil, 0, 0))

	err := suite.App.CatdropKeeper.SetClaimRecords(suite.Ctx, claimRecords)
	suite.Require().NoError(err)

	coins1, err := suite.App.CatdropKeeper.GetUserTotalClaimable(suite.Ctx, addr1)
	suite.Require().NoError(err)
	suite.Require().Equal(coins1, claimRecords[0].InitialClaimableAmount)

	suite.App.CatdropKeeper.AfterDelegationModified(suite.Ctx, addr1, sdk.ValAddress(addr1))
	claim, err := suite.App.CatdropKeeper.GetClaimRecord(suite.Ctx, addr1)
	suite.NoError(err)
	suite.False(claim.ActionCompleted[types.ActionDelegateStake])

	coins1, err = suite.App.CatdropKeeper.GetUserTotalClaimable(suite.Ctx, addr1)
	suite.Require().NoError(err)
	suite.Require().Equal(coins1, claimRecords[0].InitialClaimableAmount)
}

func (suite *KeeperTestHelper) TestDelegationAutoWithdrawAndDelegateMore() {
	suite.Setup()
	suite.App.CatdropKeeper.CreateModuleAccount(suite.Ctx, sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10000000)))
	// set airdrop enabled but with date in the future
	suite.App.CatdropKeeper.SetParams(suite.Ctx, types.Params{
		AirdropEnabled:     true,
		AirdropStartTime:   time.Now().Add(time.Hour * -1),
		ClaimDenom:         sdk.DefaultBondDenom,
		DurationUntilDecay: time.Hour,
		DurationOfDecay:    time.Hour * 4,
		AllowedClaimers:    make([]types.ClaimAuthorization, 0),
	})

	pub1 := secp256k1.GenPrivKey().PubKey()
	pub2 := secp256k1.GenPrivKey().PubKey()
	addr1 := sdk.AccAddress(pub1.Address())
	addr2 := sdk.AccAddress(pub2.Address())

	claimRecords := []types.ClaimRecord{
		{
			Address:                addr1.String(),
			InitialClaimableAmount: sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1000)),
			ActionCompleted:        []bool{false, false, false, false},
		},
		{
			Address:                addr2.String(),
			InitialClaimableAmount: sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1000)),
			ActionCompleted:        []bool{false, false, false, false},
		},
	}

	suite.App.AccountKeeper.SetAccount(suite.Ctx, authtypes.NewBaseAccount(addr1, nil, 0, 0))
	suite.App.AccountKeeper.SetAccount(suite.Ctx, authtypes.NewBaseAccount(addr2, nil, 0, 0))

	err := suite.App.CatdropKeeper.SetClaimRecords(suite.Ctx, claimRecords)
	suite.Require().NoError(err)

	cr, err := suite.App.CatdropKeeper.GetClaimRecord(suite.Ctx, addr1)
	suite.Require().NoError(err)
	suite.Require().Equal(cr, claimRecords[0])
	coins1, err := suite.App.CatdropKeeper.GetUserTotalClaimable(suite.Ctx, addr1)
	suite.Require().NoError(err)
	suite.Require().Equal(claimRecords[1].InitialClaimableAmount.String(), coins1.String())

	coins2, err := suite.App.CatdropKeeper.GetUserTotalClaimable(suite.Ctx, addr2)
	suite.Require().NoError(err)
	suite.Require().Equal(coins2, claimRecords[1].InitialClaimableAmount)

	// addr1 becomes validator
	validator, err := stakingtypes.NewValidator(sdk.ValAddress(addr1), pub1, stakingtypes.Description{})
	suite.Require().NoError(err)
	validator = stakingkeeper.TestingUpdateValidator(suite.App.StakingKeeper, suite.Ctx, validator, true)

	validator, _ = validator.AddTokensFromDel(sdk.TokensFromConsensusPower(1, sdk.DefaultPowerReduction))
	delAmount := sdk.TokensFromConsensusPower(1, sdk.DefaultPowerReduction)
	err = suite.FundAccount(suite.Ctx, addr2, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, delAmount)))
	suite.NoError(err)

	balance := suite.App.BankKeeper.GetAllBalances(suite.Ctx, addr2)
	suite.Require().Equal(
		balance.AmountOf(sdk.DefaultBondDenom).String(),
		delAmount.String())

	_, err = suite.App.StakingKeeper.Delegate(suite.Ctx, addr2, delAmount, stakingtypes.Unbonded, validator, true)
	suite.NoError(err)

	// delegation should automatically call claim and withdraw balance
	claimedCoins := suite.App.BankKeeper.GetAllBalances(suite.Ctx, addr2)
	suite.Require().Equal(
		claimedCoins.AmountOf(sdk.DefaultBondDenom).String(),
		claimRecords[1].InitialClaimableAmount.AmountOf(sdk.DefaultBondDenom).Quo(sdk.NewInt(int64(len(claimRecords[1].ActionCompleted)))).String())

	_, err = suite.App.StakingKeeper.Delegate(suite.Ctx, addr2, claimedCoins.AmountOf(sdk.DefaultBondDenom), stakingtypes.Unbonded, validator, true)
	suite.NoError(err)
}

func (suite *KeeperTestHelper) TestEndAirdrop() {
	// set airdrop enabled but with date in the future
	suite.App.CatdropKeeper.SetParams(suite.Ctx, types.Params{
		AirdropEnabled:     true,
		AirdropStartTime:   time.Now().Add(time.Hour * -1),
		ClaimDenom:         types.DefaultClaimDenom,
		DurationUntilDecay: time.Hour,
		DurationOfDecay:    time.Hour * 4,
		AllowedClaimers:    make([]types.ClaimAuthorization, 0),
	})

	pub1 := secp256k1.GenPrivKey().PubKey()
	pub2 := secp256k1.GenPrivKey().PubKey()
	addr1 := sdk.AccAddress(pub1.Address())
	addr2 := sdk.AccAddress(pub2.Address())

	claimRecords := []types.ClaimRecord{
		{
			Address:                addr1.String(),
			InitialClaimableAmount: sdk.NewCoins(sdk.NewInt64Coin(types.DefaultClaimDenom, 1000)),
			ActionCompleted:        []bool{false, false, false, false},
		},
		{
			Address:                addr2.String(),
			InitialClaimableAmount: sdk.NewCoins(sdk.NewInt64Coin(types.DefaultClaimDenom, 1000)),
			ActionCompleted:        []bool{false, false, false, false},
		},
	}

	suite.App.AccountKeeper.SetAccount(suite.Ctx, authtypes.NewBaseAccount(addr1, nil, 0, 0))
	suite.App.AccountKeeper.SetAccount(suite.Ctx, authtypes.NewBaseAccount(addr2, nil, 0, 0))

	err := suite.App.CatdropKeeper.SetClaimRecords(suite.Ctx, claimRecords)
	suite.Require().NoError(err)

	err = suite.App.CatdropKeeper.EndAirdrop(suite.Ctx)
	suite.Require().NoError(err)

	moduleAccAddr := suite.App.AccountKeeper.GetModuleAddress(types.ModuleName)
	coins := suite.App.BankKeeper.GetBalance(suite.Ctx, moduleAccAddr, types.DefaultClaimDenom)
	suite.Require().Equal(sdk.NewInt64Coin(types.DefaultClaimDenom, 0).String(), coins.String())
}

