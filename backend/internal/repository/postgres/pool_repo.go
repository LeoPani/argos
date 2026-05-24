package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/LeoPani/argos/backend/internal/domain"
)

// PoolRepo persists patent pools and their members.
type PoolRepo struct{ db *sql.DB }

func NewPoolRepo(db *sql.DB) *PoolRepo { return &PoolRepo{db: db} }

const poolCols = `id, name, description, pool_kind, royalty_rate, territory, duration_years, administrator, status, created_at, updated_at`

func (r *PoolRepo) Insert(ctx context.Context, p *domain.PatentPool) error {
	const q = `
		INSERT INTO patent_pools (name, description, pool_kind, royalty_rate, territory, duration_years, administrator, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, q,
		p.Name, p.Description, string(p.Kind), p.RoyaltyRate, p.Territory,
		p.DurationYears, p.Administrator, string(p.Status),
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *PoolRepo) GetByID(ctx context.Context, id int64) (*domain.PatentPool, error) {
	row := r.db.QueryRowContext(ctx, "SELECT "+poolCols+" FROM patent_pools WHERE id=$1", id)
	p, err := scanPool(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("pool id=%d: %w", id, domain.ErrNotFound)
	}
	if err != nil {
		return nil, err
	}
	// Hydrate members.
	p.Members, err = r.ListMembers(ctx, id)
	return p, err
}

func (r *PoolRepo) List(ctx context.Context) ([]domain.PatentPool, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+poolCols+" FROM patent_pools ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.PatentPool
	for rows.Next() {
		p, err := scanPool(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Hydrate member counts (one query for all pools).
	if len(out) > 0 {
		members, err := r.listAllMembersHydrated(ctx)
		if err == nil {
			byPool := map[int64][]domain.PoolMember{}
			for _, m := range members {
				byPool[m.PoolID] = append(byPool[m.PoolID], m)
			}
			for i := range out {
				out[i].Members = byPool[out[i].ID]
			}
		}
	}
	return out, nil
}

func (r *PoolRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM patent_pools WHERE id=$1", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("pool id=%d: %w", id, domain.ErrNotFound)
	}
	return nil
}

// ─── Members ─────────────────────────────────────────────────────────────────

func (r *PoolRepo) AddMember(ctx context.Context, poolID, patentID int64, sharePct float64) (*domain.PoolMember, error) {
	const q = `
		INSERT INTO pool_members (pool_id, patent_id, share_pct)
		VALUES ($1, $2, $3)
		ON CONFLICT (pool_id, patent_id) DO UPDATE SET share_pct = EXCLUDED.share_pct
		RETURNING id, added_at`

	m := domain.PoolMember{PoolID: poolID, PatentID: patentID, SharePct: sharePct}
	if err := r.db.QueryRowContext(ctx, q, poolID, patentID, sharePct).Scan(&m.ID, &m.AddedAt); err != nil {
		return nil, fmt.Errorf("add pool member: %w", err)
	}
	return &m, nil
}

func (r *PoolRepo) RemoveMember(ctx context.Context, poolID, patentID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM pool_members WHERE pool_id=$1 AND patent_id=$2", poolID, patentID)
	return err
}

func (r *PoolRepo) ListMembers(ctx context.Context, poolID int64) ([]domain.PoolMember, error) {
	const q = `
		SELECT pm.id, pm.pool_id, pm.patent_id, pm.share_pct, pm.added_at,
		       p.application_number, p.title
		FROM pool_members pm
		LEFT JOIN patents p ON p.id = pm.patent_id
		WHERE pm.pool_id = $1
		ORDER BY pm.share_pct DESC`

	rows, err := r.db.QueryContext(ctx, q, poolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.PoolMember
	for rows.Next() {
		var m domain.PoolMember
		var appNum, title sql.NullString
		if err := rows.Scan(&m.ID, &m.PoolID, &m.PatentID, &m.SharePct, &m.AddedAt, &appNum, &title); err != nil {
			return nil, err
		}
		if appNum.Valid {
			m.PatentNumber = appNum.String
		}
		if title.Valid {
			m.PatentTitle = title.String
		}
		out = append(out, m)
	}
	if out == nil {
		out = []domain.PoolMember{}
	}
	return out, rows.Err()
}

// listAllMembersHydrated bulk-loads members for all pools (used by List).
func (r *PoolRepo) listAllMembersHydrated(ctx context.Context) ([]domain.PoolMember, error) {
	const q = `
		SELECT pm.id, pm.pool_id, pm.patent_id, pm.share_pct, pm.added_at,
		       p.application_number, p.title
		FROM pool_members pm
		LEFT JOIN patents p ON p.id = pm.patent_id
		ORDER BY pm.pool_id, pm.share_pct DESC`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.PoolMember
	for rows.Next() {
		var m domain.PoolMember
		var appNum, title sql.NullString
		if err := rows.Scan(&m.ID, &m.PoolID, &m.PatentID, &m.SharePct, &m.AddedAt, &appNum, &title); err != nil {
			return nil, err
		}
		if appNum.Valid {
			m.PatentNumber = appNum.String
		}
		if title.Valid {
			m.PatentTitle = title.String
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

type poolScanner interface{ Scan(dest ...any) error }

func scanPool(s poolScanner) (*domain.PatentPool, error) {
	var p domain.PatentPool
	var kind, status string
	err := s.Scan(
		&p.ID, &p.Name, &p.Description, &kind, &p.RoyaltyRate,
		&p.Territory, &p.DurationYears, &p.Administrator, &status,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	p.Kind = domain.PoolKind(kind)
	p.Status = domain.PoolStatus(status)
	return &p, nil
}
