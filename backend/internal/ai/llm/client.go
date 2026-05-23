// Package llm is the (currently stubbed) adapter for the Large Language
// Model. Phase 1 ships only with stubs; the real implementation lands in
// Phase 3 (Prior Art Reports) and Phase 5 (Dispute Summaries).
//
// Design goal: when we wire the LLM in later, ONLY this file changes.
// The rest of the codebase already talks through ai.AIService and will
// pick up the new behavior with zero changes.
//
// Target backends (any one of):
//   - Ollama running LLaMA 3 / Mistral locally  (cheap, private)
//   - OpenAI Chat Completions                    (high quality, paid)
//   - Anthropic Messages API                     (high quality, paid)
//
// All three speak HTTP/JSON, so a single Client with a Provider switch
// is plenty.
package llm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/LeoPani/argos/backend/internal/ai"
)

// Provider identifies which backing LLM to call. Phase 3 will branch
// on this inside the real implementations.
type Provider string

const (
	ProviderOllama    Provider = "ollama"
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
)

// Config bundles every field a future implementation could need.
// None are consumed yet — they exist so .env wiring lands cleanly later.
type Config struct {
	Provider       Provider
	BaseURL        string        // e.g. "http://localhost:11434" for Ollama
	APIKey         string        // managed providers only
	Model          string        // "llama3" | "gpt-4o" | "claude-sonnet-4-5"
	RequestTimeout time.Duration // long: report generation is slow
	MaxTokens      int
}

// DefaultConfig returns a sensible Ollama-local default. Override every
// field from environment variables before flipping the wiring in main.go.
func DefaultConfig() Config {
	return Config{
		Provider:       ProviderOllama,
		BaseURL:        "http://localhost:11434",
		Model:          "llama3",
		RequestTimeout: 120 * time.Second,
		MaxTokens:      2048,
	}
}

// Client is the LLM adapter. Today every method returns ErrNotImplemented;
// Phase 3+ will fill in the bodies without changing the signatures.
type Client struct {
	cfg Config
}

// New builds a stub Client. It performs NO connectivity validation —
// there is nothing to connect to yet.
func New(cfg Config) *Client {
	return &Client{cfg: cfg}
}

// GeneratePriorArtReport will, in Phase 3, prompt an LLM to produce a
// structured prior-art report grounded on data fetched from the patent
// repository and Lens.org. For now it returns ErrNotImplemented so the
// service layer can surface a clear "coming soon" message.
func (c *Client) GeneratePriorArtReport(ctx context.Context, patentID string) (string, error) {
	if patentID == "" {
		return "", fmt.Errorf("llm: prior art: empty patentID")
	}
	return "", errors.Join(
		ai.ErrNotImplemented,
		fmt.Errorf("LLM provider %q wiring deferred to Phase 3", c.cfg.Provider),
	)
}

// SummarizeDispute will, in Phase 5, read the dispute timeline and
// produce a structured summary. Stub for now.
func (c *Client) SummarizeDispute(ctx context.Context, disputeID string) (string, error) {
	if disputeID == "" {
		return "", fmt.Errorf("llm: summarize: empty disputeID")
	}
	return "", errors.Join(
		ai.ErrNotImplemented,
		fmt.Errorf("LLM provider %q wiring deferred to Phase 5", c.cfg.Provider),
	)
}
