package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/LeoPani/argos/backend/internal/domain"
)

// ArbitrationRepo persists dispute subjects and AI verdicts.
type ArbitrationRepo struct{ db *sql.DB }

func NewArbitrationRepo(db *sql.DB) *ArbitrationRepo { return &ArbitrationRepo{db: db} }

// ─── Subjects ────────────────────────────────────────────────────────────────

func (r *ArbitrationRepo) AddSubject(ctx context.Context, s *domain.DisputeSubject) error {
	const q = `
		INSERT INTO dispute_subjects (dispute_id, kind, ref_id, label, party_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	if len(s.Metadata) == 0 {
		s.Metadata = []byte("{}")
	}

	err := r.db.QueryRowContext(ctx, q,
		s.DisputeID, string(s.Kind), s.RefID, s.Label, s.PartyID, s.Metadata,
	).Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		return fmt.Errorf("add subject: %w", err)
	}
	return nil
}

func (r *ArbitrationRepo) ListSubjects(ctx context.Context, disputeID int64) ([]domain.DisputeSubject, error) {
	const q = `
		SELECT id, dispute_id, kind, ref_id, label, party_id, metadata, created_at
		FROM dispute_subjects
		WHERE dispute_id = $1
		ORDER BY created_at`

	rows, err := r.db.QueryContext(ctx, q, disputeID)
	if err != nil {
		return nil, fmt.Errorf("list subjects: %w", err)
	}
	defer rows.Close()

	var out []domain.DisputeSubject
	for rows.Next() {
		var (
			s    domain.DisputeSubject
			kind string
			meta sql.NullString
		)
		if err := rows.Scan(&s.ID, &s.DisputeID, &kind, &s.RefID, &s.Label,
			&s.PartyID, &meta, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.Kind = domain.SubjectKind(kind)
		if meta.Valid {
			s.Metadata = json.RawMessage(meta.String)
		} else {
			s.Metadata = []byte("{}")
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *ArbitrationRepo) DeleteSubject(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM dispute_subjects WHERE id = $1", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("subject %d: %w", id, domain.ErrNotFound)
	}
	return nil
}

// ─── Verdicts ────────────────────────────────────────────────────────────────

func (r *ArbitrationRepo) SaveVerdict(ctx context.Context, v *domain.ArbitrationVerdict) error {
	const q = `
		INSERT INTO arbitration_verdicts
			(dispute_id, winner_subject_id, confidence, method, summary, reasoning)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	if len(v.Reasoning) == 0 {
		v.Reasoning = []byte("{}")
	}

	err := r.db.QueryRowContext(ctx, q,
		v.DisputeID, v.WinnerSubjectID, v.Confidence, string(v.Method),
		v.Summary, v.Reasoning,
	).Scan(&v.ID, &v.CreatedAt)
	if err != nil {
		return fmt.Errorf("save verdict: %w", err)
	}
	return nil
}

func (r *ArbitrationRepo) LatestVerdict(ctx context.Context, disputeID int64) (*domain.ArbitrationVerdict, error) {
	const q = `
		SELECT id, dispute_id, winner_subject_id, confidence, method, summary, reasoning, created_at
		FROM arbitration_verdicts
		WHERE dispute_id = $1
		ORDER BY created_at DESC
		LIMIT 1`

	var (
		v       domain.ArbitrationVerdict
		method  string
		reason  sql.NullString
	)
	err := r.db.QueryRowContext(ctx, q, disputeID).Scan(
		&v.ID, &v.DisputeID, &v.WinnerSubjectID, &v.Confidence,
		&method, &v.Summary, &reason, &v.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("verdict for dispute %d: %w", disputeID, domain.ErrNotFound)
	}
	if err != nil {
		return nil, err
	}
	v.Method = domain.VerdictMethod(method)
	if reason.Valid {
		v.Reasoning = json.RawMessage(reason.String)
	} else {
		v.Reasoning = []byte("{}")
	}
	return &v, nil
}
