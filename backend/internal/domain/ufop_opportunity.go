package domain

import "time"

// UFOPSource identifies which pipeline produced the opportunity record.
type UFOPSource string

const (
	UFOPSourceOAI    UFOPSource = "oai"    // repositorio.ufop.br OAI-PMH
	UFOPSourcePortal UFOPSource = "portal" // ufop.br/noticias HTML scraper
	UFOPSourceLens   UFOPSource = "lens"   // Lens.org affiliation filter
)

// UFOPOpportunityLevel represents the AI-assessed PI potential.
type UFOPOpportunityLevel string

const (
	UFOPLevelHigh   UFOPOpportunityLevel = "high"
	UFOPLevelMedium UFOPOpportunityLevel = "medium"
	UFOPLevelLow    UFOPOpportunityLevel = "low"
)

// UFOPOpportunityStatus tracks manual review lifecycle.
type UFOPOpportunityStatus string

const (
	UFOPStatusNew       UFOPOpportunityStatus = "new"
	UFOPStatusReviewed  UFOPOpportunityStatus = "reviewed"
	UFOPStatusConverted UFOPOpportunityStatus = "converted"  // became a patent consultation
	UFOPStatusDismissed UFOPOpportunityStatus = "dismissed"
)

// UFOPOpportunity is a publication / news item from UFOP that the AI
// pipeline identified as having PI potential.
type UFOPOpportunity struct {
	ID             int64                 `json:"id"`
	Source         UFOPSource            `json:"source"`
	ExternalID     string                `json:"external_id"`
	Title          string                `json:"title"`
	Authors        []string              `json:"authors"`
	Department     string                `json:"department"`
	Abstract       string                `json:"abstract"`
	URL            string                `json:"url"`
	PublishedAt    *time.Time            `json:"published_at"`

	// AI analysis
	IPCSuggestion  string               `json:"ipc_suggestion"`
	IPCCategory    IPCCategory          `json:"ipc_category"`
	Level          UFOPOpportunityLevel  `json:"opportunity_level"`
	SimilarityPct  int                  `json:"similarity_pct"`
	PIScore        float64              `json:"pi_score"`
	AIAnalysis     string               `json:"ai_analysis"`

	// Lifecycle
	Status         UFOPOpportunityStatus `json:"status"`
	PublicationID  *int64               `json:"publication_id,omitempty"`

	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

// UFOPFilter holds optional listing criteria.
type UFOPFilter struct {
	Source UFOPSource
	Level  UFOPOpportunityLevel
	Status UFOPOpportunityStatus
	Search string
	Limit  int
	Offset int
}

func (f *UFOPFilter) Normalize() {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 200 {
		f.Limit = 200
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
}
