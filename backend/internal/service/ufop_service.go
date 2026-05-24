// Package service — UFOPService orchestrates the UFOP Intelligence pipeline.
//
// Lifecycle:
//
//	1. HarvestOAI   → OAIClient.Harvest → PublicationRepo.Upsert → Analyzer.Analyze → UFOPRepo.Upsert
//	2. ScrapePortal → PortalScraper.ScrapeNews → Analyzer.Analyze → UFOPRepo.Upsert
//
// The handler layer only calls List, Count, GetByID, and UpdateStatus.
// Harvest and Scrape are invoked by the background worker (cmd/worker).
package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/repository"
	"github.com/LeoPani/argos/backend/internal/worker/ufop"
)

// UFOPService manages UFOP opportunities end-to-end.
type UFOPService struct {
	repo       repository.UFOPRepository
	pubRepo    repository.PublicationRepository
	oai        *ufop.OAIClient
	portal     *ufop.PortalScraper
	analyzer   *ufop.Analyzer
	log        *slog.Logger
}

// HarvestStats is returned by Harvest* methods.
type HarvestStats struct {
	Fetched    int
	Upserted   int
	Skipped    int
	Errors     int
}

// NewUFOPService wires up the UFOP service.
func NewUFOPService(
	repo repository.UFOPRepository,
	pubRepo repository.PublicationRepository,
	oai *ufop.OAIClient,
	portal *ufop.PortalScraper,
	analyzer *ufop.Analyzer,
	log *slog.Logger,
) *UFOPService {
	return &UFOPService{
		repo:     repo,
		pubRepo:  pubRepo,
		oai:      oai,
		portal:   portal,
		analyzer: analyzer,
		log:      log,
	}
}

// ─── Query methods (used by HTTP handler) ─────────────────────────────────────

// List returns opportunities matching the filter, plus the total count.
func (s *UFOPService) List(ctx context.Context, f domain.UFOPFilter) ([]domain.UFOPOpportunity, int64, error) {
	f.Normalize()
	items, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, fmt.Errorf("ufop service list: %w", err)
	}
	total, err := s.repo.Count(ctx, f)
	if err != nil {
		return nil, 0, fmt.Errorf("ufop service count: %w", err)
	}
	return items, total, nil
}

// GetByID fetches a single opportunity.
func (s *UFOPService) GetByID(ctx context.Context, id int64) (*domain.UFOPOpportunity, error) {
	opp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ufop service get: %w", err)
	}
	return opp, nil
}

// UpdateStatus transitions the review lifecycle of an opportunity.
func (s *UFOPService) UpdateStatus(ctx context.Context, id int64, status domain.UFOPOpportunityStatus) error {
	switch status {
	case domain.UFOPStatusNew, domain.UFOPStatusReviewed,
		domain.UFOPStatusConverted, domain.UFOPStatusDismissed:
		// valid
	default:
		return fmt.Errorf("%w: unknown status %q", domain.ErrInvalidArg, status)
	}
	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return fmt.Errorf("ufop service update status: %w", err)
	}
	return nil
}

// ─── Harvest methods (used by background worker) ──────────────────────────────

// HarvestOAI fetches recent publications from UFOP's OAI-PMH repository,
// scores each one and persists the opportunity.
//
// from: ISO-8601 date string (e.g. "2024-01-01"). Empty = full harvest.
// maxRecords: safety cap (0 = unlimited).
func (s *UFOPService) HarvestOAI(ctx context.Context, from string, maxRecords int) (HarvestStats, error) {
	stats := HarvestStats{}

	result, err := s.oai.Harvest(ctx, from, maxRecords)
	if err != nil {
		return stats, fmt.Errorf("oai harvest: %w", err)
	}
	stats.Fetched = result.Total

	for _, pub := range result.Publications {
		if ctx.Err() != nil {
			return stats, ctx.Err()
		}

		// Persist the publication first.
		if err := s.pubRepo.Upsert(ctx, pub); err != nil {
			s.log.Warn("ufop oai: persist publication failed",
				"external_id", pub.ExternalID, "err", err)
			stats.Errors++
			continue
		}

		// Analyze for PI potential.
		in := ufop.AnalyzeInput{
			Title:         pub.Title,
			Abstract:      pub.Abstract,
			Authors:       pub.Authors,
			PublicationID: &pub.ID,
			ExternalID:    pub.ExternalID,
			Source:        domain.UFOPSourceOAI,
			URL:           pub.DOI,
			PublishedAt:   pub.PublishedDate,
			Department:    departmentFromAffiliations(pub.Affiliations),
		}
		opp, err := s.analyzer.Analyze(ctx, in)
		if err != nil {
			s.log.Warn("ufop oai: analyze failed", "external_id", pub.ExternalID, "err", err)
			stats.Errors++
			continue
		}

		if err := s.repo.Upsert(ctx, opp); err != nil {
			s.log.Warn("ufop oai: upsert opportunity failed", "external_id", pub.ExternalID, "err", err)
			stats.Errors++
			continue
		}
		stats.Upserted++
	}

	s.log.Info("ufop oai: harvest complete",
		"fetched", stats.Fetched,
		"upserted", stats.Upserted,
		"errors", stats.Errors)
	return stats, nil
}

// ScrapePortal fetches news from the UFOP web portal and converts
// PI-relevant items into opportunities.
func (s *UFOPService) ScrapePortal(ctx context.Context) (HarvestStats, error) {
	stats := HarvestStats{}

	news, err := s.portal.ScrapeNews(ctx)
	if err != nil {
		return stats, fmt.Errorf("portal scrape: %w", err)
	}
	stats.Fetched = len(news)

	for _, item := range news {
		if ctx.Err() != nil {
			return stats, ctx.Err()
		}

		in := ufop.AnalyzeInput{
			Title:       item.Title,
			Abstract:    item.Abstract,
			ExternalID:  item.URL,
			Source:      domain.UFOPSourcePortal,
			URL:         item.URL,
			PublishedAt: item.Date,
		}
		opp, err := s.analyzer.Analyze(ctx, in)
		if err != nil {
			s.log.Warn("ufop portal: analyze failed", "url", item.URL, "err", err)
			stats.Errors++
			continue
		}

		if err := s.repo.Upsert(ctx, opp); err != nil {
			s.log.Warn("ufop portal: upsert failed", "url", item.URL, "err", err)
			stats.Errors++
			continue
		}
		stats.Upserted++
	}

	s.log.Info("ufop portal: scrape complete",
		"fetched", stats.Fetched,
		"upserted", stats.Upserted,
		"errors", stats.Errors)
	return stats, nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func departmentFromAffiliations(affs []string) string {
	for _, a := range affs {
		if a != "" {
			return a
		}
	}
	return "UFOP"
}
