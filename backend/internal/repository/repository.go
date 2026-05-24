// Package repository declares the persistence contracts that the service
// layer consumes. Concrete implementations live in subpackages (postgres).
//
// This is the "port" side of the Hexagonal Architecture: the application
// depends on these interfaces, never on a specific database driver.
package repository

import (
	"context"

	"github.com/LeoPani/argos/backend/internal/domain"
)

// PatentRepository is the persistence contract for the Patent aggregate.
type PatentRepository interface {
	Insert(ctx context.Context, p *domain.Patent) error
	GetByID(ctx context.Context, id int64) (*domain.Patent, error)
	GetByApplicationNumber(ctx context.Context, appNum string) (*domain.Patent, error)
	List(ctx context.Context, f domain.PatentFilter) ([]domain.Patent, error)
	Count(ctx context.Context, f domain.PatentFilter) (int64, error)
}

// TrademarkRepository is the persistence contract for the Trademark aggregate.
type TrademarkRepository interface {
	Insert(ctx context.Context, t *domain.Trademark) error
	GetByID(ctx context.Context, id int64) (*domain.Trademark, error)
	GetByProcessNumber(ctx context.Context, pn string) (*domain.Trademark, error)
	List(ctx context.Context, f domain.TrademarkFilter) ([]domain.Trademark, error)
	Count(ctx context.Context, f domain.TrademarkFilter) (int64, error)
}

// PublicationRepository is the persistence contract for scientific publications.
type PublicationRepository interface {
	Upsert(ctx context.Context, p *domain.Publication) error
	GetByID(ctx context.Context, id int64) (*domain.Publication, error)
	GetByExternalID(ctx context.Context, source domain.PublicationSource, externalID string) (*domain.Publication, error)
	List(ctx context.Context, f domain.PublicationFilter) ([]domain.Publication, error)
	Count(ctx context.Context, f domain.PublicationFilter) (int64, error)
}

// DisputeRepository is the persistence contract for arbitration disputes.
type DisputeRepository interface {
	Insert(ctx context.Context, d *domain.Dispute) error
	GetByID(ctx context.Context, id int64) (*domain.Dispute, error)
	List(ctx context.Context, f domain.DisputeFilter) ([]domain.Dispute, error)
	Count(ctx context.Context, f domain.DisputeFilter) (int64, error)
	UpdateStatus(ctx context.Context, id int64, status domain.DisputeStatus) error
	AddEvent(ctx context.Context, e *domain.DisputeEvent) error
	AddDocument(ctx context.Context, doc *domain.DisputeDocument) error
}

// UFOPRepository is the persistence contract for UFOP PI opportunities.
type UFOPRepository interface {
	Upsert(ctx context.Context, o *domain.UFOPOpportunity) error
	GetByID(ctx context.Context, id int64) (*domain.UFOPOpportunity, error)
	List(ctx context.Context, f domain.UFOPFilter) ([]domain.UFOPOpportunity, error)
	Count(ctx context.Context, f domain.UFOPFilter) (int64, error)
	UpdateStatus(ctx context.Context, id int64, status domain.UFOPOpportunityStatus) error
}

// WatchlistRepository is the persistence contract for monitoring alerts.
type WatchlistRepository interface {
	Insert(ctx context.Context, w *domain.Watchlist) error
	GetByID(ctx context.Context, id int64) (*domain.Watchlist, error)
	List(ctx context.Context) ([]domain.Watchlist, error)
	Delete(ctx context.Context, id int64) error
	UpdateCheck(ctx context.Context, id int64, newCount int, status domain.WatchStatus) error
}
