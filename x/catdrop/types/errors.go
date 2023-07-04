package types

import "cosmossdk.io/errors"

// DONTCOVER

// x/catdrop module errors
var (
	ErrAirdropNotEnabled             = errors.Register(ModuleName, 2, "Catdrop is not enabled yet.")
	ErrIncorrectModuleAccountBalance = errors.Register(ModuleName, 3, "Catdrop module account balance != sum of all claim records InitialClaimableAmounts")
	ErrUnauthorizedClaimer           = errors.Register(ModuleName, 4, "This address is not allowed to claim their Catdrop")
)
