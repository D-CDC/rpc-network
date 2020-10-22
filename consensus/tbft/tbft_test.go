package tbft

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"ethereum/rpc-network/crypto"
	"github.com/ethereum/go-ethereum/log"
	tcrypto "ethereum/rpc-network/consensus/tbft/crypto"
	"ethereum/rpc-network/consensus/tbft/help"
	ttypes "ethereum/rpc-network/consensus/tbft/types"
	"ethereum/rpc-network/core/types"
	config "ethereum/rpc-network/params"
)

type PbftAgentProxyImp struct {
	Name string
}

func NewPbftAgent(name string) *PbftAgentProxyImp {
	pap := PbftAgentProxyImp{Name: name}
	return &pap
}

var ID = big.NewInt(1)

func getID() *big.Int {
	ID = new(big.Int).Add(ID, big.NewInt(1))
	return ID
}

var IDCache = make(map[string]*big.Int)

func IDCacheInit() {
	lock = new(sync.Mutex)

	IDCache["Agent1"] = big.NewInt(1)
	IDCache["Agent2"] = big.NewInt(1)
	IDCache["Agent3"] = big.NewInt(1)
	IDCache["Agent4"] = big.NewInt(1)
	IDCache["Agent5"] = big.NewInt(1)
	IDCache["Agent5"] = Tbft5Start
}

var lock *sync.Mutex

func getIDForCache(agent string) *big.Int {
	lock.Lock()
	defer lock.Unlock()
	return IDCache[agent]
}

func IDAdd(agent string) {
	lock.Lock()
	defer lock.Unlock()
	tmp := new(big.Int).Set(IDCache[agent])
	IDCache[agent] = new(big.Int).Add(tmp, big.NewInt(1))
	if agent == "Agent3" {
		IDCache["Agent4"] = new(big.Int).Add(tmp, big.NewInt(1))
		IDCache["Agent5"] = new(big.Int).Add(tmp, big.NewInt(1))
	}
}

func (pap *PbftAgentProxyImp) FetchFastBlock(committeeID *big.Int, infos []*types.CommitteeMember) (*types.Block, error) {
	header := new(types.Header)
	header.Number = getIDForCache(pap.Name) //getID()
	fmt.Println(pap.Name, header.Number)
	header.Time = big.NewInt(time.Now().Unix())
	println("[AGENT]", pap.Name, "++++++++", "FetchFastBlock", "Number:", header.Number.Uint64())
	return types.NewBlock(header, nil, nil, nil, infos), nil
}

func (agent *PbftAgentProxyImp) GetFastLastProposer() common.Address {
	return getAddr()
}

func (pap *PbftAgentProxyImp) GetCurrentHeight() *big.Int {
	//return big.NewInt(0)
	return new(big.Int).Sub(getIDForCache(pap.Name), common.Big1)
}

func (pap *PbftAgentProxyImp) GetSeedMember() []*types.CommitteeMember {
	return nil
}

func (pap *PbftAgentProxyImp) GenerateSignWithVote(fb *types.Block, vote uint32) (*types.PbftSign, error) {
	voteSign := &types.PbftSign{
		Result:     vote,
		FastHeight: fb.Header().Number,
		FastHash:   fb.Hash(),
	}
	if vote == types.VoteAgreeAgainst {
		log.Warn("vote AgreeAgainst", "number", fb.Number(), "hash", fb.Hash())
	}
	var err error
	signHash := voteSign.HashWithNoSign().Bytes()

	s := strings.Replace(pap.Name, "Agent", "", -1)
	num, e := strconv.Atoi(s)
	if e != nil || num == 1 {
		num = 0
	} else {
		num--
	}
	pr1 := getPrivateKey(num)
	voteSign.Sign, err = crypto.Sign(signHash, pr1)
	if err != nil {
		log.Error("fb GenerateSign error ", "err", err)
	}
	return voteSign, err
}

func (pap *PbftAgentProxyImp) VerifyFastBlock(block *types.Block, sign bool) (*types.PbftSign, error) {
	//if rand.Intn(100) > 30 {
	//	return types.ErrHeightNotYet
	//}
	println("[AGENT]", pap.Name, "VerifyFastBlock", "Number:", block.Header().Number.Uint64())

	return pap.GenerateSignWithVote(block, 1)
}

var BcCount = 0

func (pap *PbftAgentProxyImp) BroadcastConsensus(block *types.Block) error {
	IDAdd(pap.Name)
	println("[AGENT]", pap.Name, "--------", "BroadcastConsensus", "Number:", block.Header().Number.Uint64())
	return nil
}

var comm = make(map[int][]byte)

func InitComm() {
	comm[0] = []byte("c1581e25937d9ab91421a3e1a2667c85b0397c75a195e643109938e987acecfc")
	comm[1] = []byte("42c4d734786eedaf5d0c51fd2bc9bbaa6c289ed23710d9381072932456aeca18")
	comm[2] = []byte("d878614f687c663baf7cdcbc32cc0fc88a036cdc6850023d880b03984426a629")
	comm[3] = []byte("26981a9479b7c4d98c546451c13a78b53c695df14c1968a086219edfe60bce2f")
	comm[4] = []byte("36981a9479b7c4d98c546451c13a78b53c695df14c1968a086219edfe60bce6f")
}

func getPrivateKey(id int) *ecdsa.PrivateKey {
	if len(comm) == 0 {
		InitComm()
	}
	key, err := hex.DecodeString(string(comm[id]))
	if err != nil {
		fmt.Println(err)
	}
	priv, err := crypto.ToECDSA(key)
	if err != nil {
		fmt.Println(err.Error())
	}
	return priv
}

func GetPub(priv *ecdsa.PrivateKey) []byte {
	pub := ecdsa.PublicKey{
		Curve: priv.Curve,
		X:     new(big.Int).Set(priv.X),
		Y:     new(big.Int).Set(priv.Y),
	}
	return crypto.FromECDSAPub(&pub)
}

func GetPubKey(priv *ecdsa.PrivateKey) *ecdsa.PublicKey {
	pub := ecdsa.PublicKey{
		Curve: priv.Curve,
		X:     new(big.Int).Set(priv.X),
		Y:     new(big.Int).Set(priv.Y),
	}
	return &pub
}

func TestPbftRunForOne(t *testing.T) {
	//log.OpenLogDebug(4)
	IDCacheInit()
	start := make(chan int)
	pr := getPrivateKey(0)
	agent1 := NewPbftAgent("Agent1")
	n, _ := NewNode(config.DefaultConfig(), "1", pr, agent1)
	n.Start()
	c1 := new(types.CommitteeInfo)
	c1.Id = big.NewInt(1)
	m1 := new(types.CommitteeMember)
	m1.Publickey = GetPub(pr)
	c1.Members = append(c1.Members, m1)
	c1.StartHeight = common.Big0
	n.PutCommittee(c1)
	n.Notify(c1.Id, Start)
	go CloseStart(start)
	<-start
}

