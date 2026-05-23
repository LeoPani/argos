// Package ai defines the AIService port that the rest of the application
// uses to talk to AI models. Concrete adapters live in subpackages:
//
//   - bert/  Fast IPC classification via the FastAPI BERTimbau service.
//   - llm/   Long-form generation via an LLM (Ollama / GPT-4 / Claude).
//
// The package itself contains no infrastructure code — only the interface,
// the narrow capability interfaces (Classifier, Generator), and the
// Composite adapter that wires them together.
//
// This is the central pillar of the Hybrid AI Architecture: Phase 1 uses
// the BERT adapter; Phases 3 and 5 plug in the LLM adapter without
// touching any other layer of the application.
package ai

import (
	"context"
	"errors"
)

// ErrNotImplemented marks methods that are declared in the AIService
// interface but not yet implemented in the current adapter chain.
// Phase 1 returns this from GeneratePriorArtReport and SummarizeDispute.
//
// Callers should use errors.Is(err, ai.ErrNotImplemented) to detect this
// case and render a friendly "coming soon" message to the user.
var ErrNotImplemented = errors.New("ai: feature not implemented in current adapter")

// AIService is the single port through which application code interacts
// with any AI capability. It bundles two concerns:
//
//  1. Fast, structured classification (BERT-backed, milliseconds).
//  2. Slow, free-text generation (LLM-backed, seconds-to-minutes).
//
// Implementations MUST:
//   - Honor context cancellation/deadlines on every call.
//   - Return wrapped errors callers can inspect with errors.Is/As.
//   - Be safe for concurrent use by multiple goroutines.
type AIService interface {
	// ClassifyPatent sends a patent abstract to the classifier and returns
	// the predicted IPC category id (0..7). Phase 1: BERTimbau via FastAPI.
	ClassifyPatent(ctx context.Context, abstract string) (int, error)

	// GeneratePriorArtReport asks an LLM to write a full prior-art report
	// for the given patent. Phase 3 implementation. Returns ErrNotImplemented
	// until the LLM adapter is wired in.
	GeneratePriorArtReport(ctx context.Context, patentID string) (string, error)

	// SummarizeDispute asks an LLM to produce a structured summary of an
	// arbitration dispute. Phase 5 implementation. Returns ErrNotImplemented
	// until the LLM adapter is wired in.
	SummarizeDispute(ctx context.Context, disputeID string) (string, error)
}
