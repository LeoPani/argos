package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/repository"
)

const pgUniqueViolation = "23505"

type PatentRepo struct {
	db *sql.DB
}

var _ repository.PatentRepository = (*PatentRepo)(nil)

func NewPatentRepo(db *sql.DB) *PatentRepo {
	return &PatentRepo{db: db}
}

const selectPatentColumns = `
	id, application_number, title, abstract, applicant, inventors,
	filing_date, publication_date, ipc_category, ipc_code,
	rpi_issue, status, created_at, updated_at`

func (r *PatentRepo) Insert(ctx context.Context, p *domain.Patent) error {
	const query = `
		INSERT INTO patents (
			application_number, title, abstract, applicant, inventors,
			filing_date, publication_date, ipc_category, ipc_code,
			rpi_issue, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at`

	var ipcCategory sql.NullInt16
	if p.IPCCategory.IsValid() {
		ipcCategory = sql.NullInt16{Int16: int16(p.IPCCategory), Valid: true}
	}

	status := p.Status
	if status == "" {
		status = domain.PatentStatusPending
	}

	// FIX: inventors column is TEXT[] NOT NULL — coerce nil to []string{}
	// so pq.Array sends '{}' instead of NULL.
	inventors := p.Inventors
	if inventors == nil {
		inventors = []string{}
	}

	err := r.db.QueryRowContext(ctx, query,
		p.ApplicationNumber,
		p.Title,
		p.Abstract,
		p.Applicant,
		pq.Array(inventors),
		p.FilingDate,
		p.PublicationDate,
		ipcCategory,
		p.IPCCode,
		p.RPIIssue,
		string(status),
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pgUniqueViolation {
			return fmt.Errorf("insert patent %q: %w",
				p.ApplicationNumber, domain.ErrDuplicate)
		}
		return fmt.Errorf("insert patent %q: %w", p.ApplicationNumber, err)
	}
	p.Status = status
	p.Inventors = inventors
	return nil
}

func (r *PatentRepo) GetByID(ctx context.Context, id int64) (*domain.Patent, error) {
	query := "SELECT " + selectPatentColumns + " FROM patents WHERE id = $1"
	row := r.db.QueryRowContext(ctx, query, id)

	p, err := scanPatent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get patent id=%d: %w", id, domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get patent id=%d: %w", id, err)
	}
	return p, nil
}

func (r *PatentRepo) GetByApplicationNumber(ctx context.Context, appNum string) (*domain.Patent, error) {
	query := "SELECT " + selectPatentColumns + " FROM patents WHERE application_number = $1"
	row := r.db.QueryRowContext(ctx, query, appNum)

	p, err := scanPatent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get patent app=%q: %w", appNum, domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get patent app=%q: %w", appNum, err)
	}
	return p, nil
}

func (r *PatentRepo) List(ctx context.Context, f domain.PatentFilter) ([]domain.Patent, error) {
	where, args := buildPatentWhere(&f)

	query := fmt.Sprintf(
		"SELECT %s FROM patents %s ORDER BY publication_date DESC NULLS LAST, id DESC LIMIT $%d OFFSET $%d",
		selectPatentColumns, where, len(args)+1, len(args)+2,
	)
	args = append(args, f.Limit, f.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list patents: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Patent, 0, f.Limit)
	for rows.Next() {
		p, err := scanPatent(rows)
		if err != nil {
			return nil, fmt.Errorf("list patents: scan: %w", err)
		}
		out = append(out, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list patents: iterate: %w", err)
	}
	return out, nil
}

func (r *PatentRepo) Count(ctx context.Context, f domain.PatentFilter) (int64, error) {
	where, args := buildPatentWhere(&f)
	query := "SELECT COUNT(*) FROM patents " + where

	var total int64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count patents: %w", err)
	}
	return total, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanPatent(s scanner) (*domain.Patent, error) {
	var (
		p          domain.Patent
		filingDate sql.NullTime
		pubDate    sql.NullTime
		ipcCat     sql.NullInt16
		inventors  pq.StringArray
		status     string
	)

	err := s.Scan(
		&p.ID,
		&p.ApplicationNumber,
		&p.Title,
		&p.Abstract,
		&p.Applicant,
		&inventors,
		&filingDate,
		&pubDate,
		&ipcCat,
		&p.IPCCode,
		&p.RPIIssue,
		&status,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	p.Inventors = []string(inventors)
	if filingDate.Valid {
		p.FilingDate = &filingDate.Time
	}
	if pubDate.Valid {
		p.PublicationDate = &pubDate.Time
	}
	if ipcCat.Valid {
		p.IPCCategory = domain.IPCCategory(ipcCat.Int16)
	} else {
		p.IPCCategory = domain.IPCCategoryUnknown
	}
	p.Status = domain.PatentStatus(status)
	return &p, nil
}

func buildPatentWhere(f *domain.PatentFilter) (string, []any) {
	var (
		clauses []string
		args    []any
		i       = 1
	)

	if f.Category != nil {
		clauses = append(clauses, fmt.Sprintf("ipc_category = $%d", i))
		args = append(args, int(*f.Category))
		i++
	}
	if f.Status != "" {
		clauses = append(clauses, fmt.Sprintf("status = $%d", i))
		args = append(args, string(f.Status))
		i++
	}
	if f.Search != "" {
		clauses = append(clauses, fmt.Sprintf("(title ILIKE $%d OR abstract ILIKE $%d)", i, i+1))
		like := "%" + f.Search + "%"
		args = append(args, like, like)
		i += 2
	}
	if f.RPIIssue != "" {
		clauses = append(clauses, fmt.Sprintf("rpi_issue = $%d", i))
		args = append(args, f.RPIIssue)
		i++
	}
	if f.FilingFrom != nil {
		clauses = append(clauses, fmt.Sprintf("filing_date >= $%d", i))
		args = append(args, *f.FilingFrom)
		i++
	}
	if f.FilingUntil != nil {
		clauses = append(clauses, fmt.Sprintf("filing_date <= $%d", i))
		args = append(args, *f.FilingUntil)
		i++
	}

	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}
