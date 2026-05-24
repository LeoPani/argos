package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/LeoPani/argos/backend/internal/domain"
)

// WatchlistRepo persists watchlist subscriptions.
type WatchlistRepo struct{ db *sql.DB }

// NewWatchlistRepo creates a new repo.
func NewWatchlistRepo(db *sql.DB) *WatchlistRepo { return &WatchlistRepo{db: db} }

const selectWatchlistCols = `id, label, watch_type, query, last_check, new_count, status, auto_dispute, similarity_threshold, created_at, updated_at`

// Insert adds a new watchlist entry.
func (r *WatchlistRepo) Insert(ctx context.Context, w *domain.Watchlist) error {
	const q = `
		INSERT INTO watchlists (label, watch_type, query, status, auto_dispute, similarity_threshold)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, last_check, new_count, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, q,
		w.Label, string(w.Type), w.Query, string(w.Status),
		w.AutoDispute, w.SimilarityThreshold,
	).Scan(&w.ID, &w.LastCheck, &w.NewCount, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert watchlist %q: %w", w.Label, err)
	}
	return nil
}

// List returns all watchlists ordered by most recent activity.
func (r *WatchlistRepo) List(ctx context.Context) ([]domain.Watchlist, error) {
	const q = `SELECT ` + selectWatchlistCols + `
		FROM watchlists
		ORDER BY status DESC, updated_at DESC`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list watchlists: %w", err)
	}
	defer rows.Close()

	var out []domain.Watchlist
	for rows.Next() {
		w, err := scanWatchlist(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *w)
	}
	return out, rows.Err()
}

// GetByID fetches a single watchlist.
func (r *WatchlistRepo) GetByID(ctx context.Context, id int64) (*domain.Watchlist, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+selectWatchlistCols+" FROM watchlists WHERE id=$1", id)
	w, err := scanWatchlist(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("watchlist id=%d: %w", id, domain.ErrNotFound)
	}
	return w, err
}

// Delete removes a watchlist.
func (r *WatchlistRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM watchlists WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete watchlist %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("watchlist id=%d: %w", id, domain.ErrNotFound)
	}
	return nil
}

// UpdateCheck records a check-run: new_count, status and last_check.
func (r *WatchlistRepo) UpdateCheck(
	ctx context.Context, id int64, newCount int, status domain.WatchStatus,
) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE watchlists
		SET last_check = NOW(), new_count = $1, status = $2
		WHERE id = $3`,
		newCount, string(status), id,
	)
	return err
}

// ─── helpers ──────────────────────────────────────────────────────────────────

type watchlistRow interface{ Scan(dest ...any) error }

func scanWatchlist(s watchlistRow) (*domain.Watchlist, error) {
	var (
		w         domain.Watchlist
		lastCheck sql.NullTime
		wType     string
		status    string
	)
	err := s.Scan(
		&w.ID, &w.Label, &wType, &w.Query, &lastCheck,
		&w.NewCount, &status, &w.AutoDispute, &w.SimilarityThreshold,
		&w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	w.Type = domain.WatchType(wType)
	w.Status = domain.WatchStatus(status)
	if lastCheck.Valid {
		t := lastCheck.Time
		w.LastCheck = &t
	}
	_ = time.Time{} // keep import
	return &w, nil
}
