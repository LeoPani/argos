// Package service — WatchlistService manages alert subscriptions and
// runs the matching engine that scans patents+trademarks for new matches.
package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/repository"
)

// WatchlistService manages user-defined alerts.
type WatchlistService struct {
	repo          repository.WatchlistRepository
	patentRepo    repository.PatentRepository
	trademarkRepo repository.TrademarkRepository
	disputeRepo   repository.DisputeRepository
}

// NewWatchlistService wires up dependencies. disputeRepo is optional;
// pass nil to disable the auto-dispute draft creation feature.
func NewWatchlistService(
	repo repository.WatchlistRepository,
	patentRepo repository.PatentRepository,
	trademarkRepo repository.TrademarkRepository,
	disputeRepo repository.DisputeRepository,
) *WatchlistService {
	return &WatchlistService{
		repo:          repo,
		patentRepo:    patentRepo,
		trademarkRepo: trademarkRepo,
		disputeRepo:   disputeRepo,
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
//
// If w.AutoDispute is true and a match exceeds w.SimilarityThreshold,
// a draft dispute is auto-created for human review.
func (s *WatchlistService) Check(ctx context.Context, id int64) (*domain.Watchlist, error) {
	w, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	matches, autoDrafts, err := s.runMatch(ctx, w)
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

	// Persist auto-drafted disputes if any (best-effort, non-fatal)
	if w.AutoDispute && s.disputeRepo != nil {
		for _, draft := range autoDrafts {
			_ = s.disputeRepo.Insert(ctx, draft)
		}
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
// match the watchlist query, and prepares auto-draft disputes for matches
// above the similarity threshold (when w.AutoDispute is true).
func (s *WatchlistService) runMatch(
	ctx context.Context, w *domain.Watchlist,
) (count int, autoDrafts []*domain.Dispute, err error) {

	// Patents: match title or abstract for term/company/patent watches.
	if w.Type != domain.WatchTypeBrand {
		patents, perr := s.patentRepo.List(ctx, domain.PatentFilter{Search: w.Query, Limit: 200})
		if perr == nil {
			for _, p := range patents {
				if w.LastCheck == nil || p.CreatedAt.After(*w.LastCheck) {
					count++
					if w.AutoDispute {
						sim := watchSimilarity(w.Query, p.Title+" "+p.Abstract)
						if sim >= w.SimilarityThreshold {
							pid := p.ID
							autoDrafts = append(autoDrafts, &domain.Dispute{
								CaseNumber: fmt.Sprintf("WL-%d-PT-%d-%d",
									w.ID, p.ID, time.Now().UnixNano()%100000),
								Title: fmt.Sprintf(
									"[Auto] Watchlist %q detectou patente similar: %s",
									w.Label, p.ApplicationNumber),
								Summary: fmt.Sprintf(
									"Watchlist '%s' (query: %s) encontrou patente '%s' com %d%% de similaridade. "+
										"Revisar e decidir se é caso real de infração ou prior art.",
									w.Label, w.Query, p.Title, sim),
								Kind:     domain.DisputeKindPatentInfringement,
								Status:   domain.DisputeStatusOpen,
								PatentID: &pid,
							})
						}
					}
				}
			}
		}
	}

	// Trademarks: match name for term/brand/company watches.
	if w.Type != domain.WatchTypePatent {
		marks, terr := s.trademarkRepo.List(ctx, domain.TrademarkFilter{Search: w.Query, Limit: 200})
		if terr == nil {
			for _, m := range marks {
				if w.LastCheck == nil || m.CreatedAt.After(*w.LastCheck) {
					count++
					if w.AutoDispute {
						sim := watchSimilarity(w.Query, m.Name+" "+m.NormalizedName)
						if sim >= w.SimilarityThreshold {
							tid := m.ID
							autoDrafts = append(autoDrafts, &domain.Dispute{
								CaseNumber: fmt.Sprintf("WL-%d-TM-%d-%d",
									w.ID, m.ID, time.Now().UnixNano()%100000),
								Title: fmt.Sprintf(
									"[Auto] Watchlist %q detectou marca similar: %s",
									w.Label, m.ProcessNumber),
								Summary: fmt.Sprintf(
									"Watchlist '%s' (query: %s) encontrou marca '%s' com %d%% de similaridade.",
									w.Label, w.Query, m.Name, sim),
								Kind:        domain.DisputeKindTrademarkInfringement,
								Status:      domain.DisputeStatusOpen,
								TrademarkID: &tid,
							})
						}
					}
				}
			}
		}
	}

	return count, autoDrafts, nil
}

// watchSimilarity — Jaccard de palavras simples (0-100).
func watchSimilarity(query, text string) int {
	q := splitWordSet(query)
	t := splitWordSet(text)
	if len(q) == 0 || len(t) == 0 {
		return 0
	}
	common := 0
	for w := range q {
		if t[w] {
			common++
		}
	}
	// Use containment (common / |query|) — favorece quando todas as
	// palavras da query estão presentes, mesmo em texto longo.
	return common * 100 / len(q)
}

func splitWordSet(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.Fields(strings.ToLower(s)) {
		if len(w) > 2 {
			out[w] = true
		}
	}
	return out
}
