package domain

import "time"

// ProofStatus tracks where a timestamping proof sits in its lifecycle.
//
//   pending  -> hash stored, not yet sent to blockchain
//   batched  -> included in a Merkle root currently in-flight to chain
//   confirmed -> on-chain transaction included in a block
//   failed   -> transaction failed permanently; manual review needed
type ProofStatus string

const (
	ProofStatusPending   ProofStatus = "pending"
	ProofStatusBatched   ProofStatus = "batched"
	ProofStatusConfirmed ProofStatus = "confirmed"
	ProofStatusFailed    ProofStatus = "failed"
)

// ProofKind tags what kind of document the proof certifies.
type ProofKind string

const (
	ProofKindPriorArtReport ProofKind = "prior_art_report"
	ProofKindTrademarkBrief ProofKind = "trademark_brief"
	ProofKindDisputeDoc     ProofKind = "dispute_document"
	ProofKindGeneric        ProofKind = "generic"
)

// Proof represents a single timestamping record. The user uploads a
// document, the frontend computes its SHA-256, and Argos stores the hash
// here. Phase 4 will add a worker that batches pending proofs into a
// Merkle tree and writes the root to Polygon.
type Proof struct {
	ID         int64       `json:"id"          db:"id"`
	Kind       ProofKind   `json:"kind"        db:"kind"`
	DocumentID string      `json:"document_id" db:"document_id"`   // free-form: patent id, dispute id, etc.
	Hash       string      `json:"hash"        db:"hash"`          // hex-encoded SHA-256, 64 chars
	Status     ProofStatus `json:"status"      db:"status"`

	// Blockchain fields, populated by the worker once the proof is
	// included in a batch and the batch is confirmed on-chain.
	BatchID         string     `json:"batch_id,omitempty"          db:"batch_id"`          // local batch UUID
	MerkleRoot      string     `json:"merkle_root,omitempty"       db:"merkle_root"`      // root of the batch's Merkle tree
	MerkleProof     []string   `json:"merkle_proof,omitempty"      db:"merkle_proof"`     // sibling hashes, for verification
	BlockchainTxID  string     `json:"blockchain_tx_id,omitempty"  db:"blockchain_tx_id"` // e.g. Polygon tx hash
	BlockchainBlock int64      `json:"blockchain_block,omitempty"  db:"blockchain_block"` // block number
	ConfirmedAt     *time.Time `json:"confirmed_at,omitempty"      db:"confirmed_at"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Validate checks that the proof carries the minimum fields needed
// for it to ever be included in a Merkle batch.
func (p *Proof) Validate() error {
	switch {
	case p.DocumentID == "":
		return wrapInvalid("document_id is required")
	case p.Hash == "":
		return wrapInvalid("hash is required")
	case len(p.Hash) != 64:
		return wrapInvalid("hash must be a 64-character hex SHA-256")
	case p.Kind == "":
		return wrapInvalid("kind is required")
	}
	return nil
}
