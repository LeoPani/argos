package service

import (
	"context"
	"fmt"

	"github.com/LeoPani/argos/backend/internal/domain"
)

// PoolStorage abstracts the pool repo.
type PoolStorage interface {
	Insert(ctx context.Context, p *domain.PatentPool) error
	GetByID(ctx context.Context, id int64) (*domain.PatentPool, error)
	List(ctx context.Context) ([]domain.PatentPool, error)
	Delete(ctx context.Context, id int64) error
	AddMember(ctx context.Context, poolID, patentID int64, sharePct float64) (*domain.PoolMember, error)
	RemoveMember(ctx context.Context, poolID, patentID int64) error
	ListMembers(ctx context.Context, poolID int64) ([]domain.PoolMember, error)
}

// PoolService orchestrates pool operations.
type PoolService struct{ repo PoolStorage }

func NewPoolService(repo PoolStorage) *PoolService { return &PoolService{repo: repo} }

func (s *PoolService) Create(ctx context.Context, p *domain.PatentPool) (*domain.PatentPool, error) {
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("pool: %w", err)
	}
	if err := s.repo.Insert(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *PoolService) GetByID(ctx context.Context, id int64) (*domain.PatentPool, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *PoolService) List(ctx context.Context) ([]domain.PatentPool, error) {
	return s.repo.List(ctx)
}

func (s *PoolService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *PoolService) AddPatent(ctx context.Context, poolID, patentID int64, sharePct float64) (*domain.PoolMember, error) {
	if sharePct < 0 || sharePct > 100 {
		return nil, fmt.Errorf("share_pct must be 0..100: %w", domain.ErrInvalidArg)
	}
	return s.repo.AddMember(ctx, poolID, patentID, sharePct)
}

func (s *PoolService) RemovePatent(ctx context.Context, poolID, patentID int64) error {
	return s.repo.RemoveMember(ctx, poolID, patentID)
}
