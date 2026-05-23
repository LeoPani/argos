package ai

import "context"

// Classifier is the narrow capability interface satisfied by the BERT
// adapter. Defining it here (consumer side) follows Go's "accept
// interfaces, return structs" idiom and lets the test suite inject a
// mock without depending on the bert package.
type Classifier interface {
	ClassifyPatent(ctx context.Context, abstract string) (int, error)
}

// Generator is the narrow capability interface satisfied by the LLM
// adapter (Phase 3+). Same dependency-inversion rationale as Classifier.
type Generator interface {
	GeneratePriorArtReport(ctx context.Context, patentID string) (string, error)
	SummarizeDispute(ctx context.Context, disputeID string) (string, error)
}

// Composite implements AIService by routing each method to the adapter
// best suited to handle it. It is the only place in the codebase that
// knows BERT does classification and the LLM does generation; everything
// upstream sees a single interface.
type Composite struct {
	classifier Classifier
	generator  Generator
}

// NewComposite assembles an AIService from a classifier and a generator.
// Either may be nil during early development; the corresponding methods
// will then return ErrNotImplemented.
//
// Typical wiring in main.go:
//
//	bertClient := bert.New(bert.DefaultConfig(cfg.AIBertURL))
//	llmClient  := llm.New(llm.DefaultConfig())   // stub for now
//	aiSvc      := ai.NewComposite(bertClient, llmClient)
func NewComposite(classifier Classifier, generator Generator) *Composite {
	return &Composite{classifier: classifier, generator: generator}
}

// ClassifyPatent routes to the BERT classifier.
func (c *Composite) ClassifyPatent(ctx context.Context, abstract string) (int, error) {
	if c.classifier == nil {
		return -1, ErrNotImplemented
	}
	return c.classifier.ClassifyPatent(ctx, abstract)
}

// GeneratePriorArtReport routes to the LLM generator.
func (c *Composite) GeneratePriorArtReport(ctx context.Context, patentID string) (string, error) {
	if c.generator == nil {
		return "", ErrNotImplemented
	}
	return c.generator.GeneratePriorArtReport(ctx, patentID)
}

// SummarizeDispute routes to the LLM generator.
func (c *Composite) SummarizeDispute(ctx context.Context, disputeID string) (string, error) {
	if c.generator == nil {
		return "", ErrNotImplemented
	}
	return c.generator.SummarizeDispute(ctx, disputeID)
}

// Compile-time check that Composite satisfies the AIService interface.
// If you ever break the contract, the build will fail right here.
var _ AIService = (*Composite)(nil)
