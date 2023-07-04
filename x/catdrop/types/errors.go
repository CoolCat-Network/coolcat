package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/catdrop module errors
var (
	ErrAirdropNotEnabled             = sdkerrors.Register(ModuleName, 2, "Catdrop is not enabled yet.")
	ErrIncorrectModuleAccountBalance = sdkerrors.Register(ModuleName, 3, "Catdrop module account balance != sum of all claim records InitialClaimableAmounts")
	ErrUnauthorizedClaimer           = sdkerrors.Register(ModuleName, 4, "This address is not allowed to claim their Catdrop")
)
