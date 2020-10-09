## RPC NetWork

### Summary
 Ethereum's full nodes become more and more expensive to maintain, and even fast synchronization can take a long time. Many contract developers deploy contracts only through Infura, but it has a limited number of times and requires registration, which is cumbersome. In Ethereum 2.0, each beacon chain node needs to connect 1.0 nodes. If it is too complicated for the verifier to build the node by yourself, it needs to build 1.0 nodes, beacon chain nodes and verify the human node. I want to get a batch of RPC nodes through P2P network. The availability of RPC node data can be verified by light les protocol, which can save some costs。

### Principle

use ethereum devp2p node discovery v4, if peers < 8, local node will find nodes. when i get many enode, i will use rpc.dial(ip+port) try connect. if connect ,i will store it. according chainid and networkid find useful nodes，use les protocol getbalance or other method verify rpc nodes result.

## Building the source

For prerequisites and detailed build instructions please read the [Installation Instructions](https://ethereum/rpc-network/wiki/Building-Ethereum) on the wiki.

Building `geth` requires both a Go (version 1.13 or later) and a C compiler. You can install
them using your favourite package manager. Once the dependencies are installed, run

```shell
make geth
```

or, to build the full suite of utilities:

```shell
make all
```

### MainNet Node
```cassandraql
url http://95.217.87.230:8545 apis map[eth:1.0 personal:1.0 rpc:1.0 web3:1.0]
url http://47.75.169.53:8545 apis map[admin:1.0 eth:1.0 net:1.0 personal:1.0 rpc:1.0 web3:1.0]
url http://3.15.200.203:8545 apis map[eth:1.0 net:1.0 personal:1.0 rpc:1.0 web3:1.0]
url http://95.217.204.233:8545 apis map[admin:1.0 eth:1.0 miner:1.0 net:1.0 personal:1.0 rpc:1.0 txpool:1.0 web3:1.0]
url http://93.115.29.78:8545 apis map[admin:1.0 debug:1.0 eth:1.0 ethash:1.0 net:1.0 personal:1.0 rpc:1.0 txpool:1.0 web3:1.0]
url http://46.4.123.142:8545 apis map[eth:1.0 net:1.0 personal:1.0 rpc:1.0 web3:1.0]
url http://151.106.32.9:8546 apis map[eth:1.0 net:1.0 personal:1.0 rpc:1.0 web3:1.0]
url http://8.210.84.25:8545 apis map[eth:1.0 net:1.0 personal:1.0 rpc:1.0 web3:1.0]
url http://138.201.255.176:8545 apis map[admin:1.0 eth:1.0 miner:1.0 personal:1.0 rpc:1.0 web3:1.0]
url http://45.77.65.188:8545 apis map[eth:1.0 net:1.0 personal:1.0 rpc:1.0 web3:1.0]
url http://193.227.103.78:8545 apis map[eth:1.0 net:1.0 personal:1.0 rpc:1.0 web3:1.0]
url http://167.99.192.187:8545 apis map[admin:1.0 debug:1.0 eth:1.0 ethash:1.0 net:1.0 personal:1.0 rpc:1.0 txpool:1.0 web3:1.0]
url http://47.52.73.207:8545 apis map[eth:1.0 net:1.0 personal:1.0 rpc:1.0 web3:1.0]
url http://118.107.46.13:8545 apis map[eth:1.0 personal:1.0 rpc:1.0 web3:1.0]
url http://182.61.190.153:8545 apis map[eth:1.0 personal:1.0 rpc:1.0 web3:1.0]
url http://173.199.127.126:8545 apis map[eth:1.0 net:1.0 rpc:1.0 web3:1.0]
url http://149.202.77.80:8545 apis map[debug:1.0 eth:1.0 net:1.0 parity:1.0 parity_accounts:1.0 parity_pubsub:1.0 parity_set:1.0 parity_transactions_pool:1.0 personal:1.0 private:1.0 pubsub:1.0 rpc:1.0 secretstore:1.0 signer:1.0 traces:1.0 web3:1.0]
url http://139.59.158.0:8545 apis map[eth:1.0 net:1.0 parity:1.0 personal:1.0 pubsub:1.0 rpc:1.0 shh:1.0 signer:1.0 traces:1.0 web3:1.0]
http://202.182.110.190:8545 apis map[eth:1.0 net:1.0 rpc:1.0 web3:1.0]
```

### Ropsten Node
```cassandraql
url http://161.35.205.60:8545 apis map[debug:1.0 eth:1.0 net:1.0 parity:1.0 parity_accounts:1.0 parity_pubsub:1.0 parity_set:1.0 parity_transactions_pool:1.0 personal:1.0 private:1.0 pubsub:1.0 rpc:1.0 secretstore:1.0 signer:1.0 traces:1.0 web3:1.0]
url http://95.217.143.211:8545 apis map[admin:1.0 eth:1.0 net:1.0 personal:1.0 rpc:1.0 txpool:1.0 web3:1.0]
url http://94.130.206.254:8545 apis map[debug:1.0 eth:1.0 net:1.0 personal:1.0 rpc:1.0 web3:1.0]
url http://45.204.3.103:8545 apis map[eth:1.0 net:1.0 personal:1.0 rpc:1.0 web3:1.0] arr 
url http://136.243.145.71:8545 apis map[eth:1.0 net:1.0 personal:1.0 rpc:1.0 web3:1.0]
url http://195.201.203.182:8545 apis map[admin:1.0 eth:1.0 net:1.0 personal:1.0 rpc:1.0 web3:1.0]
url http://128.199.150.233:8545 apis map[debug:1.0 eth:1.0 net:1.0 rpc:1.0 txpool:1.0 web3:1.0]
url http://88.99.214.131:8545 apis map[eth:1.0 net:1.0 rpc:1.0 web3:1.0]
```
### Goerli Node
```cassandraql
url http://157.245.118.249:8545 apis map[admin:1.0 debug:1.0 eth:1.0 net:1.0 personal:1.0 rpc:1.0 txpool:1.0 web3:1.0]
url http://23.106.254.77:8545 apis map[admin:1.0 debug:1.0 eth:1.0 net:1.0 personal:1.0 rpc:1.0 web3:1.0]
```

### Balance Node
```
miner 16917.707789 chainId 1 url http://188.166.0.226:8545 apis map[eth:1.0 net:1.0 rpc:1.0 web3:1.0] arr [] my 0.000000 0x4C549990A7eF3FEA8784406c1EECc98bF4211fA5
```
