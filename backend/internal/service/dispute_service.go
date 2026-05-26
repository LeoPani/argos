package service

import (
	"context"
	"fmt"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/repository"
)

// DisputeService handles arbitration logic.
type DisputeService struct {
	repo repository.DisputeRepository
}

func NewDisputeService(repo repository.DisputeRepository) *DisputeService {
	return &DisputeService{repo: repo}
}

func (s *DisputeService) Open(ctx context.Context, d *domain.Dispute) (*domain.Dispute, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Insert(ctx, d); err != nil {
		return nil, fmt.Errorf("open dispute: %w", err)
	}
	return d, nil
}

func (s *DisputeService) GetByID(ctx context.Context, id int64) (*domain.Dispute, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get dispute: %w", err)
	}
	return d, nil
}

func (s *DisputeService) List(ctx context.Context, f domain.DisputeFilter) ([]domain.Dispute, int64, error) {
	f.Normalize()
	items, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, fmt.Errorf("list disputes: %w", err)
	}
	total, err := s.repo.Count(ctx, f)
	if err != nil {
		return nil, 0, fmt.Errorf("count disputes: %w", err)
	}
	return items, total, nil
}

func (s *DisputeService) Transition(ctx context.Context, id int64, status domain.DisputeStatus) error {
	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *DisputeService) AddDocument(ctx context.Context, doc *domain.DisputeDocument) error {
	if doc.Title == "" {
		return domain.ErrInvalidArg
	}
	return s.repo.AddDocument(ctx, doc)
}
