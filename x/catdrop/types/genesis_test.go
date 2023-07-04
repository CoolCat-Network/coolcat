package types_test

import (
	"testing"
	"time"

	"github.com/coolcat-network/coolcat/v1/x/catdrop/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state",
			genState: &types.GenesisState{
				ModuleAccountBalance: sdk.NewCoin(sdk.DefaultBondDenom, sdk.ZeroInt()),
				Params: types.Params{
					AirdropEnabled:     true,
					AirdropStartTime:   time.Time{},
					DurationUntilDecay: time.Hour * 24 * 60,
					DurationOfDecay:    time.Hour * 24 * 30 * 4,
					ClaimDenom:         sdk.DefaultBondDenom,
				},
				ClaimRecords: []types.ClaimRecord{},
			},
			valid: true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