func TestPbftRunFor2(t *testing.T) {
	//log.OpenLogDebug(3)
	IDCacheInit()
	start := make(chan int)
	pr1 := getPrivateKey(0)
	pr2 := getPrivateKey(1)

	agent1 := NewPbftAgent("Agent1")
	agent2 := NewPbftAgent("Agent2")

	config1 := new(config.TbftConfig)
	*config1 = *config.TestConfig()
	p2p1 := new(config.P2PConfig)
	*p2p1 = *config1.P2P
	p2p1.ListenAddress1 = "tcp://127.0.0.1:28890"
	p2p1.ListenAddress2 = "tcp://127.0.0.1:28891"
	*config1.P2P = *p2p1

	con1 := new(config.ConsensusConfig)
	*con1 = *config1.Consensus
	con1.WalPath = filepath.Join("data", "cs.wal1", "wal")
	*config1.Consensus = *con1

	n1, _ := NewNode(config1, "1", pr1, agent1)
	n1.Start()

	config2 := new(config.TbftConfig)
	*config2 = *config.TestConfig()
	p2p2 := new(config.P2PConfig)
	*p2p2 = *config2.P2P
	p2p2.ListenAddress1 = "tcp://127.0.0.1:28893"
	p2p2.ListenAddress2 = "tcp://127.0.0.1:28894"
	*config2.P2P = *p2p2

	con2 := new(config.ConsensusConfig)
	*con2 = *config2.Consensus
	con2.WalPath = filepath.Join("data", "cs.wal2", "wal")
	*config2.Consensus = *con2

	n2, _ := NewNode(config2, "1", pr2, agent2)
	n2.Start()

	c1 := new(types.CommitteeInfo)
	c1.Id = big.NewInt(1)
	m1 := new(types.CommitteeMember)
	m1.Publickey = GetPub(pr1)
	m1.Coinbase = common.Address{0}
	m2 := new(types.CommitteeMember)
	m2.Publickey = GetPub(pr2)
	m2.Coinbase = common.Address{0}

	c1.Members = append(c1.Members, m1, m2)
	c1.StartHeight = common.Big1

	cn := make([]*types.CommitteeNode, 0)
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28890, Coinbase: m1.Coinbase, Publickey: m1.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28893, Coinbase: m2.Coinbase, Publickey: m2.Publickey})

	n1.PutCommittee(c1)
	n1.PutNodes(common.Big1, cn)
	n1.Notify(c1.Id, Start)

	n2.PutCommittee(c1)
	n2.PutNodes(common.Big1, cn)
	n2.Notify(c1.Id, Start)
	go CloseStart(start)
	<-start
}

func TestPbftRunFor4(t *testing.T) {
	log.OpenLogDebug(3)
	IDCacheInit()
	start := make(chan int)
	pr1 := getPrivateKey(0)
	pr2 := getPrivateKey(1)
	pr3 := getPrivateKey(2)
	pr4 := getPrivateKey(3)

	agent1 := NewPbftAgent("Agent1")
	agent2 := NewPbftAgent("Agent2")
	agent3 := NewPbftAgent("Agent3")
	agent4 := NewPbftAgent("Agent4")

	config1 := new(config.TbftConfig)
	*config1 = *config.DefaultConfig()
	p2p1 := new(config.P2PConfig)
	*p2p1 = *config1.P2P
	p2p1.ListenAddress1 = "tcp://127.0.0.1:28890"
	p2p1.ListenAddress2 = "tcp://127.0.0.1:28891"
	*config1.P2P = *p2p1

	con1 := new(config.ConsensusConfig)
	*con1 = *config1.Consensus
	con1.WalPath = filepath.Join("data", "cs.wal1", "wal")
	*config1.Consensus = *con1

	n1, _ := NewNode(config1, "1", pr1, agent1)
	n1.Start()

	config2 := new(config.TbftConfig)
	*config2 = *config.DefaultConfig()
	p2p2 := new(config.P2PConfig)
	*p2p2 = *config2.P2P
	p2p2.ListenAddress1 = "tcp://127.0.0.1:28893"
	p2p2.ListenAddress2 = "tcp://127.0.0.1:28894"
	*config2.P2P = *p2p2

	con2 := new(config.ConsensusConfig)
	*con2 = *config2.Consensus
	con2.WalPath = filepath.Join("data", "cs.wal2", "wal")
	*config2.Consensus = *con2

	n2, _ := NewNode(config2, "1", pr2, agent2)
	n2.Start()

	config3 := new(config.TbftConfig)
	*config3 = *config.DefaultConfig()
	p2p3 := new(config.P2PConfig)
	*p2p3 = *config3.P2P
	p2p3.ListenAddress1 = "tcp://127.0.0.1:28895"
	p2p3.ListenAddress2 = "tcp://127.0.0.1:28896"
	*config3.P2P = *p2p3

	con3 := new(config.ConsensusConfig)
	*con3 = *config3.Consensus
	con3.WalPath = filepath.Join("data", "cs.wal3", "wal")
	*config3.Consensus = *con3

	n3, _ := NewNode(config3, "1", pr3, agent3)
	n3.Start()

	config4 := new(config.TbftConfig)
	*config4 = *config.DefaultConfig()
	p2p4 := new(config.P2PConfig)
	*p2p4 = *config4.P2P
	p2p4.ListenAddress1 = "tcp://127.0.0.1:28897"
	p2p4.ExternalAddress = "tcp://127.0.0.1:28898"
	*config4.P2P = *p2p4

	con4 := new(config.ConsensusConfig)
	*con4 = *config4.Consensus
	con4.WalPath = filepath.Join("data", "cs.wal4", "wal")
	*config4.Consensus = *con4

	n4, _ := NewNode(config4, "1", pr4, agent4)
	n4.Start()

	c1 := new(types.CommitteeInfo)
	c1.Id = big.NewInt(1)
	m1 := new(types.CommitteeMember)
	m1.Publickey = GetPub(pr1)
	m1.Coinbase = common.Address{0}
	m1.Flag = types.StateUsedFlag
	m1.MType = types.TypeFixed
	m1.CommitteeBase = common.BytesToAddress(crypto.Keccak256(m1.Publickey[1:])[12:])
	m2 := new(types.CommitteeMember)
	m2.Publickey = GetPub(pr2)
	m2.Coinbase = common.Address{0}
	m2.Flag = types.StateUsedFlag
	m2.MType = types.TypeFixed
	m2.CommitteeBase = common.BytesToAddress(crypto.Keccak256(m2.Publickey[1:])[12:])
	m3 := new(types.CommitteeMember)
	m3.Publickey = GetPub(pr3)
	m3.Coinbase = common.Address{0}
	m3.Flag = types.StateUsedFlag
	m3.MType = types.TypeFixed
	m3.CommitteeBase = common.BytesToAddress(crypto.Keccak256(m3.Publickey[1:])[12:])
	m4 := new(types.CommitteeMember)
	m4.Publickey = GetPub(pr4)
	m4.Coinbase = common.Address{0}
	m4.Flag = types.StateUsedFlag
	m4.MType = types.TypeFixed
	m4.CommitteeBase = common.BytesToAddress(crypto.Keccak256(m4.Publickey[1:])[12:])
	c1.Members = append(c1.Members, m1, m2, m3, m4)
	c1.StartHeight = common.Big0

	cn := make([]*types.CommitteeNode, 0)
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28890, Port2: 28891, Coinbase: m1.Coinbase, Publickey: m1.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28893, Port2: 28894, Coinbase: m2.Coinbase, Publickey: m2.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28895, Port2: 28896, Coinbase: m3.Coinbase, Publickey: m3.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28897, Port2: 28899, Coinbase: m4.Coinbase, Publickey: m4.Publickey})

	n1.PutCommittee(c1)
	n1.PutNodes(common.Big1, cn)
	n1.Notify(c1.Id, Start)

	n2.PutCommittee(c1)
	n2.PutNodes(common.Big1, cn)
	n2.Notify(c1.Id, Start)

	n3.PutCommittee(c1)
	n3.PutNodes(common.Big1, cn)
	n3.Notify(c1.Id, Start)

	n4.PutCommittee(c1)
	n4.PutNodes(common.Big1, cn)
	n4.Notify(c1.Id, Start)
	go CloseStart(start)
	<-start
}

