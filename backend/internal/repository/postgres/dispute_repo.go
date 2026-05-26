package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/repository"
)

// DisputeRepo implements repository.DisputeRepository.
type DisputeRepo struct{ db *sql.DB }

var _ repository.DisputeRepository = (*DisputeRepo)(nil)

func NewDisputeRepo(db *sql.DB) *DisputeRepo { return &DisputeRepo{db: db} }

const selectDisputeColumns = `
	id, case_number, title, summary, kind, status,
	patent_id, trademark_id, opened_at, resolved_at,
	created_at, updated_at`

func (r *DisputeRepo) Insert(ctx context.Context, d *domain.Dispute) error {
	const q = `
		INSERT INTO disputes (case_number, title, summary, kind, status, patent_id, trademark_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, opened_at, created_at, updated_at`

	status := d.Status
	if status == "" {
		status = domain.DisputeStatusOpen
	}

	err := r.db.QueryRowContext(ctx, q,
		d.CaseNumber, d.Title, d.Summary, string(d.Kind), string(status),
		d.PatentID, d.TrademarkID,
	).Scan(&d.ID, &d.OpenedAt, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert dispute %q: %w", d.CaseNumber, err)
	}
	d.Status = status

	// Record opening event.
	_ = r.AddEvent(ctx, &domain.DisputeEvent{
		DisputeID: d.ID,
		EventType: "dispute_opened",
		Payload:   `{"auto":true}`,
	})
	return nil
}

func (r *DisputeRepo) GetByID(ctx context.Context, id int64) (*domain.Dispute, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+selectDisputeColumns+" FROM disputes WHERE id=$1", id)
	d, err := scanDispute(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("dispute id=%d: %w", id, domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("dispute id=%d: %w", id, err)
	}

	if err := r.loadRelations(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (r *DisputeRepo) List(ctx context.Context, f domain.DisputeFilter) ([]domain.Dispute, error) {
	where, args := buildDisputeWhere(f)
	q := "SELECT " + selectDisputeColumns + " FROM disputes" + where +
		" ORDER BY opened_at DESC, id DESC" +
		fmt.Sprintf(" LIMIT %d OFFSET %d", f.Limit, f.Offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list disputes: %w", err)
	}
	defer rows.Close()

	var out []domain.Dispute
	for rows.Next() {
		d, err := scanDisputeRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *DisputeRepo) Count(ctx context.Context, f domain.DisputeFilter) (int64, error) {
	where, args := buildDisputeWhere(f)
	var n int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM disputes"+where, args...).Scan(&n)
	return n, err
}

func (r *DisputeRepo) UpdateStatus(ctx context.Context, id int64, status domain.DisputeStatus) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE disputes SET status=$1 WHERE id=$2", string(status), id)
	if err != nil {
		return fmt.Errorf("update dispute status id=%d: %w", id, err)
	}
	_ = r.AddEvent(ctx, &domain.DisputeEvent{
		DisputeID: id,
		EventType: "status_changed",
		Payload:   fmt.Sprintf(`{"new_status":%q}`, status),
	})
	return nil
}

func (r *DisputeRepo) AddEvent(ctx context.Context, e *domain.DisputeEvent) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO dispute_events (dispute_id, actor_id, event_type, payload)
		 VALUES ($1,$2,$3,$4)`,
		e.DisputeID, e.ActorID, e.EventType, e.Payload)
	return err
}

func (r *DisputeRepo) AddDocument(ctx context.Context, doc *domain.DisputeDocument) error {
	const q = `
		INSERT INTO dispute_documents
		(dispute_id, uploaded_by, title, description, storage_path, hash_sha256, size_bytes, mime_type)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, uploaded_at`
	return r.db.QueryRowContext(ctx, q,
		doc.DisputeID, doc.UploadedBy, doc.Title, doc.Description,
		doc.StoragePath, doc.HashSHA256, doc.SizeBytes, doc.MimeType,
	).Scan(&doc.ID, &doc.UploadedAt)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

type disputeRow interface{ Scan(dest ...any) error }

func scanDisputeBase(s disputeRow) (*domain.Dispute, error) {
	var d domain.Dispute
	err := s.Scan(
		&d.ID, &d.CaseNumber, &d.Title, &d.Summary,
		&d.Kind, &d.Status, &d.PatentID, &d.TrademarkID,
		&d.OpenedAt, &d.ResolvedAt, &d.CreatedAt, &d.UpdatedAt,
	)
	return &d, err
}

func scanDispute(s disputeRow) (*domain.Dispute, error)     { return scanDisputeBase(s) }
func scanDisputeRows(s disputeRow) (*domain.Dispute, error) { return scanDisputeBase(s) }

func (r *DisputeRepo) loadRelations(ctx context.Context, d *domain.Dispute) error {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, dispute_id, name, role, email, document, joined_at FROM dispute_parties WHERE dispute_id=$1",
		d.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var p domain.DisputeParty
		if err := rows.Scan(&p.ID, &p.DisputeID, &p.Name, &p.Role, &p.Email, &p.Document, &p.JoinedAt); err != nil {
			return err
		}
		d.Parties = append(d.Parties, p)
	}

	evRows, err := r.db.QueryContext(ctx,
		"SELECT id, dispute_id, actor_id, event_type, payload, occurred_at FROM dispute_events WHERE dispute_id=$1 ORDER BY occurred_at",
		d.ID)
	if err != nil {
		return err
	}
	defer evRows.Close()
	for evRows.Next() {
		var e domain.DisputeEvent
		if err := evRows.Scan(&e.ID, &e.DisputeID, &e.ActorID, &e.EventType, &e.Payload, &e.OccurredAt); err != nil {
			return err
		}
		d.Events = append(d.Events, e)
	}
	return evRows.Err()
}

func buildDisputeWhere(f domain.DisputeFilter) (string, []any) {
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
	if f.Status != "" {
		add("status = $%d", string(f.Status))
	}
	if f.Kind != "" {
		add("kind = $%d", string(f.Kind))
	}
	if f.Search != "" {
		clauses = append(clauses, fmt.Sprintf("(title ILIKE $%d OR summary ILIKE $%d)", n, n))
		args = append(args, "%"+f.Search+"%")
		n++
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
