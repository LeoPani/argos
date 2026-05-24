package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"github.com/LeoPani/argos/backend/internal/domain"
)

// UFOPRepo persists UFOP opportunity records.
type UFOPRepo struct{ db *sql.DB }

func NewUFOPRepo(db *sql.DB) *UFOPRepo { return &UFOPRepo{db: db} }

const selectUFOPColumns = `
	id, source, external_id, title, authors, department,
	abstract, url, published_at,
	ipc_suggestion, ipc_category, opportunity_level,
	similarity_pct, pi_score, ai_analysis,
	status, publication_id, created_at, updated_at`

// Upsert inserts or updates a UFOP opportunity (keyed on source+external_id).
func (r *UFOPRepo) Upsert(ctx context.Context, o *domain.UFOPOpportunity) error {
	const q = `
		INSERT INTO ufop_opportunities (
			source, external_id, title, authors, department,
			abstract, url, published_at,
			ipc_suggestion, ipc_category, opportunity_level,
			similarity_pct, pi_score, ai_analysis,
			status, publication_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		ON CONFLICT (source, external_id) DO UPDATE SET
			title             = EXCLUDED.title,
			abstract          = EXCLUDED.abstract,
			ipc_suggestion    = EXCLUDED.ipc_suggestion,
			ipc_category      = EXCLUDED.ipc_category,
			opportunity_level = EXCLUDED.opportunity_level,
			similarity_pct    = EXCLUDED.similarity_pct,
			pi_score          = EXCLUDED.pi_score,
			ai_analysis       = EXCLUDED.ai_analysis,
			updated_at        = NOW()
		RETURNING id, created_at, updated_at`

	if o.Authors == nil {
		o.Authors = []string{}
	}

	var ipcCat sql.NullInt16
	if o.IPCCategory.IsValid() {
		ipcCat = sql.NullInt16{Int16: int16(o.IPCCategory), Valid: true}
	}

	level := o.Level
	if level == "" {
		level = domain.UFOPLevelLow
	}
	status := o.Status
	if status == "" {
		status = domain.UFOPStatusNew
	}

	err := r.db.QueryRowContext(ctx, q,
		string(o.Source), o.ExternalID, o.Title,
		pq.Array(o.Authors), o.Department,
		o.Abstract, o.URL, o.PublishedAt,
		o.IPCSuggestion, ipcCat, string(level),
		o.SimilarityPct, o.PIScore, o.AIAnalysis,
		string(status), o.PublicationID,
	).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert ufop opportunity %q: %w", o.ExternalID, err)
	}
	return nil
}

// GetByID fetches a single opportunity.
func (r *UFOPRepo) GetByID(ctx context.Context, id int64) (*domain.UFOPOpportunity, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+selectUFOPColumns+" FROM ufop_opportunities WHERE id=$1", id)
	o, err := scanUFOP(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("ufop opportunity id=%d: %w", id, domain.ErrNotFound)
	}
	return o, err
}

// List returns opportunities matching the filter.
func (r *UFOPRepo) List(ctx context.Context, f domain.UFOPFilter) ([]domain.UFOPOpportunity, error) {
	where, args := buildUFOPWhere(f)
	q := "SELECT " + selectUFOPColumns + " FROM ufop_opportunities" + where +
		" ORDER BY pi_score DESC, published_at DESC NULLS LAST" +
		fmt.Sprintf(" LIMIT %d OFFSET %d", f.Limit, f.Offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list ufop opportunities: %w", err)
	}
	defer rows.Close()

	var out []domain.UFOPOpportunity
	for rows.Next() {
		o, err := scanUFOP(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

// Count returns the total matching rows.
func (r *UFOPRepo) Count(ctx context.Context, f domain.UFOPFilter) (int64, error) {
	where, args := buildUFOPWhere(f)
	var n int64
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM ufop_opportunities"+where, args...).Scan(&n)
	return n, err
}

// UpdateStatus transitions the review lifecycle of an opportunity.
func (r *UFOPRepo) UpdateStatus(ctx context.Context, id int64, status domain.UFOPOpportunityStatus) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE ufop_opportunities SET status=$1 WHERE id=$2", string(status), id)
	return err
}

// ─── helpers ──────────────────────────────────────────────────────────────────

type ufopRow interface{ Scan(dest ...any) error }

func scanUFOP(s ufopRow) (*domain.UFOPOpportunity, error) {
	var o domain.UFOPOpportunity
	var ipcCat sql.NullInt16
	err := s.Scan(
		&o.ID, &o.Source, &o.ExternalID, &o.Title,
		pq.Array(&o.Authors), &o.Department,
		&o.Abstract, &o.URL, &o.PublishedAt,
		&o.IPCSuggestion, &ipcCat, &o.Level,
		&o.SimilarityPct, &o.PIScore, &o.AIAnalysis,
		&o.Status, &o.PublicationID, &o.CreatedAt, &o.UpdatedAt,
	)
	if ipcCat.Valid {
		o.IPCCategory = domain.IPCCategory(ipcCat.Int16)
	}
	return &o, err
}

func buildUFOPWhere(f domain.UFOPFilter) (string, []any) {
	var (
		clauses []string
		args    []any
		n       = 1
	)
	add := func(clause string, v any) {
		clauses = append(clauses, fmt.Sprintf(clause, n))
		args = append(args, v)
		n++
	}
	if f.Source != "" {
		add("source = $%d", string(f.Source))
	}
	if f.Level != "" {
		add("opportunity_level = $%d", string(f.Level))
	}
	if f.Status != "" {
		add("status = $%d", string(f.Status))
	}
	if f.Search != "" {
		clauses = append(clauses,
			fmt.Sprintf("(title ILIKE $%d OR abstract ILIKE $%d)", n, n))
		args = append(args, "%"+f.Search+"%")
		n++
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