func TestPbftRunFor4AndChange(t *testing.T) {
	//log.OpenLogDebug(3)
	IDCacheInit()
	start := make(chan int)
	pr1 := getPrivateKey(0)
	pr2 := getPrivateKey(1)
	pr3 := getPrivateKey(2)
	pr4 := getPrivateKey(3)

	agent1 := NewPbftAgent("Agent1")
	agent2 := NewPbftAgent("Agent2")
	agent3 := NewPbftAgent("Agent3")
	agent4 := NewPbftAgent("Agent4")

	config1 := new(config.TbftConfig)
	*config1 = *config.TestConfig()
	p2p1 := new(config.P2PConfig)
	*p2p1 = *config1.P2P
	p2p1.ListenAddress1 = "tcp://127.0.0.1:28890"
	p2p1.ListenAddress2 = "tcp://127.0.0.1:28891"
	*config1.P2P = *p2p1

	con1 := new(config.ConsensusConfig)
	*con1 = *config1.Consensus
	con1.WalPath = filepath.Join("data", "cs.wal1", "wal")
	*config1.Consensus = *con1

	n1, _ := NewNode(config1, "1", pr1, agent1)
	n1.Start()

	config2 := new(config.TbftConfig)
	*config2 = *config.TestConfig()
	p2p2 := new(config.P2PConfig)
	*p2p2 = *config2.P2P
	p2p2.ListenAddress1 = "tcp://127.0.0.1:28893"
	p2p2.ListenAddress2 = "tcp://127.0.0.1:28894"
	*config2.P2P = *p2p2

	con2 := new(config.ConsensusConfig)
	*con2 = *config2.Consensus
	con2.WalPath = filepath.Join("data", "cs.wal2", "wal")
	*config2.Consensus = *con2

	n2, _ := NewNode(config2, "1", pr2, agent2)
	n2.Start()

	config3 := new(config.TbftConfig)
	*config3 = *config.TestConfig()
	p2p3 := new(config.P2PConfig)
	*p2p3 = *config3.P2P
	p2p3.ListenAddress1 = "tcp://127.0.0.1:28895"
	p2p3.ListenAddress2 = "tcp://127.0.0.1:28896"
	*config3.P2P = *p2p3

	con3 := new(config.ConsensusConfig)
	*con3 = *config3.Consensus
	con3.WalPath = filepath.Join("data", "cs.wal3", "wal")
	*config3.Consensus = *con3

	n3, _ := NewNode(config3, "1", pr3, agent3)
	n3.Start()

	config4 := new(config.TbftConfig)
	*config4 = *config.TestConfig()
	p2p4 := new(config.P2PConfig)
	*p2p4 = *config4.P2P
	p2p4.ListenAddress1 = "tcp://127.0.0.1:28897"
	p2p4.ListenAddress2 = "tcp://127.0.0.1:28898"
	*config4.P2P = *p2p4

	con4 := new(config.ConsensusConfig)
	*con4 = *config4.Consensus
	con4.WalPath = filepath.Join("data", "cs.wal4", "wal")
	*config4.Consensus = *con4

	n4, _ := NewNode(config4, "1", pr4, agent4)
	n4.Start()

	c1 := new(types.CommitteeInfo)
	c1.Id = big.NewInt(1)
	m1 := new(types.CommitteeMember)
	m1.Publickey = GetPub(pr1)
	m1.Coinbase = common.Address{0}
	m2 := new(types.CommitteeMember)
	m2.Publickey = GetPub(pr2)
	m2.Coinbase = common.Address{0}
	m3 := new(types.CommitteeMember)
	m3.Publickey = GetPub(pr3)
	m3.Coinbase = common.Address{0}
	m4 := new(types.CommitteeMember)
	m4.Publickey = GetPub(pr4)
	m4.Coinbase = common.Address{0}
	c1.Members = append(c1.Members, m1, m2, m3, m4)
	c1.StartHeight = common.Big0

	cn := make([]*types.CommitteeNode, 0)
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28890, Port2: 28891, Coinbase: m1.Coinbase, Publickey: m1.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28893, Port2: 28894, Coinbase: m2.Coinbase, Publickey: m2.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28895, Port2: 28896, Coinbase: m3.Coinbase, Publickey: m3.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28897, Port2: 28899, Coinbase: m4.Coinbase, Publickey: m4.Publickey})

	n1.PutCommittee(c1)
	n1.PutNodes(common.Big1, cn)
	n1.Notify(c1.Id, Start)

	n2.PutCommittee(c1)
	n2.PutNodes(common.Big1, cn)
	n2.Notify(c1.Id, Start)

	n3.PutCommittee(c1)
	n3.PutNodes(common.Big1, cn)
	n3.Notify(c1.Id, Start)

	n4.PutCommittee(c1)
	n4.PutNodes(common.Big1, cn)
	n4.Notify(c1.Id, Start)

	n1.SetCommitteeStop(c1.Id, 16)
	n2.SetCommitteeStop(c1.Id, 16)
	n3.SetCommitteeStop(c1.Id, 16)
	n4.SetCommitteeStop(c1.Id, 16)

	c2 := *c1
	c2.Id = common.Big2
	c2.StartHeight = big.NewInt(17)

	n1.PutCommittee(&c2)
	n1.PutNodes(common.Big2, cn)
	n1.Notify(c1.Id, Stop)
	n1.Notify(c2.Id, Start)

	n2.PutCommittee(&c2)
	n2.PutNodes(common.Big2, cn)
	n2.Notify(c1.Id, Stop)
	n2.Notify(c2.Id, Start)

	n3.PutCommittee(&c2)
	n3.PutNodes(common.Big2, cn)
	n3.Notify(c1.Id, Stop)
	n3.Notify(c2.Id, Start)

	n4.PutCommittee(&c2)
	n4.PutNodes(common.Big2, cn)
	n4.Notify(c1.Id, Stop)
	n4.Notify(c2.Id, Start)
	go CloseStart(start)
	<-start
}

