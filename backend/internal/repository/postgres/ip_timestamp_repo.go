package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"github.com/LeoPani/argos/backend/internal/domain"
)

// IPTimestampRepo persists proof-of-existence records.
type IPTimestampRepo struct{ db *sql.DB }

func NewIPTimestampRepo(db *sql.DB) *IPTimestampRepo { return &IPTimestampRepo{db: db} }

// Insert creates a new timestamp record.
func (r *IPTimestampRepo) Insert(ctx context.Context, t *domain.IPTimestamp) error {
	const q = `
		INSERT INTO ip_timestamps
		  (title, description, authors, category, content_hash, prev_hash, chain_index)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	return r.db.QueryRowContext(ctx, q,
		t.Title, t.Description, pq.Array(t.Authors),
		t.Category, t.ContentHash, t.PrevHash, t.ChainIndex,
	).Scan(&t.ID, &t.CreatedAt)
}

// GetByID fetches a single record.
func (r *IPTimestampRepo) GetByID(ctx context.Context, id int64) (*domain.IPTimestamp, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, title, description, authors, category,
		        content_hash, COALESCE(prev_hash,''), chain_index, created_at
		 FROM ip_timestamps WHERE id = $1`, id)
	return scanTimestamp(row)
}

// GetLatest returns the most recent record (for chaining).
func (r *IPTimestampRepo) GetLatest(ctx context.Context) (*domain.IPTimestamp, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, title, description, authors, category,
		        content_hash, COALESCE(prev_hash,''), chain_index, created_at
		 FROM ip_timestamps ORDER BY chain_index DESC LIMIT 1`)
	t, err := scanTimestamp(row)
	if err == sql.ErrNoRows {
		return nil, nil // nenhum registro ainda — ok
	}
	return t, err
}

// List returns paginated records ordered by newest first.
func (r *IPTimestampRepo) List(ctx context.Context, f domain.IPTimestampFilter) ([]domain.IPTimestamp, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, title, description, authors, category,
		        content_hash, COALESCE(prev_hash,''), chain_index, created_at
		 FROM ip_timestamps
		 ORDER BY chain_index DESC
		 LIMIT $1 OFFSET $2`, limit, f.Offset)
	if err != nil {
		return nil, fmt.Errorf("list ip_timestamps: %w", err)
	}
	defer rows.Close()

	var out []domain.IPTimestamp
	for rows.Next() {
		t, err := scanTimestamp(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// Count returns total records.
func (r *IPTimestampRepo) Count(ctx context.Context) (int64, error) {
	var n int64
	return n, r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ip_timestamps`).Scan(&n)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

type tsRow interface{ Scan(dest ...any) error }

func scanTimestamp(s tsRow) (*domain.IPTimestamp, error) {
	var t domain.IPTimestamp
	err := s.Scan(
		&t.ID, &t.Title, &t.Description,
		pq.Array(&t.Authors), &t.Category,
		&t.ContentHash, &t.PrevHash, &t.ChainIndex, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
