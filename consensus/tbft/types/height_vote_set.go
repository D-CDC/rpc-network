package types

import (
	"bytes"
	"errors"
	"fmt"
	"ethereum/rpc-network/consensus/tbft/help"
	"strings"
	"sync"
)

//RoundVoteSet struct
type RoundVoteSet struct {
	Prevotes   *VoteSet
	Precommits *VoteSet
}

var (
	//ErrorGotVoteFromUnwantedRound is Peer has sent a vote that does not match our round for more than one round
	ErrorGotVoteFromUnwantedRound = errors.New("peer has sent a vote that does not match our round for more than one round")
)

// HeightVoteSet comment
/*
Keeps track of all VoteSets from round 0 to round 'round'.

Also keeps track of up to one RoundVoteSet greater than
'round' from each peer, to facilitate catchup syncing of commits.

A commit is +2/3 precommits for a block at a round,
but which round is not known in advance, so when a peer
provides a precommit for a round greater than mtx.round,
we create a new entry in roundVoteSets but also remember the
peer to prevent abuse.
We let each peer provide us with up to 2 unexpected "catchup" rounds.
One for their LastCommit round, and another for the official commit round.
*/
type HeightVoteSet struct {
	chainID string
	height  uint64
	valSet  *ValidatorSet

	mtx               sync.Mutex
	round             int                  // max tracked round
	roundVoteSets     map[int]RoundVoteSet // keys: [0...round]
	peerCatchupRounds map[string][]int     // keys: peer.ID; values: at most 2 rounds
}

//NewHeightVoteSet get new HeightVoteSet
func NewHeightVoteSet(chainID string, height uint64, valSet *ValidatorSet) *HeightVoteSet {
	hvs := &HeightVoteSet{
		chainID: chainID,
	}
	hvs.Reset(height, valSet)
	return hvs
}

//Reset is reset ValidatorSet
func (hvs *HeightVoteSet) Reset(height uint64, valSet *ValidatorSet) {
	hvs.mtx.Lock()
	defer hvs.mtx.Unlock()

	hvs.height = height
	hvs.valSet = valSet
	hvs.roundVoteSets = make(map[int]RoundVoteSet)
	hvs.peerCatchupRounds = make(map[string][]int)

	hvs.addRound(0)
	hvs.round = 0
}

//Height get now VoteSet Height
func (hvs *HeightVoteSet) Height() uint64 {
	hvs.mtx.Lock()
	defer hvs.mtx.Unlock()
	return hvs.height
}

//Round get now VoteSet round
func (hvs *HeightVoteSet) Round() int {
	hvs.mtx.Lock()
	defer hvs.mtx.Unlock()
	return hvs.round
}

// SetRound Create more RoundVoteSets up to round.
func (hvs *HeightVoteSet) SetRound(round int) {
	hvs.mtx.Lock()
	defer hvs.mtx.Unlock()
	if hvs.round != 0 && (round < hvs.round+1) {
		// cmn.PanicSanity("SetRound() must increment hvs.round")
		panic(0)
	}
	for r := hvs.round + 1; r <= round; r++ {
		if _, ok := hvs.roundVoteSets[r]; ok {
			continue // Already exists because peerCatchupRounds.
		}
		hvs.addRound(r)
	}
	hvs.round = round
}

func (hvs *HeightVoteSet) addRound(round int) {
	if _, ok := hvs.roundVoteSets[round]; ok {
		// cmn.PanicSanity("addRound() for an existing round")
		panic(0)
	}
	// log.Debug("addRound(round)", "round", round)
	prevotes := NewVoteSet(hvs.chainID, hvs.height, round, VoteTypePrevote, hvs.valSet)
	precommits := NewVoteSet(hvs.chainID, hvs.height, round, VoteTypePrecommit, hvs.valSet)
	hvs.roundVoteSets[round] = RoundVoteSet{
		Prevotes:   prevotes,
		Precommits: precommits,
	}
}

//AddVote Duplicate votes return added=false, err=nil.
// By convention, peerID is "" if origin is self.
func (hvs *HeightVoteSet) AddVote(vote *Vote, peerID string) (added bool, err error) {
	hvs.mtx.Lock()
	defer hvs.mtx.Unlock()
	if !IsVoteTypeValid(vote.Type) {
		return
	}
	voteSet := hvs.getVoteSet(int(vote.Round), vote.Type)
	if voteSet == nil {
		if rndz := hvs.peerCatchupRounds[peerID]; len(rndz) < 2 {
			hvs.addRound(int(vote.Round))
			voteSet = hvs.getVoteSet(int(vote.Round), vote.Type)
			hvs.peerCatchupRounds[peerID] = append(rndz, int(vote.Round))
		} else {
			// punish peer
			err = ErrorGotVoteFromUnwantedRound
			return
		}
	}
	added, err = voteSet.AddVote(vote)
	return
}

//Prevotes get preVote vote
func (hvs *HeightVoteSet) Prevotes(round int) *VoteSet {
	hvs.mtx.Lock()
	defer hvs.mtx.Unlock()
	return hvs.getVoteSet(round, VoteTypePrevote)
}

//Precommits get  preCommit vote
func (hvs *HeightVoteSet) Precommits(round int) *VoteSet {
	hvs.mtx.Lock()
	defer hvs.mtx.Unlock()
	return hvs.getVoteSet(round, VoteTypePrecommit)
}

