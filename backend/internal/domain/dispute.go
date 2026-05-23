package domain

import "time"

// DisputeStatus tracks the lifecycle of an internal arbitration case.
type DisputeStatus string

const (
	DisputeStatusOpen      DisputeStatus = "open"
	DisputeStatusInReview  DisputeStatus = "in_review"
	DisputeStatusAwaiting  DisputeStatus = "awaiting_info"
	DisputeStatusResolved  DisputeStatus = "resolved"
	DisputeStatusWithdrawn DisputeStatus = "withdrawn"
	DisputeStatusEscalated DisputeStatus = "escalated"
)

// DisputeKind categorizes the nature of the IP conflict.
type DisputeKind string

const (
	DisputeKindTrademarkInfringement DisputeKind = "trademark_infringement"
	DisputeKindPatentInfringement    DisputeKind = "patent_infringement"
	DisputeKindAuthorship            DisputeKind = "authorship"
	DisputeKindLicensing             DisputeKind = "licensing"
	DisputeKindOther                 DisputeKind = "other"
)

// PartyRole identifies a participant's posture in a dispute.
type PartyRole string

const (
	PartyRoleClaimant   PartyRole = "claimant"
	PartyRoleRespondent PartyRole = "respondent"
	PartyRoleArbitrator PartyRole = "arbitrator"
	PartyRoleWitness    PartyRole = "witness"
	PartyRoleObserver   PartyRole = "observer"
)

// DisputeParty is a single participant in a dispute.
type DisputeParty struct {
	ID        int64     `json:"id"         db:"id"`
	DisputeID int64     `json:"dispute_id" db:"dispute_id"`
	Name      string    `json:"name"       db:"name"`
	Role      PartyRole `json:"role"       db:"role"`
	Email     string    `json:"email"      db:"email"`
	Document  string    `json:"document"   db:"document"`
	JoinedAt  time.Time `json:"joined_at"  db:"joined_at"`
}

// DisputeDocument is an attachment uploaded by one of the parties.
// HashSHA256 cross-links with Proof.Hash for blockchain timestamping (Phase 4).
type DisputeDocument struct {
	ID          int64     `json:"id"           db:"id"`
	DisputeID   int64     `json:"dispute_id"   db:"dispute_id"`
	UploadedBy  int64     `json:"uploaded_by"  db:"uploaded_by"`
	Title       string    `json:"title"        db:"title"`
	Description string    `json:"description"  db:"description"`
	StoragePath string    `json:"storage_path" db:"storage_path"`
	HashSHA256  string    `json:"hash_sha256"  db:"hash_sha256"`
	SizeBytes   int64     `json:"size_bytes"   db:"size_bytes"`
	MimeType    string    `json:"mime_type"    db:"mime_type"`
	UploadedAt  time.Time `json:"uploaded_at"  db:"uploaded_at"`
}

// DisputeEvent is one entry in the dispute's chronological timeline.
type DisputeEvent struct {
	ID         int64     `json:"id"          db:"id"`
	DisputeID  int64     `json:"dispute_id"  db:"dispute_id"`
	ActorID    *int64    `json:"actor_id"    db:"actor_id"`
	EventType  string    `json:"event_type"  db:"event_type"`
	Payload    string    `json:"payload"     db:"payload"`
	OccurredAt time.Time `json:"occurred_at" db:"occurred_at"`
}

// Dispute is the aggregate root for an arbitration case.
type Dispute struct {
	ID         int64         `json:"id"          db:"id"`
	CaseNumber string        `json:"case_number" db:"case_number"`
	Title      string        `json:"title"       db:"title"`
	Summary    string        `json:"summary"     db:"summary"`
	Kind       DisputeKind   `json:"kind"        db:"kind"`
	Status     DisputeStatus `json:"status"      db:"status"`

	PatentID    *int64 `json:"patent_id,omitempty"    db:"patent_id"`
	TrademarkID *int64 `json:"trademark_id,omitempty" db:"trademark_id"`

	Parties   []DisputeParty    `json:"parties,omitempty"   db:"-"`
	Documents []DisputeDocument `json:"documents,omitempty" db:"-"`
	Events    []DisputeEvent    `json:"events,omitempty"    db:"-"`

	OpenedAt   time.Time  `json:"opened_at"   db:"opened_at"`
	ResolvedAt *time.Time `json:"resolved_at" db:"resolved_at"`
	CreatedAt  time.Time  `json:"created_at"  db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"  db:"updated_at"`
}

// Validate checks the minimum fields required to open a dispute.
func (d *Dispute) Validate() error {
	switch {
	case d.CaseNumber == "":
		return wrapInvalid("case_number is required")
	case d.Title == "":
		return wrapInvalid("title is required")
	case d.Kind == "":
		return wrapInvalid("kind is required")
	}
	return nil
}

// DisputeFilter holds optional listing criteria.
type DisputeFilter struct {
	Status      DisputeStatus
	Kind        DisputeKind
	Search      string
	OpenedFrom  *time.Time
	OpenedUntil *time.Time

	Limit  int
	Offset int
}

// Normalize clamps pagination to safe bounds.
func (f *DisputeFilter) Normalize() {
	const (
		defaultLimit = 50
		maxLimit     = 200
	)
	if f.Limit <= 0 {
		f.Limit = defaultLimit
	}
	if f.Limit > maxLimit {
		f.Limit = maxLimit
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
}