func TestPbftRunFor5(t *testing.T) {
	//log.OpenLogDebug(4)
	IDCacheInit()

	start := make(chan int)
	pr1 := getPrivateKey(0)
	pr2 := getPrivateKey(1)
	pr3 := getPrivateKey(2)
	pr4 := getPrivateKey(3)
	pr5 := getPrivateKey(4)

	agent1 := NewPbftAgent("Agent1")
	agent2 := NewPbftAgent("Agent2")
	agent3 := NewPbftAgent("Agent3")
	agent4 := NewPbftAgent("Agent4")
	agent5 := NewPbftAgent("Agent5")

	config1 := new(config.TbftConfig)
	*config1 = *config.TestConfig()
	p2p1 := new(config.P2PConfig)
	*p2p1 = *config1.P2P
	p2p1.ListenAddress1 = "tcp://127.0.0.1:28890"
	p2p1.ListenAddress2 = "tcp://127.0.0.1:28891"
	*config1.P2P = *p2p1

	con1 := new(config.ConsensusConfig)
	*con1 = *config1.Consensus
	con1.WalPath = filepath.Join("data", "cs.wal1", "wal")
	*config1.Consensus = *con1

	n1, _ := NewNode(config1, "1", pr1, agent1)
	n1.Start()

	config2 := new(config.TbftConfig)
	*config2 = *config.TestConfig()
	p2p2 := new(config.P2PConfig)
	*p2p2 = *config2.P2P
	p2p2.ListenAddress1 = "tcp://127.0.0.1:28893"
	p2p2.ListenAddress2 = "tcp://127.0.0.1:28894"
	*config2.P2P = *p2p2

	con2 := new(config.ConsensusConfig)
	*con2 = *config2.Consensus
	con2.WalPath = filepath.Join("data", "cs.wal2", "wal")
	*config2.Consensus = *con2

	n2, _ := NewNode(config2, "1", pr2, agent2)
	n2.Start()

	config3 := new(config.TbftConfig)
	*config3 = *config.TestConfig()
	p2p3 := new(config.P2PConfig)
	*p2p3 = *config3.P2P
	p2p3.ListenAddress1 = "tcp://127.0.0.1:28895"
	p2p3.ListenAddress2 = "tcp://127.0.0.1:28896"
	*config3.P2P = *p2p3

	con3 := new(config.ConsensusConfig)
	*con3 = *config3.Consensus
	con3.WalPath = filepath.Join("data", "cs.wal3", "wal")
	*config3.Consensus = *con3

	n3, _ := NewNode(config3, "1", pr3, agent3)
	n3.Start()

	config4 := new(config.TbftConfig)
	*config4 = *config.TestConfig()
	p2p4 := new(config.P2PConfig)
	*p2p4 = *config4.P2P
	p2p4.ListenAddress1 = "tcp://127.0.0.1:28897"
	p2p4.ListenAddress2 = "tcp://127.0.0.1:28898"
	*config4.P2P = *p2p4

	con4 := new(config.ConsensusConfig)
	*con4 = *config4.Consensus
	con4.WalPath = filepath.Join("data", "cs.wal4", "wal")
	*config4.Consensus = *con4

	n4, _ := NewNode(config4, "1", pr4, agent4)
	n4.Start()

	config5 := new(config.TbftConfig)
	*config5 = *config.TestConfig()
	p2p5 := new(config.P2PConfig)
	*p2p5 = *config5.P2P
	p2p5.ListenAddress1 = "tcp://127.0.0.1:28899"
	p2p5.ListenAddress2 = "tcp://127.0.0.1:28900"
	*config5.P2P = *p2p5

	con5 := new(config.ConsensusConfig)
	*con5 = *config5.Consensus
	con5.WalPath = filepath.Join("data", "cs.wal5", "wal")
	*config5.Consensus = *con5

	n5, _ := NewNode(config5, "1", pr5, agent5)
	n5.Start()

	c1 := new(types.CommitteeInfo)
	c1.Id = big.NewInt(1)

	m1 := new(types.CommitteeMember)
	m1.Publickey = GetPub(pr1)
	m1.Coinbase = common.Address{0}

	m2 := new(types.CommitteeMember)
	m2.Publickey = GetPub(pr2)
	m2.Coinbase = common.Address{0}

	m3 := new(types.CommitteeMember)
	m3.Publickey = GetPub(pr3)
	m3.Coinbase = common.Address{0}

	m4 := new(types.CommitteeMember)
	m4.Publickey = GetPub(pr4)
	m4.Coinbase = common.Address{0}

	m5 := new(types.CommitteeMember)
	m5.Publickey = GetPub(pr5)
	m5.Coinbase = common.Address{0}

	c1.Members = append(c1.Members, m1, m2, m3, m4, m5)
	c1.StartHeight = common.Big1

	n1.PutCommittee(c1)
	n1.Notify(c1.Id, Start)

	n2.PutCommittee(c1)
	n2.Notify(c1.Id, Start)

	n3.PutCommittee(c1)
	n3.Notify(c1.Id, Start)

	n4.PutCommittee(c1)
	n4.Notify(c1.Id, Start)

	n5.PutCommittee(c1)
	n5.Notify(c1.Id, Start)

	cn := make([]*types.CommitteeNode, 0)
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28890, Coinbase: m1.Coinbase, Publickey: m1.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28893, Coinbase: m2.Coinbase, Publickey: m2.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28895, Coinbase: m3.Coinbase, Publickey: m3.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28897, Coinbase: m4.Coinbase, Publickey: m4.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28899, Coinbase: m5.Coinbase, Publickey: m5.Publickey})

	n5.PutNodes(common.Big1, cn)
	n4.PutNodes(common.Big1, cn)
	n1.PutNodes(common.Big1, cn)
	n2.PutNodes(common.Big1, cn)
	n3.PutNodes(common.Big1, cn)
	go CloseStart(start)
	<-start
}

