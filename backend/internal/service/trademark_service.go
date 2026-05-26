package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/repository"
)

// TrademarkService handles trademark business logic.
type TrademarkService struct {
	repo repository.TrademarkRepository
}

func NewTrademarkService(repo repository.TrademarkRepository) *TrademarkService {
	return &TrademarkService{repo: repo}
}

// Create validates and persists a new trademark.
func (s *TrademarkService) Create(ctx context.Context, t *domain.Trademark) (*domain.Trademark, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Insert(ctx, t); err != nil {
		if errors.Is(err, domain.ErrDuplicate) {
			return nil, fmt.Errorf("trademark %q already exists: %w", t.ProcessNumber, domain.ErrDuplicate)
		}
		return nil, fmt.Errorf("create trademark: %w", err)
	}
	return t, nil
}

// GetByID fetches a single trademark.
func (s *TrademarkService) GetByID(ctx context.Context, id int64) (*domain.Trademark, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get trademark: %w", err)
	}
	return t, nil
}

// List returns trademarks with filters and pagination.
func (s *TrademarkService) List(ctx context.Context, f domain.TrademarkFilter) ([]domain.Trademark, int64, error) {
	f.Normalize()

	items, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, fmt.Errorf("list trademarks: %w", err)
	}

	total, err := s.repo.Count(ctx, f)
	if err != nil {
		return nil, 0, fmt.Errorf("count trademarks: %w", err)
	}
	return items, total, nil
}
