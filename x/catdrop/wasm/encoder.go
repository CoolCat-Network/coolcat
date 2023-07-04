package wasm

import (
	"encoding/json"
	"fmt"

	catdroptypes "github.com/coolcat-network/coolcat/v1/x/catdrop/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type ClaimAction string

const (
	ClaimActionProfile = "create_profile"
	ClaimActionClowder  = "create_join_clowder"
)

type ClaimFor struct {
	Address string      `json:"address"`
	Action  ClaimAction `json:"action"`
}

func (a ClaimAction) ToAction() (catdroptypes.Action, error) {
	if a == ClaimActionProfile {
		return catdroptypes.ActionCreateProfile, nil
	}

	if a == ClaimActionClowder {
		return catdroptypes.ActionUseClowder, nil
	}

	return 0, fmt.Errorf("invalid action")
}

type ClaimMsg struct {
	ClaimFor *ClaimFor `json:"claim_for,omitempty"`
}

func (c ClaimFor) Encode(contract sdk.AccAddress) ([]sdk.Msg, error) {
	action, err := c.Action.ToAction()
	if err != nil {
		return nil, err
	}
	msg := catdroptypes.NewMsgClaimFor(contract.String(), c.Address, action)
	return []sdk.Msg{msg}, nil
}

func Encoder(contract sdk.AccAddress, data json.RawMessage, version string) ([]sdk.Msg, error) {
	msg := &ClaimMsg{}
	err := json.Unmarshal(data, msg)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}
	if msg.ClaimFor != nil {
		return msg.ClaimFor.Encode(contract)
	}
	return nil, fmt.Errorf("wasm: invalid custom claim message")
}