func TestRunPbft1(t *testing.T) {
	log.OpenLogDebug(3)
	IDCacheInit()
	start := make(chan int)
	pr1 := getPrivateKey(0)
	pr2 := getPrivateKey(1)
	pr3 := getPrivateKey(2)
	pr4 := getPrivateKey(3)

	agent1 := NewPbftAgent("Agent1")

	config1 := new(config.TbftConfig)
	*config1 = *config.DefaultConfig()
	p2p1 := new(config.P2PConfig)
	*p2p1 = *config1.P2P
	p2p1.ListenAddress1 = "tcp://127.0.0.1:28890"
	p2p1.ListenAddress2 = "tcp://127.0.0.1:28891"
	*config1.P2P = *p2p1

	con1 := new(config.ConsensusConfig)
	*con1 = *config1.Consensus
	con1.WalPath = filepath.Join("data", "cs.wal1", "wal")
	*config1.Consensus = *con1

	n1, _ := NewNode(config1, "1", pr1, agent1)

	c1 := new(types.CommitteeInfo)
	c1.Id = big.NewInt(1)
	m1 := new(types.CommitteeMember)
	m1.Publickey = GetPub(pr1)
	m1.Coinbase = common.Address{0}
	m1.MType = types.TypeWorked
	m1.Flag = types.StateUsedFlag
	m2 := new(types.CommitteeMember)
	m2.Publickey = GetPub(pr2)
	m2.Coinbase = common.Address{0}
	m2.MType = types.TypeWorked
	m2.Flag = types.StateUsedFlag
	m3 := new(types.CommitteeMember)
	m3.Publickey = GetPub(pr3)
	m3.Coinbase = common.Address{0}
	m3.MType = types.TypeWorked
	m3.Flag = types.StateUsedFlag
	m4 := new(types.CommitteeMember)
	m4.Publickey = GetPub(pr4)
	m4.Coinbase = common.Address{0}
	m4.MType = types.TypeWorked
	m4.Flag = types.StateUsedFlag
	c1.Members = append(c1.Members, m1, m2, m3, m4)
	c1.StartHeight = common.Big1
	c1.EndHeight = big.NewInt(11111)

	n1.Start()
	n1.PutCommittee(c1)

	cn := make([]*types.CommitteeNode, 0)
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28890, Port2: 28891, Coinbase: m1.Coinbase, Publickey: m1.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28892, Port2: 28893, Coinbase: m2.Coinbase, Publickey: m2.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28894, Port2: 28895, Coinbase: m3.Coinbase, Publickey: m3.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28896, Port2: 28897, Coinbase: m4.Coinbase, Publickey: m4.Publickey})

	n1.PutNodes(common.Big1, cn)
	n1.Notify(c1.Id, Start)

	//for {
	//	time.Sleep(time.Minute * 5)
	//	c1.Members[3].Flag = types.StateRemovedFlag
	//	c1.Members[3].MType = types.TypeWorked
	//	c1.StartHeight = getIDForCache("Agent3")
	//	c1.EndHeight = new(big.Int).Add(c1.StartHeight, big.NewInt(20))
	//	n1.UpdateCommittee(c1)
	//}
	go CloseStart(start)
	<-start
}

func TestRunPbft2(t *testing.T) {
	//log.OpenLogDebug(3)
	IDCacheInit()
	start := make(chan int)
	pr1 := getPrivateKey(0)
	pr2 := getPrivateKey(1)
	pr3 := getPrivateKey(2)
	pr4 := getPrivateKey(3)

	agent2 := NewPbftAgent("Agent2")

	config2 := new(config.TbftConfig)
	*config2 = *config.DefaultConfig()
	p2p2 := new(config.P2PConfig)
	*p2p2 = *config2.P2P
	p2p2.ListenAddress1 = "tcp://127.0.0.1:28892"
	p2p2.ListenAddress2 = "tcp://127.0.0.1:28893"
	*config2.P2P = *p2p2

	con2 := new(config.ConsensusConfig)
	*con2 = *config2.Consensus
	con2.WalPath = filepath.Join("data", "cs.wal2", "wal")
	*config2.Consensus = *con2

	n2, _ := NewNode(config2, "1", pr2, agent2)

	c1 := new(types.CommitteeInfo)
	c1.Id = big.NewInt(1)
	m1 := new(types.CommitteeMember)
	m1.Publickey = GetPub(pr1)
	m1.Coinbase = common.Address{0}
	m1.MType = types.TypeWorked
	m1.Flag = types.StateUsedFlag
	m2 := new(types.CommitteeMember)
	m2.Publickey = GetPub(pr2)
	m2.Coinbase = common.Address{0}
	m2.MType = types.TypeWorked
	m2.Flag = types.StateUsedFlag
	m3 := new(types.CommitteeMember)
	m3.Publickey = GetPub(pr3)
	m3.Coinbase = common.Address{0}
	m3.MType = types.TypeWorked
	m3.Flag = types.StateUsedFlag
	m4 := new(types.CommitteeMember)
	m4.Publickey = GetPub(pr4)
	m4.Coinbase = common.Address{0}
	m4.MType = types.TypeWorked
	m4.Flag = types.StateUsedFlag
	c1.Members = append(c1.Members, m1, m2, m3, m4)
	c1.StartHeight = common.Big1
	c1.EndHeight = big.NewInt(11111)

	n2.Start()
	n2.PutCommittee(c1)
	n2.Notify(c1.Id, Start)

	cn := make([]*types.CommitteeNode, 0)
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28890, Port2: 28891, Coinbase: m1.Coinbase, Publickey: m1.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28892, Port2: 28893, Coinbase: m2.Coinbase, Publickey: m2.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28894, Port2: 28895, Coinbase: m3.Coinbase, Publickey: m3.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28896, Port2: 28897, Coinbase: m4.Coinbase, Publickey: m4.Publickey})

	n2.PutNodes(common.Big1, cn)

	//for {
	//	time.Sleep(time.Minute * 5)
	//	c1.Members[3].Flag = types.StateRemovedFlag
	//	c1.Members[3].MType = types.TypeWorked
	//	c1.StartHeight = getIDForCache("Agent3")
	//	c1.EndHeight = new(big.Int).Add(c1.StartHeight, big.NewInt(20))
	//	n2.UpdateCommittee(c1)
	//}
	go CloseStart(start)
	<-start
}

func getAddr() common.Address {
	pr1 := getPrivateKey(0)
	pub := GetPubKey(pr1)
	return crypto.PubkeyToAddress(*pub)
}

