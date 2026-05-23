package domain

import "time"

// IPCCategory represents the predicted International Patent Classification
// category as returned by the BERTimbau classifier. Values 0..7 map to
// the eight top-level IPC sections the model was fine-tuned on.
type IPCCategory int

// IPCCategoryUnknown is the sentinel for "not yet classified" or
// "classification failed". The repository stores it as NULL.
const IPCCategoryUnknown IPCCategory = -1

// IsValid reports whether the category falls inside the model's
// output range. The repository should reject inserts where IsValid
// is false (or store NULL for IPCCategoryUnknown).
func (c IPCCategory) IsValid() bool {
	return c >= 0 && c <= 7
}

// PatentStatus tracks where a patent record is in the ingestion pipeline.
// It's mostly useful for the worker (which retries failed classifications)
// and for the dashboard (which can filter "show me only failed records").
type PatentStatus string

const (
	PatentStatusPending      PatentStatus = "pending"       // extracted from XML, not yet classified
	PatentStatusClassified   PatentStatus = "classified"    // AI classification succeeded
	PatentStatusFailed       PatentStatus = "failed"        // AI classification failed permanently
	PatentStatusReclassified PatentStatus = "reclassified"  // manually overridden by a human reviewer
)

// Patent is the core aggregate of the system: a single patent record
// extracted from an INPI RPI publication and (optionally) enriched by
// the AI classifier.
//
// JSON tags drive the public REST API; db tags document the corresponding
// PostgreSQL columns (used by the repository layer's row scanners).
type Patent struct {
	ID                int64        `json:"id"                 db:"id"`
	ApplicationNumber string       `json:"application_number" db:"application_number"` // unique, e.g. "BR102023001234-5"
	Title             string       `json:"title"              db:"title"`
	Abstract          string       `json:"abstract"           db:"abstract"`
	Applicant         string       `json:"applicant"          db:"applicant"`           // "depositante"
	Inventors         []string     `json:"inventors"          db:"inventors"`           // stored as TEXT[]
	FilingDate        *time.Time   `json:"filing_date"        db:"filing_date"`         // "data de depósito"
	PublicationDate   *time.Time   `json:"publication_date"   db:"publication_date"`    // RPI publication date
	IPCCategory       IPCCategory  `json:"ipc_category"       db:"ipc_category"`        // AI-predicted, 0..7 or -1
	IPCCode           string       `json:"ipc_code"           db:"ipc_code"`            // raw IPC code from INPI, if present
	RPIIssue          string       `json:"rpi_issue"          db:"rpi_issue"`           // e.g. "2750"
	Status            PatentStatus `json:"status"             db:"status"`
	CreatedAt         time.Time    `json:"created_at"         db:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"         db:"updated_at"`
}

// Validate performs cheap, synchronous checks on required fields before
// the repository attempts an insert. Returns an error wrapped with
// ErrInvalidArg so handlers can map it to HTTP 400.
func (p *Patent) Validate() error {
	switch {
	case p.ApplicationNumber == "":
		return wrapInvalid("application_number is required")
	case p.Title == "":
		return wrapInvalid("title is required")
	case p.Abstract == "":
		return wrapInvalid("abstract is required")
	}
	return nil
}

// PatentFilter encapsulates optional criteria for listing patents.
// All fields are optional; the zero value means "no filter on this field".
type PatentFilter struct {
	Category    *IPCCategory  // exact match if non-nil
	Status      PatentStatus  // exact match if non-empty
	Search      string        // ILIKE on title/abstract if non-empty
	RPIIssue    string        // exact match if non-empty
	FilingFrom  *time.Time
	FilingUntil *time.Time

	// Pagination
	Limit  int
	Offset int
}

// Normalize clamps user-supplied pagination into safe bounds. Always
// call this before passing a filter to the repository.
func (f *PatentFilter) Normalize() {
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
