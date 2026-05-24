// Package service — WatchlistService manages alert subscriptions and
// runs the matching engine that scans patents+trademarks for new matches.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/repository"
)

// WatchlistService manages user-defined alerts.
type WatchlistService struct {
	repo          repository.WatchlistRepository
	patentRepo    repository.PatentRepository
	trademarkRepo repository.TrademarkRepository
}

// NewWatchlistService wires up dependencies.
func NewWatchlistService(
	repo repository.WatchlistRepository,
	patentRepo repository.PatentRepository,
	trademarkRepo repository.TrademarkRepository,
) *WatchlistService {
	return &WatchlistService{
		repo:          repo,
		patentRepo:    patentRepo,
		trademarkRepo: trademarkRepo,
	}
}

// Create validates and persists a new watchlist.
func (s *WatchlistService) Create(ctx context.Context, w *domain.Watchlist) (*domain.Watchlist, error) {
	if err := w.Validate(); err != nil {
		return nil, fmt.Errorf("watchlist: %w", err)
	}
	if err := s.repo.Insert(ctx, w); err != nil {
		return nil, fmt.Errorf("watchlist create: %w", err)
	}
	return w, nil
}

// List returns all watchlists.
func (s *WatchlistService) List(ctx context.Context) ([]domain.Watchlist, error) {
	return s.repo.List(ctx)
}

// Delete removes a watchlist by ID.
func (s *WatchlistService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// Check runs the matching engine for a single watchlist:
// counts patents+trademarks matching the query that were created after
// the last check, then updates the watchlist with the new count.
func (s *WatchlistService) Check(ctx context.Context, id int64) (*domain.Watchlist, error) {
	w, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	matches, err := s.runMatch(ctx, w)
	if err != nil {
		return nil, fmt.Errorf("watchlist check: %w", err)
	}

	status := domain.WatchStatusOK
	if matches > 0 {
		status = domain.WatchStatusAlert
	}

	if err := s.repo.UpdateCheck(ctx, id, matches, status); err != nil {
		return nil, fmt.Errorf("watchlist update check: %w", err)
	}

	now := time.Now()
	w.LastCheck = &now
	w.NewCount = matches
	w.Status = status
	return w, nil
}

// CheckAll runs Check on every watchlist; non-fatal errors are skipped.
func (s *WatchlistService) CheckAll(ctx context.Context) (int, error) {
	all, err := s.repo.List(ctx)
	if err != nil {
		return 0, err
	}
	ok := 0
	for _, w := range all {
		if _, err := s.Check(ctx, w.ID); err == nil {
			ok++
		}
	}
	return ok, nil
}

// runMatch counts how many patents and trademarks created since LastCheck
// match the watchlist query.
func (s *WatchlistService) runMatch(ctx context.Context, w *domain.Watchlist) (int, error) {
	count := 0

	// Patents: match title or abstract for term/company/patent watches.
	if w.Type != domain.WatchTypeBrand {
		patents, err := s.patentRepo.List(ctx, domain.PatentFilter{Search: w.Query, Limit: 200})
		if err == nil {
			for _, p := range patents {
				if w.LastCheck == nil || p.CreatedAt.After(*w.LastCheck) {
					count++
				}
			}
		}
	}

	// Trademarks: match name for term/brand/company watches.
	if w.Type != domain.WatchTypePatent {
		marks, err := s.trademarkRepo.List(ctx, domain.TrademarkFilter{Search: w.Query, Limit: 200})
		if err == nil {
			for _, m := range marks {
				if w.LastCheck == nil || m.CreatedAt.After(*w.LastCheck) {
					count++
				}
			}
		}
	}

	return count, nil
}
