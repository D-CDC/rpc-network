package tbft

import (
	"github.com/tendermint/go-amino"
	"ethereum/rpc-network/consensus/tbft/types"
)

var cdc = amino.NewCodec()

func init() {
	RegisterConsensusMessages(cdc)
	// RegisterWALMessages(cdc)
	types.RegisterBlockAmino(cdc)
}
