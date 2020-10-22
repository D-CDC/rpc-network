package tbft

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ethereum/rpc-network/consensus/tbft/testlog"

	tcrypto "ethereum/rpc-network/consensus/tbft/crypto"
	"ethereum/rpc-network/consensus/tbft/help"
	"ethereum/rpc-network/consensus/tbft/tp2p"
	"ethereum/rpc-network/consensus/tbft/tp2p/pex"
	ttypes "ethereum/rpc-network/consensus/tbft/types"
	"ethereum/rpc-network/core/types"
	"ethereum/rpc-network/crypto"
	"github.com/ethereum/go-ethereum/log"
	cfg "ethereum/rpc-network/params"
)

type service struct {
	sw               *tp2p.Switch
	consensusState   *ConsensusState   // latest consensus state
	consensusReactor *ConsensusReactor // for participating in the consensus
	sa               *ttypes.StateAgentImpl
	nodeTable        map[tp2p.ID]*nodeInfo
	lock             *sync.Mutex
	updateChan       chan bool
	eventBus         *ttypes.EventBus // pub/sub for services
	addrBook         pex.AddrBook     // known peers
	healthMgr        *ttypes.HealthMgr
	selfID           tp2p.ID
	singleCon        int32
}

type nodeInfo struct {
	ID      tp2p.ID
	Adrress *tp2p.NetAddress
	IP      string
	Port    uint32
	Enable  bool
	Flag    uint32
}

const (
	//Start is status for notify
	Start int = iota
	//Stop is status for notify
	Stop
	//Switch is status for notify
	Switch
)

func (n *nodeInfo) toString() string {
	return fmt.Sprintf("%+v", n)
}

func newNodeService(p2pcfg *cfg.P2PConfig, cscfg *cfg.ConsensusConfig, state *ttypes.StateAgentImpl,
	store *ttypes.BlockStore, cid uint64) *service {
	return &service{
		sw:             tp2p.NewSwitch(p2pcfg, state),
		consensusState: NewConsensusState(cscfg, state, store),
		// nodeTable:      make(map[p2p.ID]*nodeInfo),
		lock:       new(sync.Mutex),
		updateChan: make(chan bool, 2),
		//eventBus:   ttypes.NewEventBus(),
		// If PEX is on, it should handle dialing the seeds. Otherwise the switch does it.
		// Note we currently use the addrBook regardless at least for AddOurAddress
		addrBook:  pex.NewAddrBook(p2pcfg.AddrBookFile(), p2pcfg.AddrBookStrict),
		healthMgr: ttypes.NewHealthMgr(cid),
		singleCon: 0,
	}
}

func (s *service) nodesHaveSelf() bool {
	if s.sa.Priv == nil {
		return true
	}
	v, ok := s.nodeTable[s.selfID]
	return ok && v.Flag == types.StateUsedFlag
}

