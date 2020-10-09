package sendtx

import (
	"context"
	"ethereum/rpc-network/params"
	"ethereum/rpc-network/rpc"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"testing"
)

func TestBalanceAt(t *testing.T) {
	ctx := context.Background()
	ip := "188.166.0.226"
	//ip := "127.0.0.1"
	url := "http://" + ip + ":8545"
	client, err := rpc.DialContext(context.Background(), url)
	if err != nil {
		fmt.Printf("Failed to connect to Ethereum node: %v \n", err)
		return
	}
	var result hexutil.Big
	err = client.CallContext(ctx, &result, "eth_chainId")
	if err != nil {
		fmt.Println("eth_chainId", err)
		return
	}
	str, err := client.SupportedModules()
	fmt.Println("str", str, "chainid", result.ToInt())

	if result.ToInt().Cmp(params.MainnetChainConfig.ChainID) != 0 {
		netId, _ := NetworkID(client)
		fmt.Println("netId", netId, "enode", url)
	}

	sendMoreTx(client)
	//QueryDetail(client,url,str,result.ToInt())
}
