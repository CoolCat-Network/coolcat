package app

import (
	"encoding/json"

	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
)

var defaultGenesisBz []byte

func getDefaultGenesisStateBytes(appCodec codec.Codec) []byte {
	if len(defaultGenesisBz) == 0 {
		genesisState := NewDefaultGenesisState(appCodec)
		stateBytes, err := json.MarshalIndent(genesisState, "", " ")
		if err != nil {
			panic(err)
		}
		defaultGenesisBz = stateBytes
	}
	return defaultGenesisBz
}

// SetupWithCustomHome initializes a new OsmosisApp with a custom home directory
func SetupWithCustomHome(isCheckTx bool, dir string) *CoolCatApp {
	db := dbm.NewMemDB()
	app := NewCoolCatApp(log.NewNopLogger(), db, nil, true, map[int64]bool{}, dir, simtestutil.EmptyAppOptions{}, EmptyWasmOpts)
	if !isCheckTx {
		stateBytes := getDefaultGenesisStateBytes(app.appCodec)

		app.InitChain(
			abci.RequestInitChain{
				Validators:      []abci.ValidatorUpdate{},
				ConsensusParams: simtestutil.DefaultConsensusParams,
				AppStateBytes:   stateBytes,
			},
		)
	}

	return app
}

// Setup initializes a new CoolCatApp.
func Setup(isCheckTx bool) *CoolCatApp {
	return SetupWithCustomHome(isCheckTx, DefaultNodeHome)
}