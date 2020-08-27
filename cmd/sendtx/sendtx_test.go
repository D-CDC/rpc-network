package sendtx

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"testing"
)

func TestBalanceAt(t *testing.T) {
	ctx := context.Background()
	ip := "142.93.227.185"
	//ip := "127.0.0.1"
	url := "http://"+ ip+":8545"
	client, err := rpc.DialContext(context.Background(), url)
	if err != nil {
		fmt.Printf("Failed to connect to Ethereum node: %v \n", err)
		return
	}
	var result hexutil.Big
	err = client.CallContext(ctx, &result, "eth_chainId")
	if err != nil {
		fmt.Println("eth_chainId",err)
		return
	}
	str,err := client.SupportedModules()
	fmt.Println("str",str,"chainid",result.ToInt())

	if result.ToInt().Cmp(params.MainnetChainConfig.ChainID) != 0 {
		netId,_ := NetworkID(client)
		fmt.Println("netId",netId,"enode",url)
	}

	sendMoreTx(client)
	//QueryDetail(client,url,str,result.ToInt())
}