func (s *service) setNodes(nodes map[tp2p.ID]*nodeInfo) {
	s.nodeTable = nodes
}
func (s *service) start(cid *big.Int, node *Node) error {
	//err := s.eventBus.Start()
	//if err != nil {
	//	return err
	//}
	// Create & add listener
	if s.sw.IsRunning() {
		log.Warn("service is running")
		return errors.New("service is running")
	}

	lstr := node.config.P2P.ListenAddress2
	if cid.Uint64()%2 == 0 {
		lstr = node.config.P2P.ListenAddress1
	}
	nodeinfo := node.nodeinfo
	_, lAddr := help.ProtocolAndAddress(lstr)
	lAddrIP, lAddrPort := tp2p.SplitHostPort(lAddr)
	nodeinfo.ListenAddr = fmt.Sprintf("%v:%v", lAddrIP, lAddrPort)
	// Add ourselves to addrbook to prevent dialing ourselves
	s.addrBook.AddOurAddress(nodeinfo.NetAddress())
	// Add private IDs to addrbook to block those peers being added
	s.addrBook.AddPrivateIDs(help.SplitAndTrim(node.config.P2P.PrivatePeerIDs, ",", " "))

	s.sw.SetNodeInfo(nodeinfo)
	s.sw.SetNodeKey(&node.nodekey)
	l := tp2p.NewDefaultListener(
		lstr,
		node.config.P2P.ExternalAddress,
		node.config.P2P.UPNP,
		log.New("p2p", "self"))
	s.sw.AddListener(l)

	privValidator := ttypes.NewPrivValidator(*node.priv)
	s.consensusState.SetPrivValidator(privValidator)
	s.sa.SetPrivValidator(privValidator)
	// Start the switch (the P2P server).
	help.CheckAndPrintError(s.healthMgr.OnStart())
	err := s.sw.Start()
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case update := <-s.updateChan:
				if update {
					if swap := atomic.CompareAndSwapInt32(&s.singleCon, 0, 1); swap {
						go s.updateNodes()
					}
				} else {
					return // exit
				}
			}
		}
	}()
	return nil
}
func (s *service) stop() error {
	log.Info("begin service stop")
	if s.sw.IsRunning() {
		s.updateChan <- false
		s.healthMgr.OnStop()
		help.CheckAndPrintError(s.sw.Stop())
		//help.CheckAndPrintError(s.eventBus.Stop())
		log.Info("end service stop")
	}
	return nil
}
func (s *service) getStateAgent() *ttypes.StateAgentImpl {
	return s.sa
}
func (s *service) putNodes(cid *big.Int, nodes []*types.CommitteeNode) {
	log.Trace("putNodes", "cid", cid, "nodes", nodes)
	if nodes == nil {
		return
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	update := false
	nodeString := make([]string, len(nodes))
	for i, node := range nodes {
		nodeString[i] = node.String()
		pub, err := crypto.UnmarshalPubkey(node.Publickey)
		if err != nil {
			log.Debug("putnode:", "err", err, "ip", node.IP, "port", node.Port)
			continue
		}
		// check node pk
		address := crypto.PubkeyToAddress(*pub)
		port := node.Port2
		if cid.Uint64()%2 == 0 {
			port = node.Port
		}
		id := tp2p.ID(hex.EncodeToString(address[:]))
		addr, err := tp2p.NewNetAddressString(tp2p.IDAddressString(id,
			fmt.Sprintf("%v:%v", node.IP, port)))
		if v, ok := s.nodeTable[id]; ok {
			v.Adrress = addr
			if v.IP != node.IP || v.Port != port {
				v.IP, v.Port = node.IP, port
				update = true
			}
		}

		s.healthMgr.UpdataHealthInfo(id, node.IP, port, node.Publickey)
	}
	if update && s.nodesHaveSelf() { //} ((s.sa.Priv != nil && s.consensusState.Validators.HasAddress(s.sa.Priv.GetAddress())) || s.sa.Priv == nil) {
		select {
		case s.updateChan <- true:
		default:
		}
	}
}

//pkToP2pID pk to p2p id
func pkToP2pID(pk *ecdsa.PublicKey) tp2p.ID {
	publicKey := crypto.FromECDSAPub(pk)
	pub, err := crypto.UnmarshalPubkey(publicKey)
	if err != nil {
		return ""
	}
	address := crypto.PubkeyToAddress(*pub)
	return tp2p.ID(hex.EncodeToString(address[:]))
}

func (s *service) updateNodes() {
	s.lock.Lock()
	defer s.lock.Unlock()
	defer atomic.StoreInt32(&s.singleCon, 0)

	testlog.AddLog("nodeTableLen", len(s.nodeTable))

	for _, v := range s.nodeTable {
		if v != nil {
			testlog.AddLog("connToBegin", v.ID)
			if s.canConn(v) {
				testlog.AddLog("connTo", v.toString())
				//Give each connection a time difference and reduce peer-to-peer connectivity issues
				time.Sleep(time.Duration(help.RandInt31n(5)) * time.Second)
				s.connTo(v)
			}
		}
	}
}

//add self check
func (s *service) canConn(v *nodeInfo) bool {
	if !v.Enable && v.Flag == types.StateUsedFlag && v.Adrress != nil && v.ID != s.selfID {
		return true
	}
	return false
}

func (s *service) connTo(node *nodeInfo) {
	if node.Enable {
		return
	}
	log.Trace("[put nodes]connTo", "addr", node.Adrress)
	errDialErr := s.sw.DialPeerWithAddress(node.Adrress, true)
	if errDialErr != nil {
		testlog.AddLog("errDialErr:", errDialErr.Error())
		if strings.HasPrefix(errDialErr.Error(), "Duplicate peer ID") {
			go s.checkPeerForDuplicate(node)
			node.Enable = true
		} else {
			log.Debug("[connTo] dail peer " + errDialErr.Error())
		}
	} else {
		node.Enable = true
	}
}

// Solve peer-to-peer connectivity issues
// When two nodes are connected at the same time,
// it may confirm the other party's connection and close their own connection at the same time,
// causing a pseudo connection.
func (s *service) checkPeerForDuplicate(node *nodeInfo) {
	log.Warn("checkPeerForDuplicate", "node", node.ID)
	cnt := 0
	for {
		if !s.sw.IsRunning() {
			testlog.AddLog("checkPeerForDuplicate", "stop", "node", node.ID)
			break
		}
		time.Sleep(time.Second)
		cnt++
		if cnt > 35 {
			tick := s.healthMgr.GetHealthTick(node.ID)
			if tick < 30 || tick > 1800 {
				testlog.AddLog("checkPeerForDuplicate", "stop", "node", node.ID, "tick", tick)
				break
			}
			p := s.sw.Peers().Get(node.ID)
			if p != nil {
				testlog.AddLog("checkPeerForDuplicate", "stop", "node", node.ID, "peer", "stop")
				s.sw.StopPeerGracefully(p)
			}

			time.Sleep(time.Duration(help.RandInt31n(5)) * time.Second)
			err := s.sw.DialPeerWithAddress(node.Adrress, true)
			if err == nil {
				testlog.AddLog("checkPeerForDuplicate", "stop", "node", node.ID, "DialPeerWithAddress", "ok")
				break
			}
			if strings.HasPrefix(err.Error(), "Duplicate peer ID") {
				cnt = 0
				testlog.AddLog("checkPeerForDuplicate", "new round", "node", node.ID, "Duplicate peer", "again")
			} else {
				node.Enable = false
				testlog.AddLog("checkPeerForDuplicate", "new round", "node", node.ID, "err", err.Error())
				break
			}
		}
	}
}

// EventBus returns the Node's EventBus.
func (s *service) EventBus() *ttypes.EventBus {
	return s.eventBus
}

//------------------------------------------------------------------------------

// Node is the highest level interface to a full truechain node.
// It includes all configuration information and running services.
type Node struct {
	help.BaseService
	// configt
	config *cfg.TbftConfig
	Agent  types.PbftAgentProxy
	priv   *ecdsa.PrivateKey // local node's validator key

	// services
	services   map[uint64]*service
	nodekey    tp2p.NodeKey
	nodeinfo   tp2p.NodeInfo
	chainID    string
	lock       *sync.Mutex
	servicePre uint64
}

// NewNode returns a new, ready to go, truechain Node.
func NewNode(config *cfg.TbftConfig, chainID string, priv *ecdsa.PrivateKey,
	agent types.PbftAgentProxy) (*Node, error) {

	// Optionally, start the pex reactor
	// We need to set Seeds and PersistentPeers on the switch,
	// since it needs to be able to use these (and their DNS names)
	// even if the PEX is off. We can include the DNS name in the NetAddress,
	// but it would still be nice to have a clear list of the current "PersistentPeers"
	// somewhere that we can return with net_info.

	// services which will be publishing and/or subscribing for messages (events)
	// consensusReactor will set it on consensusState and blockExecutor
	node := &Node{
		config:   config,
		priv:     priv,
		chainID:  chainID,
		Agent:    agent,
		lock:     new(sync.Mutex),
		services: make(map[uint64]*service),
		nodekey: tp2p.NodeKey{
			PrivKey: tcrypto.PrivKeyTrue(*priv),
		},
	}
	node.BaseService = *help.NewBaseService("Node", node)
	return node, nil
}

// OnStart starts the Node. It implements help.Service.
func (n *Node) OnStart() error {
	n.nodeinfo = n.makeNodeInfo()
	help.BeginWatchMgr()
	return nil
}

// OnStop stops the Node. It implements help.Service.
func (n *Node) OnStop() {
	n.lock.Lock()
	defer n.lock.Unlock()
	for _, v := range n.services {
		help.CheckAndPrintError(v.stop())
	}
	help.EndWatchMgr()
	// first stop the non-reactor services
	// now stop the reactors
	// TODO: gracefully disconnect from peers.
}

// RunForever waits for an interrupt signal and stops the node.
func (n *Node) RunForever() {
	// Sleep forever and then...
	//cmn.TrapSignal(func() {
	//	n.Stop()
	//})
}

func (n *Node) makeNodeInfo() tp2p.NodeInfo {
	nodeInfo := tp2p.NodeInfo{
		ID:      n.nodekey.ID(),
		Network: n.chainID,
		Version: "0.1.0",
		Channels: []byte{
			StateChannel,
			DataChannel,
			VoteChannel,
			VoteSetBitsChannel,
		},
		Moniker: n.config.Moniker,
		Other: []string{
			fmt.Sprintf("p2p_version=%v", "0.1.0"),
			fmt.Sprintf("consensus_version=%v", "0.1.0"),
		},
	}
	// Split protocol, address, and port.
	_, lAddr := help.ProtocolAndAddress(n.config.P2P.ListenAddress1)
	lAddrIP, lAddrPort := tp2p.SplitHostPort(lAddr)
	nodeInfo.ListenAddr = fmt.Sprintf("%v:%v", lAddrIP, lAddrPort)
	return nodeInfo
}

//Notify is agent change server order
func (n *Node) Notify(id *big.Int, action int) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	switch action {
	case Start:
		log.Info("Notify Node Action:start", "id", id.Uint64())
		if server, ok := n.services[id.Uint64()]; ok {
			if server.consensusState == nil {
				panic(0)
			}
			//check and delete service
			if n.servicePre != id.Uint64() {
				if _, ok := n.services[n.servicePre]; ok {
					delete(n.services, n.servicePre)
				}
				n.servicePre = id.Uint64()
			}
			log.Info("Begin start committee", "id", id.Uint64(), "cur", server.consensusState.Height, "begin", server.sa.BeginHeight, "stop", server.sa.EndHeight)
			help.CheckAndPrintError(server.start(id, n))
			log.Info("End start committee", "id", id.Uint64(), "cur", server.consensusState.Height, "begin", server.sa.BeginHeight, "stop", server.sa.EndHeight)
			return nil
		}
		return errors.New("wrong conmmitt ID:" + id.String())

	case Stop:
		log.Info("Notify Node Action:stop", "id", id.Uint64())
		if server, ok := n.services[id.Uint64()]; ok {
			log.Info("Begin stop committee", "id", id.Uint64(), "cur", server.consensusState.Height, "begin", server.sa.BeginHeight, "stop", server.sa.EndHeight)
			help.CheckAndPrintError(server.stop())
			//delete(n.services, id.Uint64())
			log.Info("End stop committee", "id", id.Uint64(), "cur", server.consensusState.Height, "begin", server.sa.BeginHeight, "stop", server.sa.EndHeight)
		}
		return nil
	case Switch:
		// begin to make network..
		return nil
	}
	return nil
}