func TestRunPbft3(t *testing.T) {
	log.OpenLogDebug(3)
	IDCacheInit()
	start := make(chan int)
	pr1 := getPrivateKey(0)
	pr2 := getPrivateKey(1)
	pr3 := getPrivateKey(2)
	pr4 := getPrivateKey(3)

	agent3 := NewPbftAgent("Agent3")

	config3 := new(config.TbftConfig)
	*config3 = *config.DefaultConfig()
	p2p3 := new(config.P2PConfig)
	*p2p3 = *config3.P2P
	p2p3.ListenAddress1 = "tcp://127.0.0.1:28894"
	p2p3.ListenAddress2 = "tcp://127.0.0.1:28895"
	*config3.P2P = *p2p3

	con3 := new(config.ConsensusConfig)
	*con3 = *config3.Consensus
	con3.WalPath = filepath.Join("data", "cs.wal3", "wal")
	*config3.Consensus = *con3

	n3, _ := NewNode(config3, "1", pr3, agent3)

	c1 := new(types.CommitteeInfo)
	c1.Id = big.NewInt(1)
	m1 := new(types.CommitteeMember)
	m1.Publickey = GetPub(pr1)
	m1.Coinbase = common.Address{0}
	m1.MType = types.TypeWorked
	m1.Flag = types.StateUsedFlag
	m2 := new(types.CommitteeMember)
	m2.Publickey = GetPub(pr2)
	m2.Coinbase = common.Address{0}
	m2.MType = types.TypeWorked
	m2.Flag = types.StateUsedFlag
	m3 := new(types.CommitteeMember)
	m3.Publickey = GetPub(pr3)
	m3.Coinbase = common.Address{0}
	m3.MType = types.TypeWorked
	m3.Flag = types.StateUsedFlag
	m4 := new(types.CommitteeMember)
	m4.Publickey = GetPub(pr4)
	m4.Coinbase = common.Address{0}
	m4.MType = types.TypeWorked
	m4.Flag = types.StateUsedFlag
	c1.Members = append(c1.Members, m1, m2, m3, m4)
	c1.StartHeight = common.Big1
	c1.EndHeight = big.NewInt(11111)

	n3.Start()
	n3.PutCommittee(c1)
	n3.Notify(c1.Id, Start)

	cn := make([]*types.CommitteeNode, 0)
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28890, Port2: 28891, Coinbase: m1.Coinbase, Publickey: m1.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28892, Port2: 28893, Coinbase: m2.Coinbase, Publickey: m2.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28894, Port2: 28895, Coinbase: m3.Coinbase, Publickey: m3.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28896, Port2: 28897, Coinbase: m4.Coinbase, Publickey: m4.Publickey})

	n3.PutNodes(common.Big1, cn)

	//for {
	//	time.Sleep(time.Minute * 5)
	//	c1.Members[3].Flag = types.StateRemovedFlag
	//	c1.Members[3].MType = types.TypeWorked
	//	c1.StartHeight = getIDForCache("Agent3")
	//	c1.EndHeight = new(big.Int).Add(c1.StartHeight, big.NewInt(20))
	//	n3.UpdateCommittee(c1)
	//}
	go CloseStart(start)
	<-start
}

func TestRunPbft4(t *testing.T) {
	//log.OpenLogDebug(3)
	IDCacheInit()
	start := make(chan int)
	pr1 := getPrivateKey(0)
	pr2 := getPrivateKey(1)
	pr3 := getPrivateKey(2)
	pr4 := getPrivateKey(3)

	agent4 := NewPbftAgent("Agent4")

	config4 := new(config.TbftConfig)
	*config4 = *config.DefaultConfig()
	p2p4 := new(config.P2PConfig)
	*p2p4 = *config4.P2P
	p2p4.ListenAddress1 = "tcp://127.0.0.1:28896"
	p2p4.ListenAddress2 = "tcp://127.0.0.1:28897"
	*config4.P2P = *p2p4

	con4 := new(config.ConsensusConfig)
	*con4 = *config4.Consensus
	con4.WalPath = filepath.Join("data", "cs.wal4", "wal")
	*config4.Consensus = *con4

	n4, _ := NewNode(config4, "1", pr4, agent4)

	c1 := new(types.CommitteeInfo)
	c1.Id = big.NewInt(1)
	m1 := new(types.CommitteeMember)
	m1.Publickey = GetPub(pr1)
	m1.Coinbase = common.Address{0}
	m1.MType = types.TypeWorked
	m1.Flag = types.StateUsedFlag
	m2 := new(types.CommitteeMember)
	m2.Publickey = GetPub(pr2)
	m2.Coinbase = common.Address{0}
	m2.MType = types.TypeWorked
	m2.Flag = types.StateUsedFlag
	m3 := new(types.CommitteeMember)
	m3.Publickey = GetPub(pr3)
	m3.Coinbase = common.Address{0}
	m3.MType = types.TypeWorked
	m3.Flag = types.StateUsedFlag
	m4 := new(types.CommitteeMember)
	m4.Publickey = GetPub(pr4)
	m4.Coinbase = common.Address{0}
	m4.MType = types.TypeWorked
	m4.Flag = types.StateUsedFlag
	c1.Members = append(c1.Members, m1, m2, m3, m4)
	c1.StartHeight = common.Big1
	c1.EndHeight = big.NewInt(11111)

	n4.Start()
	n4.PutCommittee(c1)
	n4.Notify(c1.Id, Start)

	cn := make([]*types.CommitteeNode, 0)
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28890, Port2: 28891, Coinbase: m1.Coinbase, Publickey: m1.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28892, Port2: 28893, Coinbase: m2.Coinbase, Publickey: m2.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28894, Port2: 28895, Coinbase: m3.Coinbase, Publickey: m3.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "127.0.0.1", Port: 28896, Port2: 28897, Coinbase: m4.Coinbase, Publickey: m4.Publickey})

	n4.PutNodes(common.Big1, cn)

	//for {
	//	time.Sleep(time.Minute * 5)
	//	c1.Members[3].Flag = types.StateRemovedFlag
	//	c1.Members[3].MType = types.TypeWorked
	//	c1.StartHeight = getIDForCache("Agent3")
	//	c1.EndHeight = new(big.Int).Add(c1.StartHeight, big.NewInt(20))
	//	n4.UpdateCommittee(c1)
	//}
	go CloseStart(start)
	<-start
}

