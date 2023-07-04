package keeper

import (
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/coolcat-network/coolcat/v1/x/catdrop/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, data types.GenesisState) []abci.ValidatorUpdate {
	k.CreateModuleAccount(ctx, data.ModuleAccountBalance)
	if data.Params.AirdropEnabled && data.Params.AirdropStartTime.Equal(time.Time{}) {
		data.Params.AirdropStartTime = ctx.BlockTime()
	}
	err := k.SetClaimRecords(ctx, data.ClaimRecords)
	if err != nil {
		panic(err)
	}
	k.SetParams(ctx, data.Params)
	return nil
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	genesis := types.DefaultGenesis()

	params := k.GetParams(ctx)
	genesis.ModuleAccountBalance = k.GetModuleAccountBalance(ctx)
	genesis.Params = params
	genesis.ClaimRecords = k.ClaimRecords(ctx)
	return genesis
}
