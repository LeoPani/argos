package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/lib/pq"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/repository"
)

// TrademarkRepo implements repository.TrademarkRepository.
type TrademarkRepo struct{ db *sql.DB }

var _ repository.TrademarkRepository = (*TrademarkRepo)(nil)

func NewTrademarkRepo(db *sql.DB) *TrademarkRepo { return &TrademarkRepo{db: db} }

const selectTrademarkColumns = `
	id, process_number, name, normalized_name, kind, status, owner,
	nice_classes, image_url, filing_date, publication_date,
	granted_date, rpi_issue, created_at, updated_at`

// normalize strips accents and uppercases for fuzzy search.
func normalizeName(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, _ := transform.String(t, s)
	return strings.ToUpper(strings.TrimSpace(out))
}

func (r *TrademarkRepo) Insert(ctx context.Context, t *domain.Trademark) error {
	const q = `
		INSERT INTO trademarks (
			process_number, name, normalized_name, kind, status, owner,
			nice_classes, image_url, filing_date, publication_date,
			granted_date, rpi_issue
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id, created_at, updated_at`

	t.NormalizedName = normalizeName(t.Name)

	classes := t.NiceClasses
	if classes == nil {
		classes = []int{}
	}

	status := t.Status
	if status == "" {
		status = domain.TrademarkStatusFiled
	}

	err := r.db.QueryRowContext(ctx, q,
		t.ProcessNumber, t.Name, t.NormalizedName,
		string(t.Kind), string(status), t.Owner,
		pq.Array(classes), t.ImageURL,
		t.FilingDate, t.PublicationDate, t.GrantedDate, t.RPIIssue,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pgUniqueViolation {
			return fmt.Errorf("insert trademark %q: %w", t.ProcessNumber, domain.ErrDuplicate)
		}
		return fmt.Errorf("insert trademark %q: %w", t.ProcessNumber, err)
	}
	t.Status = status
	return nil
}

func (r *TrademarkRepo) GetByID(ctx context.Context, id int64) (*domain.Trademark, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+selectTrademarkColumns+" FROM trademarks WHERE id = $1", id)
	t, err := scanTrademark(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("trademark id=%d: %w", id, domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("trademark id=%d: %w", id, err)
	}
	return t, nil
}

func (r *TrademarkRepo) GetByProcessNumber(ctx context.Context, pn string) (*domain.Trademark, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+selectTrademarkColumns+" FROM trademarks WHERE process_number = $1", pn)
	t, err := scanTrademark(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("trademark %q: %w", pn, domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("trademark %q: %w", pn, err)
	}
	return t, nil
}

func (r *TrademarkRepo) List(ctx context.Context, f domain.TrademarkFilter) ([]domain.Trademark, error) {
	where, args := buildTrademarkWhere(f)
	q := "SELECT " + selectTrademarkColumns + " FROM trademarks" + where +
		" ORDER BY filing_date DESC NULLS LAST, id DESC" +
		fmt.Sprintf(" LIMIT %d OFFSET %d", f.Limit, f.Offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list trademarks: %w", err)
	}
	defer rows.Close()

	var out []domain.Trademark
	for rows.Next() {
		t, err := scanTrademark(rows)
		if err != nil {
			return nil, fmt.Errorf("scan trademark: %w", err)
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (r *TrademarkRepo) Count(ctx context.Context, f domain.TrademarkFilter) (int64, error) {
	where, args := buildTrademarkWhere(f)
	var n int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM trademarks"+where, args...).Scan(&n)
	return n, err
}

// ─── helpers ─────────────────────────────────────────────────────────────────

type trademarkScanner interface {
	Scan(dest ...any) error
}

func scanTrademark(s trademarkScanner) (*domain.Trademark, error) {
	var t domain.Trademark
	var niceClasses []int64 // pq.Array can't scan into []int directly
	err := s.Scan(
		&t.ID, &t.ProcessNumber, &t.Name, &t.NormalizedName,
		&t.Kind, &t.Status, &t.Owner,
		pq.Array(&niceClasses), &t.ImageURL,
		&t.FilingDate, &t.PublicationDate, &t.GrantedDate,
		&t.RPIIssue, &t.CreatedAt, &t.UpdatedAt,
	)
	t.NiceClasses = make([]int, len(niceClasses))
	for i, v := range niceClasses {
		t.NiceClasses[i] = int(v)
	}
	return &t, err
}

func buildTrademarkWhere(f domain.TrademarkFilter) (string, []any) {
	var clauses []string
	var args []any
	n := 1

	add := func(clause string, v any) {
		clauses = append(clauses, fmt.Sprintf(clause, n))
		args = append(args, v)
		n++
	}

	if f.Kind != "" {
		add("kind = $%d", string(f.Kind))
	}
	if f.Status != "" {
		add("status = $%d", string(f.Status))
	}
	if f.NiceClass != nil {
		add("$%d = ANY(nice_classes)", *f.NiceClass)
	}
	if f.Search != "" {
		add("(normalized_name ILIKE $%d OR name ILIKE $%d)", "%"+strings.ToUpper(f.Search)+"%")
		// adjust: use single arg for two placeholders
		clauses[len(clauses)-1] = fmt.Sprintf("(normalized_name ILIKE $%d OR name ILIKE $%d)", n-1, n-1)
	}
	if f.FilingFrom != nil {
		add("filing_date >= $%d", f.FilingFrom)
	}
	if f.FilingUntil != nil {
		add("filing_date <= $%d", f.FilingUntil)
	}

	if len(clauses) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