func TestAddVote(t *testing.T) {
	IDCacheInit()
	const privCount int = 4
	var privs [privCount]*ecdsa.PrivateKey
	vals := make([]*ttypes.Validator, 0, 0)
	vPrivValidator := make([]ttypes.PrivValidator, 0, 0)

	var chainID = "9999"
	var height uint64 = 1
	var round = 0
	var typeB = ttypes.VoteTypePrevote

	for i := 0; i < privCount; i++ {
		privs[i] = getPrivateKey(i)
		pub := GetPubKey(privs[i])
		vp := ttypes.NewPrivValidator(*privs[i])
		vPrivValidator = append(vPrivValidator, vp)
		v := ttypes.NewValidator(tcrypto.PubKeyTrue(*pub), 1)
		vals = append(vals, v)
	}
	vset := ttypes.NewValidatorSet(vals)
	vVoteSet := ttypes.NewVoteSet(chainID, height, round, typeB, vset)
	// make block
	agent := NewPbftAgent("Agent1")
	cid := big.NewInt(1)
	block, _ := agent.FetchFastBlock(cid, nil)
	hash := block.Hash()
	fmt.Println(common.ToHex(hash[:]))
	ps, _ := ttypes.MakePartSet(65535, block)
	// make vote
	for i, v := range vPrivValidator {
		var vote1 *ttypes.Vote
		if i == 3 {
			vote1 = signAddVote(v, vset, vVoteSet, height, chainID, uint(round), typeB, nil, ttypes.PartSetHeader{}, nil)
		} else {
			vote1 = signAddVote(v, vset, vVoteSet, height, chainID, uint(round), typeB, hash[:], ps.Header(), nil)
		}
		if vote1 != nil {
			vVoteSet.AddVote(vote1)
		}
	}
	bsuc := vVoteSet.HasTwoThirdsMajority()
	fmt.Println(bsuc)
	maj, _ := vVoteSet.TwoThirdsMajority()
	fmt.Println(maj.String())
	signs, _ := vVoteSet.MakePbftSigns(hash[:])
	fmt.Println(signs)

}

func signVote(privV ttypes.PrivValidator, vset *ttypes.ValidatorSet, height uint64, chainid string,
	round uint, typeB byte, hash []byte, header ttypes.PartSetHeader) (*ttypes.Vote, error) {
	addr := privV.GetAddress()
	valIndex, _ := vset.GetByAddress(addr)
	vote := &ttypes.Vote{
		ValidatorAddress: addr,
		ValidatorIndex:   uint(valIndex),
		Height:           height,
		Round:            round,
		Timestamp:        time.Now().UTC(),
		Type:             typeB,
		BlockID:          ttypes.BlockID{Hash: hash, PartsHeader: header},
	}

	err := privV.SignVote(chainid, vote)
	return vote, err
}

func signAddVote(privV ttypes.PrivValidator, vset *ttypes.ValidatorSet, voteset *ttypes.VoteSet, height uint64, chainid string,
	round uint, typeB byte, hash []byte, header ttypes.PartSetHeader, keepsign *ttypes.KeepBlockSign) *ttypes.Vote {

	vote, err := signVote(privV, vset, height, chainid, round, typeB, hash, header)
	if err == nil {
		// if hash != nil && keepsign == nil {
		// 	if prevote := voteset.Prevotes(int(round)); prevote != nil {
		// 		keepsign = prevote.GetSignByAddress(privV.GetAddress())
		// 	}
		// }
		if hash != nil && keepsign != nil && bytes.Equal(hash, keepsign.Hash[:]) {
			vote.Result = keepsign.Result
			vote.ResultSign = make([]byte, len(keepsign.Sign))
			copy(vote.ResultSign, keepsign.Sign)
		}
		log.Trace("Signed and pushed vote", "height", height, "round", round, "vote", vote, "err", err)
		return vote
	}
	log.Trace("Error signing vote", "height", height, "round", round, "vote", vote, "err", err)
	return nil
}

func TestVote(t *testing.T) {
	bid := makeBlockID(nil, ttypes.PartSetHeader{})
	fmt.Println(bid.String())
	aa := len(bid.Hash)
	fmt.Println("aa:", aa)
}

func makeBlockID(hash []byte, header ttypes.PartSetHeader) ttypes.BlockID {
	blockid := ttypes.BlockID{Hash: hash, PartsHeader: header}
	fmt.Println(blockid.String())
	return blockid
}

func TestTock(t *testing.T) {
	taskTimeOut := 3
	var d = time.Duration(taskTimeOut) * time.Second
	ttock := NewTimeoutTicker("ttock")
	ttock.Start()

	ttock.ScheduleTimeout(timeoutInfo{d, 1, uint(0), 0, 1})
	go TimeoutRoutine(&ttock)

	time.Sleep(30 * time.Second)
	ttock.Stop()
}

func TimeoutRoutine(tt *TimeoutTicker) {
	var pos uint
	for {
		if pos >= 30 {
			return
		}
		select {
		case <-(*tt).Chan(): // tockChan:
			pos++
			fmt.Println(time.Now(), pos)
		}
	}

}

func TestPrivKey(t *testing.T) {

	//f05acdc37795769c2afc87c9dac7b22d45c2ae36038d0b70644eebd3aa03e31b
	//70e7a8e57012a04a8b7dbecda7de6af7e2134a2be237cf6049d9bd846362dccc
	//014e09b2406ad03c0539f5f0a0af075b8e70098486f45d3aa1601e00a8d54a93
	//9177485bfecacf47d2f9f63a0c29cc0eae299c955283ce1fbb2060fee041c040
	//d0c3b151031a8a90841dc18463d838cc8db29a10e7889b6991be0a3088702ca7
	//c007a7302da54279edc472174a140b0093580d7d73cdbbb205654ea79f606c95

	priv1, _ := crypto.HexToECDSA("0577aa0d8e070dccfffc5add7ea64ab8a167a3a8badb4ce18e336838e4ce3757")
	tPriv1 := tcrypto.PrivKeyTrue(*priv1)
	addr1 := tPriv1.PubKey().Address()
	id1 := hex.EncodeToString(addr1[:])
	fmt.Println("id1", id1)

	priv2, _ := crypto.HexToECDSA("77b635c48aa8ef386a3cee1094de7ea90f58a634ddb821206e0381f06b860f0f")
	tPriv2 := tcrypto.PrivKeyTrue(*priv2)
	addr2 := tPriv2.PubKey().Address()
	id2 := hex.EncodeToString(addr2[:])
	fmt.Println("id2", id2)

	priv3, _ := crypto.HexToECDSA("14deff62bd0a4b7968cb9fe7d08e41e48d0535ea791d01b5a6d590f265b1ae1c")
	tPriv3 := tcrypto.PrivKeyTrue(*priv3)
	addr3 := tPriv3.PubKey().Address()
	id3 := hex.EncodeToString(addr3[:])
	fmt.Println("id3", id3)

	priv4, _ := crypto.HexToECDSA("20cd359032ba766bcb1468466cf17af81952e6a6be6c8968ed3b18b856950e04")
	tPriv4 := tcrypto.PrivKeyTrue(*priv4)
	addr4 := tPriv4.PubKey().Address()
	id4 := hex.EncodeToString(addr4[:])
	fmt.Println("id4", id4)
}