//PutCommittee is agent put all committee to server
func (n *Node) PutCommittee(committeeInfo *types.CommitteeInfo) error {
	id := committeeInfo.Id
	members := committeeInfo.Members
	if id == nil || len(members) <= 0 {
		return errors.New("wrong params")
	}
	n.lock.Lock()
	defer n.lock.Unlock()
	if _, ok := n.services[id.Uint64()]; ok {
		return errors.New("repeat ID:" + id.String())
	}
	log.Info("pbft PutCommittee", "info", committeeInfo.String())
	// Make StateAgent
	startHeight := committeeInfo.StartHeight.Uint64()
	cid := id.Uint64()
	state := ttypes.NewStateAgent(n.Agent, n.chainID, MakeValidators(committeeInfo), startHeight, cid)
	if state == nil {
		return errors.New("make the nil state")
	}
	if committeeInfo.EndHeight != nil && committeeInfo.EndHeight.Cmp(committeeInfo.StartHeight) > 0 {
		state.SetEndHeight(committeeInfo.EndHeight.Uint64())
	}

	store := ttypes.NewBlockStore()
	service := newNodeService(n.config.P2P, n.config.Consensus, state, store, cid)

	if len(committeeInfo.Members) < cfg.MinimumCommitteeNumber {
		return fmt.Errorf("members len is error :want big to %d get %d", cfg.MinimumCommitteeNumber, len(committeeInfo.Members))
	}

	n.AddHealthForCommittee(service.healthMgr, committeeInfo)

	service.consensusState.SetHealthMgr(service.healthMgr)
	service.consensusState.SetCommitteeInfo(committeeInfo)
	nodeInfo := makeCommitteeMembers(service, committeeInfo)
	log.Trace("put committee", "nodeinfo", nodeInfo)
	if nodeInfo == nil {
		help.CheckAndPrintError(service.stop())
		return errors.New("make the nil CommitteeMembers")
	}

	service.setNodes(nodeInfo)
	service.sa = state
	service.consensusReactor = NewConsensusReactor(service.consensusState, false)
	service.sw.AddReactor("CONSENSUS", service.consensusReactor)
	service.sw.SetAddrBook(service.addrBook)
	service.consensusReactor.SetHealthMgr(service.healthMgr)
	//service.consensusReactor.SetEventBus(service.eventBus)
	service.selfID = n.nodekey.ID()
	n.services[id.Uint64()] = service
	return nil
}

