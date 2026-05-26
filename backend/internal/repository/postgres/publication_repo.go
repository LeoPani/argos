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

// PublicationRepo implements repository.PublicationRepository.
type PublicationRepo struct{ db *sql.DB }

var _ repository.PublicationRepository = (*PublicationRepo)(nil)

func NewPublicationRepo(db *sql.DB) *PublicationRepo { return &PublicationRepo{db: db} }

const selectPublicationColumns = `
	id, source, external_id, doi, title, abstract,
	authors, affiliations, kind, journal, published_date,
	citation_count, keywords, url, ipc_category,
	created_at, updated_at`

func (r *PublicationRepo) Upsert(ctx context.Context, p *domain.Publication) error {
	const q = `
		INSERT INTO publications (
			source, external_id, doi, title, abstract,
			authors, affiliations, kind, journal, published_date,
			citation_count, keywords, url, ipc_category
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (source, external_id) DO UPDATE SET
			title          = EXCLUDED.title,
			abstract       = EXCLUDED.abstract,
			citation_count = EXCLUDED.citation_count,
			keywords       = EXCLUDED.keywords,
			updated_at     = NOW()
		RETURNING id, created_at, updated_at`

	if p.Authors == nil {
		p.Authors = []string{}
	}
	if p.Affiliations == nil {
		p.Affiliations = []string{}
	}
	if p.Keywords == nil {
		p.Keywords = []string{}
	}

	var ipcCat sql.NullInt16
	if p.IPCCategory.IsValid() {
		ipcCat = sql.NullInt16{Int16: int16(p.IPCCategory), Valid: true}
	}

	err := r.db.QueryRowContext(ctx, q,
		string(p.Source), p.ExternalID, p.DOI, p.Title, p.Abstract,
		pq.Array(p.Authors), pq.Array(p.Affiliations),
		string(p.Kind), p.Journal, p.PublishedDate,
		p.CitationCount, pq.Array(p.Keywords), p.URL, ipcCat,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert publication %q/%q: %w", p.Source, p.ExternalID, err)
	}
	return nil
}

func (r *PublicationRepo) GetByID(ctx context.Context, id int64) (*domain.Publication, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+selectPublicationColumns+" FROM publications WHERE id = $1", id)
	p, err := scanPub(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("publication id=%d: %w", id, domain.ErrNotFound)
	}
	return p, err
}

func (r *PublicationRepo) GetByExternalID(ctx context.Context, source domain.PublicationSource, externalID string) (*domain.Publication, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+selectPublicationColumns+" FROM publications WHERE source=$1 AND external_id=$2",
		string(source), externalID)
	p, err := scanPub(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("publication %s/%s: %w", source, externalID, domain.ErrNotFound)
	}
	return p, err
}

func (r *PublicationRepo) List(ctx context.Context, f domain.PublicationFilter) ([]domain.Publication, error) {
	where, args := buildPublicationWhere(f)
	q := "SELECT " + selectPublicationColumns + " FROM publications" + where +
		" ORDER BY published_date DESC NULLS LAST, id DESC" +
		fmt.Sprintf(" LIMIT %d OFFSET %d", f.Limit, f.Offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list publications: %w", err)
	}
	defer rows.Close()

	var out []domain.Publication
	for rows.Next() {
		p, err := scanPub(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *PublicationRepo) Count(ctx context.Context, f domain.PublicationFilter) (int64, error) {
	where, args := buildPublicationWhere(f)
	var n int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM publications"+where, args...).Scan(&n)
	return n, err
}

type pubRow interface{ Scan(dest ...any) error }

func scanPub(s pubRow) (*domain.Publication, error) {
	var p domain.Publication
	var ipcCat sql.NullInt16
	err := s.Scan(
		&p.ID, &p.Source, &p.ExternalID, &p.DOI, &p.Title, &p.Abstract,
		pq.Array(&p.Authors), pq.Array(&p.Affiliations),
		&p.Kind, &p.Journal, &p.PublishedDate,
		&p.CitationCount, pq.Array(&p.Keywords), &p.URL, &ipcCat,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if ipcCat.Valid {
		p.IPCCategory = domain.IPCCategory(ipcCat.Int16)
	}
	return &p, err
}

func buildPublicationWhere(f domain.PublicationFilter) (string, []any) {
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
	if f.Kind != "" {
		add("kind = $%d", string(f.Kind))
	}
	if f.Category != nil {
		add("ipc_category = $%d", int16(*f.Category))
	}
	if f.Search != "" {
		clauses = append(clauses, fmt.Sprintf("(title ILIKE $%d OR abstract ILIKE $%d)", n, n))
		args = append(args, "%"+f.Search+"%")
		n++
	}
	if f.PublishedFrom != nil {
		add("published_date >= $%d", f.PublishedFrom)
	}
	if f.PublishedUntil != nil {
		add("published_date <= $%d", f.PublishedUntil)
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
