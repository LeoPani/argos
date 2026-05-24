package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/LeoPani/argos/backend/internal/domain"
)

// ChatRepo persists chat threads and their messages.
type ChatRepo struct{ db *sql.DB }

func NewChatRepo(db *sql.DB) *ChatRepo { return &ChatRepo{db: db} }

const chatThreadCols = `id, title, pinned, archived, message_count, created_at, updated_at`

// ─── Threads ─────────────────────────────────────────────────────────────────

func (r *ChatRepo) CreateThread(ctx context.Context, t *domain.ChatThread) error {
	const q = `
		INSERT INTO chat_threads (title, pinned, archived)
		VALUES ($1, $2, $3)
		RETURNING id, message_count, created_at, updated_at`

	return r.db.QueryRowContext(ctx, q, t.Title, t.Pinned, t.Archived).
		Scan(&t.ID, &t.MessageCount, &t.CreatedAt, &t.UpdatedAt)
}

func (r *ChatRepo) GetThread(ctx context.Context, id int64) (*domain.ChatThread, error) {
	row := r.db.QueryRowContext(ctx, "SELECT "+chatThreadCols+" FROM chat_threads WHERE id=$1", id)
	t, err := scanThread(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("thread id=%d: %w", id, domain.ErrNotFound)
	}
	return t, err
}

func (r *ChatRepo) ListThreads(ctx context.Context, includeArchived bool) ([]domain.ChatThread, error) {
	q := "SELECT " + chatThreadCols + " FROM chat_threads"
	if !includeArchived {
		q += " WHERE archived = FALSE"
	}
	q += " ORDER BY pinned DESC, updated_at DESC"

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.ChatThread
	for rows.Next() {
		t, err := scanThread(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (r *ChatRepo) UpdateThreadTitle(ctx context.Context, id int64, title string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE chat_threads SET title=$1 WHERE id=$2", title, id)
	return err
}

func (r *ChatRepo) DeleteThread(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM chat_threads WHERE id=$1", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("thread id=%d: %w", id, domain.ErrNotFound)
	}
	return nil
}

// ─── Messages ────────────────────────────────────────────────────────────────

func (r *ChatRepo) AppendMessage(ctx context.Context, m *domain.ChatMessage) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx, `
		INSERT INTO chat_messages (thread_id, role, content)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`,
		m.ThreadID, string(m.Role), m.Content,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return fmt.Errorf("append message: %w", err)
	}

	// Bump thread counter + updated_at (the trigger handles updated_at).
	if _, err := tx.ExecContext(ctx,
		"UPDATE chat_threads SET message_count = message_count + 1 WHERE id=$1",
		m.ThreadID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ChatRepo) ListMessages(ctx context.Context, threadID int64) ([]domain.ChatMessage, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, thread_id, role, content, created_at
		FROM chat_messages
		WHERE thread_id = $1
		ORDER BY created_at`, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.ChatMessage
	for rows.Next() {
		var (
			m    domain.ChatMessage
			role string
		)
		if err := rows.Scan(&m.ID, &m.ThreadID, &role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Role = domain.ChatRole(role)
		out = append(out, m)
	}
	return out, rows.Err()
}

// ─── helpers ──────────────────────────────────────────────────────────────────

type threadScanner interface{ Scan(dest ...any) error }

func scanThread(s threadScanner) (*domain.ChatThread, error) {
	var t domain.ChatThread
	err := s.Scan(&t.ID, &t.Title, &t.Pinned, &t.Archived, &t.MessageCount, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