func (n *Node) AddHealthForCommittee(h *ttypes.HealthMgr, c *types.CommitteeInfo) {

	for _, v := range c.Members {
		pk, e := crypto.UnmarshalPubkey(v.Publickey)
		if e != nil {
			log.Debug("AddHealthForCommittee pk error", "pk", v.Publickey)
		}
		id := pkToP2pID(pk)
		//exclude self
		self := false
		if n.nodekey.PubKey().Equals(tcrypto.PubKeyTrue(*pk)) {
			self = true
		}
		val := ttypes.NewValidator(tcrypto.PubKeyTrue(*pk), 1)
		health := ttypes.NewHealth(id, v.MType, v.Flag, val, self)
		h.PutWorkHealth(health)
	}

	for _, v := range c.BackMembers {
		pk, e := crypto.UnmarshalPubkey(v.Publickey)
		if e != nil {
			log.Debug("AddHealthForCommittee pk error", "pk", v.Publickey)
		}
		id := pkToP2pID(pk)
		val := ttypes.NewValidator(tcrypto.PubKeyTrue(*pk), 1)
		self := false
		if n.nodekey.PubKey().Equals(tcrypto.PubKeyTrue(*pk)) {
			self = true
		}
		health := ttypes.NewHealth(id, v.MType, v.Flag, val, self)
		h.PutBackHealth(health)
	}
}

