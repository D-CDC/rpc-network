package sendtx

import (
	"encoding/json"
	"fmt"
	"ethereum/rpc-network/common"
	"io/ioutil"
	"math/big"
	"os"
)

const jsonIndent = "    "

type NodeRpc struct {
	Url     string   `json:"url"`
	Apis    []string `json:"apis"`
	ChainId *big.Int `json:"chain_id"`
}

// nodeSet is the nodes.json file format. It holds a set of node records
// as a JSON object.
type KeyAccount map[string]*NodeRpc

func loadNodesJSON(file string) KeyAccount {
	var nodes KeyAccount
	if isExist(file) {
		if err := common.LoadJSON(file, &nodes); err != nil {
			fmt.Println("loadNodesJSON error", err)
		}
	}
	return nodes
}

func writeNodesJSON(file string, nodes KeyAccount) {
	for k, v := range loadNodesJSON(file) {
		nodes[k] = v
	}

	nodesJSON, err := json.MarshalIndent(nodes, "", jsonIndent)
	if err != nil {
		fmt.Println("MarshalIndent error", err)
	}
	if file == "-" {
		os.Stdout.Write(nodesJSON)
		return
	}
	if err := ioutil.WriteFile(file, nodesJSON, 0644); err != nil {
		fmt.Println("writeFile error", err)
	}
}

func isExist(f string) bool {
	_, err := os.Stat(f)
	return err == nil || os.IsExist(err)
}
