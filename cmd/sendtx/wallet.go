package sendtx

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
)

// rawWallet is a JSON representation of an accounts.Wallet interface, with its
// data contents extracted into plain fields.
type rawWallet struct {
	URL      string             `json:"url"`
	Status   string             `json:"status"`
	Failure  string             `json:"failure,omitempty"`
	Accounts []accounts.Account `json:"accounts,omitempty"`
}

type rawAccount struct {
	Status   string             `json:"status"`
	Accounts string `json:"accounts,omitempty"`
}

type SendTxArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *hexutil.Uint64 `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	Nonce    *hexutil.Uint64 `json:"nonce"`
	// We accept "data" and "input" for backwards-compatibility reasons. "input" is the
	// newer name and should be preferred by clients.
	Data  *hexutil.Bytes `json:"data"`
	Input *hexutil.Bytes `json:"input"`
}

func sendRawTransaction(client *rpc.Client, from, to common.Address, value *big.Int) (common.Hash, error) {

	mapData := make(map[string]interface{})
	mapData["from"] = from.String()
	mapData["to"] = to.String()
	mapData["value"] = hexutil.Encode(value.Bytes())
	var result common.Hash
	err := client.Call(&result, "eth_sendTransaction", mapData)
	return result, err
}