//PutNodes is agent put peer's ip port
func (n *Node) PutNodes(id *big.Int, nodes []*types.CommitteeNode) error {
	if id == nil || len(nodes) <= 0 {
		return errors.New("wrong params")
	}
	n.lock.Lock()
	defer n.lock.Unlock()

	server, ok := n.services[id.Uint64()]
	if !ok {
		return errors.New("wrong ID:" + id.String())
	}
	server.putNodes(id, nodes)

	return nil
}

func (n *Node) checkValidatorSet(service *service, info *types.CommitteeInfo) (selfStop bool, remove []*types.CommitteeMember) {
	allMembers := append(info.Members, info.BackMembers...)
	for _, v := range allMembers {
		if v.Flag == types.StateRemovedFlag {
			pk, e := crypto.UnmarshalPubkey(v.Publickey)
			if e != nil {
				log.Debug("checkValidatorSet pk error", "pk", v.Publickey)
			}
			if service.consensusState.state.GetPubKey().Equals(tcrypto.PubKeyTrue(*pk)) {
				selfStop = true
			}
			remove = append(remove, v)
		}
	}
	return
}

// UpdateCommittee update the committee info from agent when the members was changed
func (n *Node) UpdateCommittee(info *types.CommitteeInfo) error {
	n.lock.Lock()
	defer n.lock.Unlock()
	log.Trace("UpdateCommittee", "info", info)
	if service, ok := n.services[info.Id.Uint64()]; ok {
		//update validator
		stop, member := n.checkValidatorSet(service, info)
		val := MakeValidators(info)
		service.consensusState.UpdateValidatorsSet(val, info.StartHeight.Uint64(), info.EndHeight.Uint64())

		for _, v := range member {
			pk, e := crypto.UnmarshalPubkey(v.Publickey)
			if e != nil {
				log.Debug("UpdateCommittee pk error", "pk", v.Publickey)
			}
			pID := pkToP2pID(pk)
			p := service.sw.GetPeerForID(string(pID))
			if p != nil {
				service.sw.StopPeerForError(p, nil)
			}
		}

		//update node info
		service.lock.Lock()
		ni := makeCommitteeMembers(service, info)
		for k, v := range service.nodeTable {
			if vn, ok := ni[k]; ok {
				v.Flag = vn.Flag
			}
		}
		service.lock.Unlock()

		if stop {
			help.CheckAndPrintError(service.stop())
		}

		//update nodes
		//nodes := makeCommitteeMembersForUpdateCommittee(info)
		//service.setNodes(nodes)
		go func() { service.updateChan <- true }()
		//update health
		service.healthMgr.UpdateFromCommittee(info.Members, info.BackMembers)
		return nil

	}
	return errors.New("service not found")
}

