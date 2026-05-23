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
// Implementations MUST:
//   - Honor context cancellation/timeouts on every method.
//   - Translate driver-specific errors into domain sentinels
//     (domain.ErrNotFound on no rows, domain.ErrDuplicate on
//     unique-constraint violations, etc.).
//   - Be safe for concurrent use across goroutines.
type PatentRepository interface {
	// Insert persists a new patent. The Patent.ID and timestamp fields
	// are populated on the input struct in-place.
	//
	// Returns domain.ErrDuplicate if ApplicationNumber already exists.
	Insert(ctx context.Context, p *domain.Patent) error

	// GetByID fetches a single patent by its primary key.
	// Returns domain.ErrNotFound if no row matches.
	GetByID(ctx context.Context, id int64) (*domain.Patent, error)

	// GetByApplicationNumber fetches a patent by its (unique) INPI
	// application number. Used by the worker to deduplicate ingestion.
	// Returns domain.ErrNotFound if no row matches.
	GetByApplicationNumber(ctx context.Context, appNum string) (*domain.Patent, error)

	// List returns patents matching the filter, ordered by publication
	// date descending then id descending. The caller MUST call
	// f.Normalize() before invoking this method.
	List(ctx context.Context, f domain.PatentFilter) ([]domain.Patent, error)

	// Count returns the total rows matching the filter, ignoring the
	// Limit/Offset fields. Used by handlers to render pagination metadata.
	Count(ctx context.Context, f domain.PatentFilter) (int64, error)
}