// POLInfo Last round and blockID that has +2/3 prevotes for a particular block or nil.
// Returns -1 if no such round exists.
func (hvs *HeightVoteSet) POLInfo() (polRound int, polBlockID BlockID) {
	hvs.mtx.Lock()
	defer hvs.mtx.Unlock()
	for r := hvs.round; r >= 0; r-- {
		rvs := hvs.getVoteSet(r, VoteTypePrevote)
		polBlockID, ok := rvs.TwoThirdsMajority()
		if ok {
			return r, polBlockID
		}
	}
	return -1, BlockID{}
}

func (hvs *HeightVoteSet) getVoteSet(round int, typeB byte) *VoteSet {
	rvs, ok := hvs.roundVoteSets[round]
	if !ok {
		return nil
	}
	switch typeB {
	case VoteTypePrevote:
		return rvs.Prevotes
	case VoteTypePrecommit:
		return rvs.Precommits
	default:
		// cmn.PanicSanity(cmn.Fmt("Unexpected vote type %X", type_))
		return nil
	}
}

// SetPeerMaj23 If a peer claims that it has 2/3 majority for given blockKey, call this.
// NOTE: if there are too many peers, or too much peer churn,
// this can cause memory issues.
// TODO: implement ability to remove peers too
func (hvs *HeightVoteSet) SetPeerMaj23(round int, typeB byte, peerID string, blockID BlockID) error {
	hvs.mtx.Lock()
	defer hvs.mtx.Unlock()
	if !IsVoteTypeValid(typeB) {
		return fmt.Errorf("SetPeerMaj23: Invalid vote type %v", typeB)
	}
	voteSet := hvs.getVoteSet(round, typeB)
	if voteSet == nil {
		return nil // something we don't know about yet
	}
	return voteSet.SetPeerMaj23(P2PID(peerID), blockID)
}

//GetSignsFromVote get sign from all round
func (hvs *HeightVoteSet) GetSignsFromVote(round int, hash []byte, addr help.Address) *KeepBlockSign {
	for r := round; r >= 0; r-- {
		if prevote := hvs.Prevotes(r); prevote != nil {
			keepsign := prevote.GetSignByAddress(addr)
			if keepsign != nil && bytes.Equal(hash, keepsign.Hash[:]) {
				return keepsign
			}
		}
	}
	return nil
}

//---------------------------------------------------------
// string and json
func (hvs *HeightVoteSet) String() string {
	return hvs.StringIndented("")
}

//StringIndented get Indented voteSet string
func (hvs *HeightVoteSet) StringIndented(indent string) string {
	hvs.mtx.Lock()
	defer hvs.mtx.Unlock()
	vsStrings := make([]string, 0, (len(hvs.roundVoteSets)+1)*2)
	// rounds 0 ~ hvs.round inclusive
	for round := 0; round <= hvs.round; round++ {
		voteSetString := hvs.roundVoteSets[round].Prevotes.StringShort()
		vsStrings = append(vsStrings, voteSetString)
		voteSetString = hvs.roundVoteSets[round].Precommits.StringShort()
		vsStrings = append(vsStrings, voteSetString)
	}
	// all other peer catchup rounds
	for round, roundVoteSet := range hvs.roundVoteSets {
		if round <= hvs.round {
			continue
		}
		voteSetString := roundVoteSet.Prevotes.StringShort()
		vsStrings = append(vsStrings, voteSetString)
		voteSetString = roundVoteSet.Precommits.StringShort()
		vsStrings = append(vsStrings, voteSetString)
	}
	return fmt.Sprintf(`HeightVoteSet{H:%v R:0~%v
%s  %v
%s}`,
		hvs.height, hvs.round,
		indent, strings.Join(vsStrings, "\n"+indent+"  "),
		indent)
}

//MarshalJSON get all votes json
func (hvs *HeightVoteSet) MarshalJSON() ([]byte, error) {
	hvs.mtx.Lock()
	defer hvs.mtx.Unlock()

	allVotes := hvs.toAllRoundVotes()
	return cdc.MarshalJSON(allVotes)
}

func (hvs *HeightVoteSet) toAllRoundVotes() []roundVotes {
	totalRounds := hvs.round + 1
	allVotes := make([]roundVotes, totalRounds)
	// rounds 0 ~ hvs.round inclusive
	for round := 0; round < totalRounds; round++ {
		allVotes[round] = roundVotes{
			Round:              round,
			Prevotes:           hvs.roundVoteSets[round].Prevotes.VoteStrings(),
			PrevotesBitArray:   hvs.roundVoteSets[round].Prevotes.BitArrayString(),
			Precommits:         hvs.roundVoteSets[round].Precommits.VoteStrings(),
			PrecommitsBitArray: hvs.roundVoteSets[round].Precommits.BitArrayString(),
		}
	}
	// TODO: all other peer catchup rounds
	return allVotes
}

type roundVotes struct {
	Round              int      `json:"round"`
	Prevotes           []string `json:"prevotes"`
	PrevotesBitArray   string   `json:"prevotes_bit_array"`
	Precommits         []string `json:"precommits"`
	PrecommitsBitArray string   `json:"precommits_bit_array"`
}
