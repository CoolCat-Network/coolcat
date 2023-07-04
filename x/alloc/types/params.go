package types

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter store keys
var (
	KeyDistributionProportions  = []byte("DistributionProportions")
)

// ParamTable for module.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

func NewParams(
	distrProportions DistributionProportions,
) Params {

	return Params{
		DistributionProportions: distrProportions,
	}
}

// default module parameters
func DefaultParams() Params {
	return Params{
		DistributionProportions: DistributionProportions{
			CommunityPool: sdk.NewDecWithPrec(10, 2), // 10%
		},
	}
}

// validate params
func (p Params) Validate() error {
	err := validateDistributionProportions(p.DistributionProportions)
		return err
}

// Implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyDistributionProportions, &p.DistributionProportions, validateDistributionProportions),
	}
}

func validateDistributionProportions(i interface{}) error {
	v, ok := i.(DistributionProportions)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.CommunityPool.IsNegative() {
		return errors.New("community pool distribution ratio should not be negative")
	}

	totalProportions := v.CommunityPool

	// 10% is allocated to this module
	// 90% validators for staking rewards
	if !totalProportions.Equal(sdk.NewDecWithPrec(10, 2)) {
		return errors.New("total distributions ratio should be 10%")
	}

	return nil
}