//MakeValidators is make CommitteeInfo to ValidatorSet
func MakeValidators(cmm *types.CommitteeInfo) *ttypes.ValidatorSet {
	id := cmm.Id
	members := append(cmm.Members, cmm.BackMembers...)
	if id == nil || len(members) <= 0 {
		return nil
	}
	vals := make([]*ttypes.Validator, 0, 0)
	var power int64 = 1
	for i, m := range members {
		if m.Flag != types.StateUsedFlag {
			continue
		}
		if i == 0 {
			power = 1
		} else {
			power = 1
		}
		pk, e := crypto.UnmarshalPubkey(m.Publickey)
		if e != nil {
			log.Debug("MakeValidators pk error", "pk", m.Publickey)
		}
		v := ttypes.NewValidator(tcrypto.PubKeyTrue(*pk), power)
		vals = append(vals, v)
	}
	return ttypes.NewValidatorSet(vals)
}
func makeCommitteeMembers(ss *service, cmm *types.CommitteeInfo) map[tp2p.ID]*nodeInfo {
	members := append(cmm.Members, cmm.BackMembers...)
	if ss == nil || len(members) <= 0 {
		return nil
	}
	tab := make(map[tp2p.ID]*nodeInfo)
	for i, m := range members {
		id := tp2p.ID(hex.EncodeToString(m.CommitteeBase.Bytes()))
		tab[id] = &nodeInfo{
			ID:   id,
			Flag: m.Flag,
		}
		log.Trace("CommitteeMembers", "index", i, "id", id)
	}
	return tab
}

