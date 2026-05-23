package domain

import "time"

// TrademarkKind classifies how the mark is constituted, following the
// INPI's standard categories.
type TrademarkKind string

const (
	TrademarkKindNominative      TrademarkKind = "nominative"       // text only ("MAGAZINE LUIZA")
	TrademarkKindFigurative      TrademarkKind = "figurative"       // logo / image only
	TrademarkKindMixed           TrademarkKind = "mixed"            // text + image
	TrademarkKindThreeDimensional TrademarkKind = "three_dimensional"
)

// TrademarkStatus tracks the lifecycle of the trademark application
// inside the INPI process. Mirrors INPI's "situação" field.
type TrademarkStatus string

const (
	TrademarkStatusFiled     TrademarkStatus = "filed"      // depositada
	TrademarkStatusPublished TrademarkStatus = "published"  // publicada para oposição
	TrademarkStatusGranted   TrademarkStatus = "granted"    // registrada
	TrademarkStatusDenied    TrademarkStatus = "denied"     // indeferida
	TrademarkStatusArchived  TrademarkStatus = "archived"   // arquivada
	TrademarkStatusExpired   TrademarkStatus = "expired"    // extinta
)

// Trademark is the core aggregate for the INPI trademark domain. It feeds
// the prior-art search engine (Phase 3) and the future blockchain
// timestamping service (Phase 4).
type Trademark struct {
	ID              int64           `json:"id"                db:"id"`
	ProcessNumber   string          `json:"process_number"    db:"process_number"`   // "número do processo", unique
	Name            string          `json:"name"              db:"name"`             // textual element of the mark
	NormalizedName  string          `json:"normalized_name"   db:"normalized_name"`  // upper, no accents — for fuzzy match
	Kind            TrademarkKind   `json:"kind"              db:"kind"`
	Status          TrademarkStatus `json:"status"            db:"status"`
	Owner           string          `json:"owner"             db:"owner"`            // "titular / depositante"
	NiceClasses     []int           `json:"nice_classes"      db:"nice_classes"`     // Nice Classification, e.g. [9, 42]
	ImageURL        string          `json:"image_url"         db:"image_url"`        // figurative/mixed marks only
	FilingDate      *time.Time      `json:"filing_date"       db:"filing_date"`
	PublicationDate *time.Time      `json:"publication_date"  db:"publication_date"`
	GrantedDate     *time.Time      `json:"granted_date"      db:"granted_date"`
	RPIIssue        string          `json:"rpi_issue"         db:"rpi_issue"`
	CreatedAt       time.Time       `json:"created_at"        db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"        db:"updated_at"`
}

// Validate performs minimal field checks before insert.
func (t *Trademark) Validate() error {
	switch {
	case t.ProcessNumber == "":
		return wrapInvalid("process_number is required")
	case t.Name == "":
		return wrapInvalid("name is required")
	case t.Kind == "":
		return wrapInvalid("kind is required")
	}
	return nil
}

// TrademarkFilter holds optional listing criteria.
type TrademarkFilter struct {
	Kind        TrademarkKind   // exact match if non-empty
	Status      TrademarkStatus // exact match if non-empty
	NiceClass   *int            // matches if value is in NiceClasses array
	Search      string          // ILIKE on name / normalized_name
	FilingFrom  *time.Time
	FilingUntil *time.Time

	Limit  int
	Offset int
}

// Normalize clamps pagination to safe bounds.
func (f *TrademarkFilter) Normalize() {
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
