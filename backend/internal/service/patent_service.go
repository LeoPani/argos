// Package service holds the use-case layer: the orchestrators that
// coordinate domain logic, AI calls, and persistence to fulfill a
// business operation.
//
// Services depend on INTERFACES (ai.AIService, repository.PatentRepository),
// never on concrete adapters. This keeps the service layer trivially
// testable with mocks.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/LeoPani/argos/backend/internal/ai"
	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/repository"
)

// PatentService orchestrates patent use cases:
//   - Ingest: validate → classify (BERT) → persist
//   - List:   normalize filter → count + page
//   - GetByID / GetByApplicationNumber: pass-through with sentinel errors
type PatentService struct {
	repo  repository.PatentRepository
	aiSvc ai.AIService
}

// NewPatentService wires the dependencies. Both MUST be non-nil.
func NewPatentService(repo repository.PatentRepository, aiSvc ai.AIService) *PatentService {
	return &PatentService{repo: repo, aiSvc: aiSvc}
}

// Ingest is the canonical write path for patents:
//
//  1. Domain validation (cheap, sync).
//  2. AI classification of the abstract. Failure is tolerated:
//     the patent is saved with status=failed so a future worker run
//     can re-classify (e.g. after the FastAPI service comes back up).
//  3. Persist via the repository.
//
// Returns the inserted patent (with ID + timestamps populated).
func (s *PatentService) Ingest(ctx context.Context, p *domain.Patent) (*domain.Patent, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	// Try to classify the abstract. The AI service can be unavailable
	// (Phase 1 only has BERT wired up; LLM features still stub out).
	// We DON'T fail the whole ingestion if classification fails — we
	// just mark the row as "failed" and the worker retries later.
	category, err := s.aiSvc.ClassifyPatent(ctx, p.Abstract)
	if err != nil {
		slog.Warn("classification failed; saving as failed",
			"application_number", p.ApplicationNumber, "err", err)
		p.IPCCategory = domain.IPCCategoryUnknown
		p.Status = domain.PatentStatusFailed
	} else {
		p.IPCCategory = domain.IPCCategory(category)
		p.Status = domain.PatentStatusClassified
	}

	if err := s.repo.Insert(ctx, p); err != nil {
		return nil, fmt.Errorf("ingest patent: %w", err)
	}
	return p, nil
}

// List returns a page of patents plus the total count for pagination
// metadata. The filter is normalized inside this method so handlers
// don't have to remember.
func (s *PatentService) List(ctx context.Context, f domain.PatentFilter) ([]domain.Patent, int64, error) {
	f.Normalize()

	total, err := s.repo.Count(ctx, f)
	if err != nil {
		return nil, 0, fmt.Errorf("list patents: count: %w", err)
	}

	items, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, fmt.Errorf("list patents: %w", err)
	}
	return items, total, nil
}

// GetByID is a pass-through. Returns domain.ErrNotFound for unknown ids.
func (s *PatentService) GetByID(ctx context.Context, id int64) (*domain.Patent, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		// Translate any non-sentinel error; preserve sentinels for handlers.
		if errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("get patent: %w", err)
	}
	return p, nil
}

// GetByApplicationNumber is used by both handlers and the ingestion
// worker (for deduplication).
func (s *PatentService) GetByApplicationNumber(ctx context.Context, appNum string) (*domain.Patent, error) {
	p, err := s.repo.GetByApplicationNumber(ctx, appNum)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("get patent: %w", err)
	}
	return p, nil
}
