package types

import (
	"fmt"
	"ethereum/rpc-network/consensus/tbft/help"
	"time"
)

var (
//ErrInvalidBlockPartSignature is Error invalid block part signature
//ErrInvalidBlockPartSignature = errors.New("error invalid block part signature")
//ErrInvalidBlockPartHash is Error invalid block part hash
//ErrInvalidBlockPartHash = errors.New("error invalid block part hash")
)

// Proposal defines a block proposal for the consensus.
// It refers to the block only by its PartSetHeader.
// It must be signed by the correct proposer for the given Height/Round
// to be considered valid. It may depend on votes from a previous round,
// a so-called Proof-of-Lock (POL) round, as noted in the POLRound and POLBlockID.
type Proposal struct {
	Height           uint64        `json:"height"`
	Round            uint          `json:"round"`
	Timestamp        time.Time     `json:"timestamp"`
	BlockPartsHeader PartSetHeader `json:"block_parts_header"`
	POLRound         uint          `json:"pol_round"`    // -1 if null.
	POLBlockID       BlockID       `json:"pol_block_id"` // zero if null.
	Signature        []byte        `json:"signature"`
}

// NewProposal returns a new Proposal.
// If there is no POLRound, polRound should be -1.
func NewProposal(height uint64, round int, blockPartsHeader PartSetHeader, polRound uint, polBlockID BlockID) *Proposal {
	return &Proposal{
		Height:           height,
		Round:            uint(round),
		Timestamp:        time.Now().UTC(),
		BlockPartsHeader: blockPartsHeader,
		POLRound:         polRound,
		POLBlockID:       polBlockID,
	}
}

// String returns a string representation of the Proposal.
func (p *Proposal) String() string {
	return fmt.Sprintf("Proposal{%v/%v %v (%v,%v) %X @ %s}",
		p.Height, p.Round, p.BlockPartsHeader, p.POLRound,
		p.POLBlockID,
		help.Fingerprint(p.Signature), CanonicalTime(p.Timestamp))
}

// SignBytes returns the Proposal bytes for signing
func (p *Proposal) SignBytes(chainID string) []byte {
	bz, err := cdc.MarshalJSON(CanonicalProposal(chainID, p))
	if err != nil {
		panic(err)
	}
	signBytes := help.RlpHash([]interface{}{bz})
	return signBytes[:]
}