//SetCommitteeStop is stop committeeID server
func (n *Node) SetCommitteeStop(committeeID *big.Int, stop uint64) error {
	log.Trace("SetCommitteeStop", "id", committeeID, "stop", stop)
	n.lock.Lock()
	defer n.lock.Unlock()

	if server, ok := n.services[committeeID.Uint64()]; ok {
		server.getStateAgent().SetEndHeight(stop)
		return nil
	}
	return errors.New("wrong conmmitt ID:" + committeeID.String())
}

func getCommittee(n *Node, cid uint64) (info *service) {
	n.lock.Lock()
	defer n.lock.Unlock()
	if server, ok := n.services[cid]; ok {
		return server
	}
	return nil
}

func getNodeStatus(s *service) map[string]interface{} {
	result := make(map[string]interface{})
	result[strconv.Itoa(int(s.consensusState.Height))] = s.consensusState.GetRoundState().Votes
	return result
}

//GetCommitteeStatus is show committee info in api
func (n *Node) GetCommitteeStatus(committeeID *big.Int) map[string]interface{} {
	log.Trace("GetCommitteeStatus", "committeeID", committeeID.Uint64())
	result := make(map[string]interface{})
	s := getCommittee(n, committeeID.Uint64())
	if s != nil {
		committee := make(map[string]interface{})
		committee["id"] = committeeID.Uint64()
		committee["nodes"] = s.nodeTable
		committee["nodes_cnt"] = len(s.nodeTable)
		result["committee_now"] = committee
		result["nodeStatus"] = getNodeStatus(s)
	} else {
		log.Trace("GetCommitteeStatus", "error", "server not have")
	}

	s1 := getCommittee(n, committeeID.Uint64()+1)
	if s1 != nil {
		committee := make(map[string]interface{})
		committee["id"] = committeeID.Uint64() + 1
		committee["nodes"] = s1.nodeTable
		result["committee_next"] = committee
	}
	result["stat"] = help.DurationStat.PrintDurStat()
	return result
}

func (n *Node) IsLeader(committeeID *big.Int) bool {
	s := getCommittee(n, committeeID.Uint64())
	if s != nil && s.consensusState != nil {
		return s.consensusState.isProposer()
	}
	return false
}

//check Committee
func (n *Node) verifyCommitteeInfo(cm *types.CommitteeInfo) error {
	//checkFlag
	if cm.Id.Uint64() == 0 {
		for _, v := range cm.Members {
			if v.Flag != types.StateUsedFlag ||
				v.MType != types.StateRemovedFlag {
				return errors.New("committee member error 0")
			}
		}
		return nil
	}

	for _, v := range cm.Members {
		if v.Flag != types.StateUsedFlag &&
			v.Flag != types.StateRemovedFlag {
			return errors.New("committee member error 1")
		}
	}

	var seeds []*types.CommitteeMember

	for _, v := range cm.BackMembers {
		if v.Flag != types.StateUsedFlag &&
			v.Flag != types.StateRemovedFlag &&
			v.Flag != types.StateUnusedFlag {
			return errors.New("committee member error 2")
		}
		if v.Flag == types.TypeFixed {
			seeds = append(seeds, v)
		}
	}

	cSeeds := n.Agent.GetSeedMember()

	return n.verifySeedNode(seeds, cSeeds)
}

//check seed node
func (n *Node) verifySeedNode(seeds []*types.CommitteeMember, cSeeds []*types.CommitteeMember) error {
	if len(seeds) == 0 || len(cSeeds) == 0 || (len(seeds) != len(cSeeds)) {
		return errors.New("committee member error 3")
	}
	for i := len(seeds) - 1; i >= 0; i-- {
		if !seeds[i].Compared(cSeeds[i]) {
			return errors.New("committee member error 4")
		}
	}
	return nil
}
