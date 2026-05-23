package domain

import "time"

// PublicationSource identifies which external database supplied a record.
// We keep a single Publication entity for all sources and discriminate
// at read time; this avoids a combinatorial explosion of tables for what
// is, conceptually, the same thing.
type PublicationSource string

const (
	PublicationSourceLens          PublicationSource = "lens"            // Lens.org
	PublicationSourceWebOfScience  PublicationSource = "web_of_science"  // Clarivate WoS
	PublicationSourceScielo        PublicationSource = "scielo"          // future
	PublicationSourceManual        PublicationSource = "manual"          // user-uploaded
)

// PublicationKind distinguishes the document type.
type PublicationKind string

const (
	PublicationKindArticle    PublicationKind = "article"
	PublicationKindReview     PublicationKind = "review"
	PublicationKindConference PublicationKind = "conference"
	PublicationKindBook       PublicationKind = "book"
	PublicationKindThesis     PublicationKind = "thesis"
	PublicationKindPreprint   PublicationKind = "preprint"
	PublicationKindOther      PublicationKind = "other"
)

// Publication is the unified record for any scientific publication
// imported from an external bibliographic database.
type Publication struct {
	ID             int64             `json:"id"               db:"id"`
	Source         PublicationSource `json:"source"           db:"source"`
	ExternalID     string            `json:"external_id"      db:"external_id"`  // (source, external_id) is UNIQUE
	DOI            string            `json:"doi"              db:"doi"`
	Title          string            `json:"title"            db:"title"`
	Abstract       string            `json:"abstract"         db:"abstract"`
	Authors        []string          `json:"authors"          db:"authors"`      // TEXT[]
	Affiliations   []string          `json:"affiliations"     db:"affiliations"` // TEXT[]
	Kind           PublicationKind   `json:"kind"             db:"kind"`
	Journal        string            `json:"journal"          db:"journal"`
	PublishedDate  *time.Time        `json:"published_date"   db:"published_date"`
	CitationCount  int               `json:"citation_count"   db:"citation_count"`
	Keywords       []string          `json:"keywords"         db:"keywords"`     // TEXT[]
	URL            string            `json:"url"              db:"url"`
	IPCCategory    IPCCategory       `json:"ipc_category"     db:"ipc_category"` // for cross-domain analytics
	CreatedAt      time.Time         `json:"created_at"       db:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"       db:"updated_at"`
}

// Validate checks required fields.
func (p *Publication) Validate() error {
	switch {
	case p.Source == "":
		return wrapInvalid("source is required")
	case p.ExternalID == "":
		return wrapInvalid("external_id is required")
	case p.Title == "":
		return wrapInvalid("title is required")
	}
	return nil
}

// PublicationFilter holds optional listing criteria.
type PublicationFilter struct {
	Source       PublicationSource // exact match if non-empty
	Kind         PublicationKind   // exact match if non-empty
	Category     *IPCCategory      // exact match if non-nil
	Search       string            // ILIKE on title / abstract
	Author       string            // ILIKE inside authors array
	Journal      string            // ILIKE on journal
	PublishedFrom  *time.Time
	PublishedUntil *time.Time

	Limit  int
	Offset int
}

// Normalize clamps pagination to safe bounds.
func (f *PublicationFilter) Normalize() {
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
