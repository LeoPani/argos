// Package service — ChatService persists threads + messages of the
// Argos assistant. Title generation is naive: takes the first user
// message and truncates to 60 chars. Doesn't call Claude itself;
// that's the Next.js BFF's job. Backend is just storage.
package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/LeoPani/argos/backend/internal/domain"
)

// ChatStorage abstracts the repo for testing.
type ChatStorage interface {
	CreateThread(ctx context.Context, t *domain.ChatThread) error
	GetThread(ctx context.Context, id int64) (*domain.ChatThread, error)
	ListThreads(ctx context.Context, includeArchived bool) ([]domain.ChatThread, error)
	UpdateThreadTitle(ctx context.Context, id int64, title string) error
	DeleteThread(ctx context.Context, id int64) error
	AppendMessage(ctx context.Context, m *domain.ChatMessage) error
	ListMessages(ctx context.Context, threadID int64) ([]domain.ChatMessage, error)
}

type ChatService struct{ repo ChatStorage }

func NewChatService(repo ChatStorage) *ChatService { return &ChatService{repo: repo} }

// CreateThread starts a new conversation with a derived title.
func (s *ChatService) CreateThread(ctx context.Context, firstMessage string) (*domain.ChatThread, error) {
	title := deriveTitle(firstMessage)
	t := &domain.ChatThread{Title: title}
	if err := s.repo.CreateThread(ctx, t); err != nil {
		return nil, fmt.Errorf("chat: create thread: %w", err)
	}
	return t, nil
}

func (s *ChatService) ListThreads(ctx context.Context) ([]domain.ChatThread, error) {
	return s.repo.ListThreads(ctx, false)
}

func (s *ChatService) GetThread(ctx context.Context, id int64) (*domain.ChatThread, error) {
	t, err := s.repo.GetThread(ctx, id)
	if err != nil {
		return nil, err
	}
	t.Messages, err = s.repo.ListMessages(ctx, id)
	return t, err
}

func (s *ChatService) DeleteThread(ctx context.Context, id int64) error {
	return s.repo.DeleteThread(ctx, id)
}

func (s *ChatService) AppendMessage(ctx context.Context, m *domain.ChatMessage) error {
	if err := m.Validate(); err != nil {
		return fmt.Errorf("chat: %w", err)
	}
	return s.repo.AppendMessage(ctx, m)
}

func (s *ChatService) UpdateTitle(ctx context.Context, id int64, title string) error {
	if title == "" {
		return fmt.Errorf("title: %w", domain.ErrInvalidArg)
	}
	return s.repo.UpdateThreadTitle(ctx, id, title)
}

// deriveTitle takes the first user message and produces a short title.
// Strips newlines, truncates at word boundary near 60 chars.
func deriveTitle(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if s == "" {
		return "Nova conversa"
	}
	const max = 60
	if len(s) <= max {
		return s
	}
	cut := s[:max]
	if idx := strings.LastIndex(cut, " "); idx > 30 {
		cut = cut[:idx]
	}
	return cut + "…"
}
