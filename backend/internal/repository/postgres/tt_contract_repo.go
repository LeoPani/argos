package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/LeoPani/argos/backend/internal/domain"
)

// TTContractRepo persists tt_contracts rows.
type TTContractRepo struct{ db *sql.DB }

func NewTTContractRepo(db *sql.DB) *TTContractRepo { return &TTContractRepo{db: db} }

const ttContractCols = `
	id, contract_number, patent_id, pool_id,
	licensor, licensee, licensee_cnpj,
	license_kind, sublicensable, territory, field_of_use,
	royalty_rate, royalty_floor_annual, upfront_fee, inventor_share_pct,
	milestones, signed_at, expires_at,
	status, nit_approved, audit_rights,
	notes, created_at, updated_at`

func (r *TTContractRepo) Insert(ctx context.Context, c *domain.TTContract) error {
	const q = `
		INSERT INTO tt_contracts (
			contract_number, patent_id, pool_id,
			licensor, licensee, licensee_cnpj,
			license_kind, sublicensable, territory, field_of_use,
			royalty_rate, royalty_floor_annual, upfront_fee, inventor_share_pct,
			milestones, signed_at, expires_at,
			status, nit_approved, audit_rights, notes
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21
		)
		RETURNING id, created_at, updated_at`

	if len(c.Milestones) == 0 {
		c.Milestones = []byte("[]")
	}

	err := r.db.QueryRowContext(ctx, q,
		c.ContractNumber, c.PatentID, c.PoolID,
		c.Licensor, c.Licensee, c.LicenseeCNPJ,
		string(c.LicenseKind), c.Sublicensable, c.Territory, c.FieldOfUse,
		c.RoyaltyRate, c.RoyaltyFloorAnnual, c.UpfrontFee, c.InventorSharePct,
		c.Milestones, c.SignedAt, c.ExpiresAt,
		string(c.Status), c.NITApproved, c.AuditRights, c.Notes,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert tt_contract %s: %w", c.ContractNumber, err)
	}
	return nil
}

func (r *TTContractRepo) GetByID(ctx context.Context, id int64) (*domain.TTContract, error) {
	row := r.db.QueryRowContext(ctx, "SELECT "+ttContractCols+" FROM tt_contracts WHERE id=$1", id)
	c, err := scanTTContract(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("tt_contract id=%d: %w", id, domain.ErrNotFound)
	}
	return c, err
}

func (r *TTContractRepo) List(ctx context.Context, f domain.TTContractFilter) ([]domain.TTContract, error) {
	where, args := buildTTWhere(f)
	q := "SELECT " + ttContractCols + " FROM tt_contracts" + where +
		" ORDER BY created_at DESC" +
		fmt.Sprintf(" LIMIT %d OFFSET %d", f.Limit, f.Offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list tt_contracts: %w", err)
	}
	defer rows.Close()

	var out []domain.TTContract
	for rows.Next() {
		c, err := scanTTContract(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *TTContractRepo) Count(ctx context.Context, f domain.TTContractFilter) (int64, error) {
	where, args := buildTTWhere(f)
	var n int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tt_contracts"+where, args...).Scan(&n)
	return n, err
}

func (r *TTContractRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM tt_contracts WHERE id=$1", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("tt_contract id=%d: %w", id, domain.ErrNotFound)
	}
	return nil
}

func (r *TTContractRepo) UpdateStatus(ctx context.Context, id int64, status domain.ContractStatus) error {
	res, err := r.db.ExecContext(ctx,
		"UPDATE tt_contracts SET status=$1 WHERE id=$2", string(status), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("tt_contract id=%d: %w", id, domain.ErrNotFound)
	}
	return nil
}

type ttScanner interface{ Scan(dest ...any) error }

func scanTTContract(s ttScanner) (*domain.TTContract, error) {
	var (
		c          domain.TTContract
		kindStr    string
		statusStr  string
		milestones sql.NullString
	)
	err := s.Scan(
		&c.ID, &c.ContractNumber, &c.PatentID, &c.PoolID,
		&c.Licensor, &c.Licensee, &c.LicenseeCNPJ,
		&kindStr, &c.Sublicensable, &c.Territory, &c.FieldOfUse,
		&c.RoyaltyRate, &c.RoyaltyFloorAnnual, &c.UpfrontFee, &c.InventorSharePct,
		&milestones, &c.SignedAt, &c.ExpiresAt,
		&statusStr, &c.NITApproved, &c.AuditRights,
		&c.Notes, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.LicenseKind = domain.LicenseKind(kindStr)
	c.Status = domain.ContractStatus(statusStr)
	if milestones.Valid {
		c.Milestones = json.RawMessage(milestones.String)
	} else {
		c.Milestones = []byte("[]")
	}
	return &c, nil
}

func buildTTWhere(f domain.TTContractFilter) (string, []any) {
	var clauses []string
	var args []any
	n := 1
	add := func(clause string, v any) {
		clauses = append(clauses, fmt.Sprintf(clause, n))
		args = append(args, v)
		n++
	}
	if f.Status != "" {
		add("status = $%d", string(f.Status))
	}
	if f.PatentID != nil {
		add("patent_id = $%d", *f.PatentID)
	}
	if f.PoolID != nil {
		add("pool_id = $%d", *f.PoolID)
	}
	if f.Search != "" {
		clauses = append(clauses, fmt.Sprintf(
			"(contract_number ILIKE $%d OR licensee ILIKE $%d OR notes ILIKE $%d)", n, n, n))
		args = append(args, "%"+f.Search+"%")
		n++
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