func TestPutNodes(t *testing.T) {
	IDCacheInit()
	start := make(chan int)
	pr1, _ := crypto.HexToECDSA("2ee9b9082e3eb19378d478f450e0e818e94cf7e3bf13ad5dd657ef2a35fbb0a8")
	pr2, _ := crypto.HexToECDSA("1bc73ab677ed9c3518417339bb5716e32fbc56e888c98d2e63e190dd51ca7eda")
	pr3, _ := crypto.HexToECDSA("d0c3b151031a8a90841dc18463d838cc8db29a10e7889b6991be0a3088702ca7")
	pr4, _ := crypto.HexToECDSA("c007a7302da54279edc472174a140b0093580d7d73cdbbb205654ea79f606c95")
	agent1 := NewPbftAgent("Agent1")

	config1 := new(config.TbftConfig)
	*config1 = *config.TestConfig()
	p2p1 := new(config.P2PConfig)
	*p2p1 = *config1.P2P
	p2p1.ListenAddress1 = "tcp://39.98.44.213:30310"
	p2p1.ListenAddress2 = "tcp://39.98.44.213:30311"
	*config1.P2P = *p2p1

	con1 := new(config.ConsensusConfig)
	*con1 = *config1.Consensus
	con1.WalPath = filepath.Join("data", "cs.wal1", "wal")
	*config1.Consensus = *con1

	n1, _ := NewNode(config1, "1", pr1, agent1)
	n1.Start()

	c1 := new(types.CommitteeInfo)
	c1.Id = big.NewInt(1)
	m1 := new(types.CommitteeMember)
	m1.Publickey = GetPub(pr1)
	m1.Coinbase = common.Address{0}
	m2 := new(types.CommitteeMember)
	m2.Publickey = GetPub(pr2)
	m2.Coinbase = common.Address{0}
	m3 := new(types.CommitteeMember)
	m3.Publickey = GetPub(pr3)
	m3.Coinbase = common.Address{0}
	m4 := new(types.CommitteeMember)
	m4.Publickey = GetPub(pr4)
	m4.Coinbase = common.Address{0}

	c1.Members = append(c1.Members, m1, m2, m3, m4)
	c1.StartHeight = common.Big1

	cn := make([]*types.CommitteeNode, 0)
	cn = append(cn, &types.CommitteeNode{IP: "39.98.44.213", Port: 30310, Port2: 30311, Coinbase: m1.Coinbase, Publickey: m1.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "39.98.58.86", Port: 30310, Port2: 30311, Coinbase: m2.Coinbase, Publickey: m2.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "39.98.56.108", Port: 30310, Port2: 30311, Coinbase: m3.Coinbase, Publickey: m3.Publickey})
	cn = append(cn, &types.CommitteeNode{IP: "39.98.36.181", Port: 30310, Port2: 30311, Coinbase: m4.Coinbase, Publickey: m4.Publickey})

	n1.PutCommittee(c1)
	n1.PutNodes(common.Big1, cn)
	n1.Notify(c1.Id, Start)

	go CloseStart(start)

	<-start
}

func TestWatch(t *testing.T) {
	log.OpenLogDebug(5)
	help.BeginWatchMgr()
	w := help.NewTWatch(3, "111")
	time.Sleep(time.Second * 70)
	w.EndWatch()
	w.Finish("end")
}

func TestValidatorSet(t *testing.T) {

	IDCacheInit()
	const privCount int = 4
	var privs [privCount]*ecdsa.PrivateKey
	vals := make([]*ttypes.Validator, 0, 0)
	vPrivValidator := make([]ttypes.PrivValidator, 0, 0)
	// make validatorset
	for i := 0; i < privCount; i++ {
		privs[i] = getPrivateKey(i)
		pub := GetPubKey(privs[i])
		vp := ttypes.NewPrivValidator(*privs[i])
		vPrivValidator = append(vPrivValidator, vp)
		v := ttypes.NewValidator(tcrypto.PubKeyTrue(*pub), 1)
		vals = append(vals, v)
	}
	vset := ttypes.NewValidatorSet(vals)

	fmt.Println("vs:", vset.String(), " p:", vset.GetProposer())

	vset = newRound(1, vset)
	fmt.Println("p:", vset.GetProposer())
	vset = newRound(0, vset)
	fmt.Println("p:", vset.GetProposer())

	vset = newRound(2, vset)
	fmt.Println("p:", vset.GetProposer())
	vset = newRound(0, vset)
	fmt.Println("p:", vset.GetProposer())

	vset = newRound(3, vset)
	fmt.Println("p:", vset.GetProposer())
	vset = newRound(0, vset)
	fmt.Println("p:", vset.GetProposer())

	vset = newRound(4, vset)
	fmt.Println("p:", vset.GetProposer())
	vset = newRound(0, vset)
	fmt.Println("p:", vset.GetProposer())

	vset = newRound(5, vset)
	fmt.Println("p:", vset.GetProposer())
	vset = newRound(0, vset)
	fmt.Println("p:", vset.GetProposer())

	vset = newRound(6, vset)
	fmt.Println("p:", vset.GetProposer())
	vset = newRound(0, vset)
	fmt.Println("p:", vset.GetProposer())

	fmt.Println("finish")
}

func newRound(round int, vs *ttypes.ValidatorSet) *ttypes.ValidatorSet {
	validators := vs
	if round > 0 {
		validators = validators.Copy()
		validators.IncrementAccum(uint(round))
	}
	return validators
}

func TestStat(t *testing.T) {
	for i := 0; i <= 30; i++ {
		help.DurationStat.AddStartStatTime("b", uint64(i))
		help.DurationStat.AddEndStatTime("b", uint64(i))
		help.DurationStat.AddOtherStat("a", i, uint64(i))
		if i > 20 {
			fmt.Println(help.DurationStat.PrintDurStat())
		}
	}
}

func TestTimer(t *testing.T) {
	timeoutTicker:= NewTimeoutTicker("TimeoutTicker")
	timeoutTicker.Start()
	timeoutTicker.ScheduleTimeout(timeoutInfo{5, 10, uint(0), ttypes.RoundStepNewHeight, 1})
	go func(){
		pos := 0
		for {	
			select {
			case <-timeoutTicker.Chan(): // tockChan:
				pos++
				fmt.Println(pos)
			}
		}
	}()
	time.Sleep(20*time.Second)
	fmt.Println("finish")
}