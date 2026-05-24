package domain

import (
	"encoding/json"
	"time"
)

// SubjectKind classifies what's being arbitrated.
type SubjectKind string

const (
	SubjectKindTrademark SubjectKind = "trademark"
	SubjectKindPatent    SubjectKind = "patent"
	SubjectKindInventor  SubjectKind = "inventor"
	SubjectKindOther     SubjectKind = "other"
)

// DisputeSubject is one item being arbitrated (a trademark, a patent,
// a candidate inventor, etc.). A dispute may have many subjects that
// compete with each other.
type DisputeSubject struct {
	ID         int64           `json:"id"`
	DisputeID  int64           `json:"dispute_id"`
	Kind       SubjectKind     `json:"kind"`
	RefID      *int64          `json:"ref_id,omitempty"` // FK to trademarks/patents
	Label      string          `json:"label"`
	PartyID    *int64          `json:"party_id,omitempty"`
	Metadata   json.RawMessage `json:"metadata"`
	CreatedAt  time.Time       `json:"created_at"`
}

// VerdictMethod identifies how the verdict was produced.
type VerdictMethod string

const (
	VerdictMethodHeuristic VerdictMethod = "heuristic_v1"
	VerdictMethodClaude    VerdictMethod = "claude_v1"
	VerdictMethodHybrid    VerdictMethod = "hybrid"
)

// ArbitrationVerdict is the AI analysis output for one dispute run.
//
// Reasoning is a structured JSONB with per-subject scoring and
// human-readable bullet points. The frontend renders it as a report.
type ArbitrationVerdict struct {
	ID              int64           `json:"id"`
	DisputeID       int64           `json:"dispute_id"`
	WinnerSubjectID *int64          `json:"winner_subject_id,omitempty"`
	Confidence      int             `json:"confidence"` // 0..100
	Method          VerdictMethod   `json:"method"`
	Summary         string          `json:"summary"`
	Reasoning       json.RawMessage `json:"reasoning"`
	CreatedAt       time.Time       `json:"created_at"`
}

// SubjectScore is the per-subject portion of a verdict's reasoning.
// Stored inside ArbitrationVerdict.Reasoning as JSON.
type SubjectScore struct {
	SubjectID    int64    `json:"subject_id"`
	Label        string   `json:"label"`
	Score        float64  `json:"score"` // 0..100 — higher = stronger claim
	ProArguments []string `json:"pro"`
	ConArguments []string `json:"con"`
}

// VerdictReasoning is the JSON structure stored in arbitration_verdicts.reasoning.
type VerdictReasoning struct {
	Subjects []SubjectScore `json:"subjects"`
	Factors  []string       `json:"factors"` // textual narrative of what the AI considered
}
