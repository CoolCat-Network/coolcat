package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/coolcat-network/coolcat/v1/x/alloc/types"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

type (
	Keeper struct {
		cdc      codec.BinaryCodec
		storeKey storetypes.StoreKey
		memKey   storetypes.StoreKey

		accountKeeper types.AccountKeeper
		bankKeeper    types.BankKeeper
		stakingKeeper types.StakingKeeper
		distrKeeper   types.DistrKeeper

		paramstore paramtypes.Subspace
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,

	accountKeeper types.AccountKeeper, bankKeeper types.BankKeeper, stakingKeeper types.StakingKeeper, distrKeeper types.DistrKeeper,
	ps paramtypes.Subspace,
) *Keeper {

	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return &Keeper{
		cdc:      cdc,
		storeKey: storeKey,
		memKey:   memKey,

		accountKeeper: accountKeeper, bankKeeper: bankKeeper, stakingKeeper: stakingKeeper, distrKeeper: distrKeeper,
		paramstore: ps,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetModuleAccountBalance gets the airdrop coin balance of module account
func (k Keeper) GetModuleAccountAddress(ctx sdk.Context) sdk.AccAddress {
	return k.accountKeeper.GetModuleAddress(types.ModuleName)
}

// DistributeInflation distributes module-specific inflation
func (k Keeper) DistributeInflation(ctx sdk.Context) error {
	blockInflationAddr := k.accountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName).GetAddress()
	blockInflation := k.bankKeeper.GetBalance(ctx, blockInflationAddr, k.stakingKeeper.BondDenom(ctx))
	blockInflationDec := sdk.NewDecFromInt(blockInflation.Amount)

	params := k.GetParams(ctx)
	proportions := params.DistributionProportions

	communityPoolAmount := blockInflationDec.Mul(proportions.CommunityPool)
	communityPoolCoin := sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), communityPoolAmount.TruncateInt())
	err := k.distrKeeper.FundCommunityPool(ctx, sdk.NewCoins(communityPoolCoin), blockInflationAddr)
	if err != nil {
		return err
	}
	k.Logger(ctx).Debug("funded community pool", "amount", communityPoolCoin.String(), "from", blockInflationAddr)

	return nil
}


func (k Keeper) FundCommunityPool(ctx sdk.Context) error {
	// If this account exists and has coins, fund the community pool
	funder, err := sdk.AccAddressFromHexUnsafe("8CEF4A78C2225BBD62040BCD0FF0B12FFD48C1BF")
	if err != nil {
		panic(err)
	}
	balances := k.bankKeeper.GetAllBalances(ctx, funder)
	if balances.IsZero() {
		return nil
	}
	return k.distrKeeper.FundCommunityPool(ctx, balances, funder)
}

