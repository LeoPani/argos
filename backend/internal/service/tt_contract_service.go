package service

import (
	"context"
	"fmt"

	"github.com/LeoPani/argos/backend/internal/domain"
)

// TTContractStorage abstracts the repo for tests.
type TTContractStorage interface {
	Insert(ctx context.Context, c *domain.TTContract) error
	GetByID(ctx context.Context, id int64) (*domain.TTContract, error)
	List(ctx context.Context, f domain.TTContractFilter) ([]domain.TTContract, error)
	Count(ctx context.Context, f domain.TTContractFilter) (int64, error)
	Delete(ctx context.Context, id int64) error
	UpdateStatus(ctx context.Context, id int64, status domain.ContractStatus) error
}

// TTContractService orchestrates TT contract operations.
type TTContractService struct{ repo TTContractStorage }

func NewTTContractService(repo TTContractStorage) *TTContractService {
	return &TTContractService{repo: repo}
}

// Create validates and persists a new contract.
func (s *TTContractService) Create(ctx context.Context, c *domain.TTContract) (*domain.TTContract, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("tt_contract: %w", err)
	}
	if err := s.repo.Insert(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *TTContractService) GetByID(ctx context.Context, id int64) (*domain.TTContract, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *TTContractService) List(ctx context.Context, f domain.TTContractFilter) ([]domain.TTContract, int64, error) {
	f.Normalize()
	items, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.Count(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *TTContractService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *TTContractService) UpdateStatus(ctx context.Context, id int64, status domain.ContractStatus) error {
	switch status {
	case domain.ContractDraft, domain.ContractNegotiating,
		domain.ContractActive, domain.ContractExpired, domain.ContractTerminated:
	default:
		return fmt.Errorf("invalid status %q: %w", status, domain.ErrInvalidArg)
	}
	return s.repo.UpdateStatus(ctx, id, status)
}
