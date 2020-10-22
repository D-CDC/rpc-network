package tbft

import (
	"fmt"
	types2 "ethereum/rpc-network/consensus/tbft/types"
	"ethereum/rpc-network/core/types"
	"math/big"
	"testing"
	"time"
)

func RandHexBytes20() [20]byte {
	var b [20]byte
	for i := 0; i < 20; i++ {
		b[i] = RandUint() % 255
	}
	return b
}

func TestReader(t *testing.T) {
	header := &types.Header{
		Time:        uint64(time.Now().Unix()),
		ParentHash:  RandHexBytes32(),
		Root:        RandHexBytes32(),
		TxHash:      RandHexBytes32(),
		ReceiptHash: RandHexBytes32(),
		Extra:       nil,
	}

	var tr []*types.Transaction

	for i := 0; i < 1000; i++ {
		t := types.NewTransaction(1, RandHexBytes20(), big.NewInt(1), 8888, big.NewInt(1), nil)
		tr = append(tr, t)
	}

	var re []*types.Receipt
	var si []*types.PbftSign

	bTmp := types.NewBlock(header, tr, re, si, nil)

	ps, _ := types2.MakePartSet(64*1024, bTmp)
	pe := types2.NewPartSetFromHeader(ps.Header())

	for i := 0; i < int(ps.Total()); i++ {
		pe.AddPart(ps.GetPart(uint(i)))
	}

	header2 := &types.Header{}
	n, e := cdc.UnmarshalBinaryReader(pe.GetReader(), &header2, 37502)
	fmt.Println(n, e)
	fmt.Println(header)
	fmt.Println(header2)

}
