package domain

import "time"

// ChatRole identifica quem mandou a mensagem.
type ChatRole string

const (
	ChatRoleUser      ChatRole = "user"
	ChatRoleAssistant ChatRole = "assistant"
	ChatRoleSystem    ChatRole = "system"
)

// ChatThread agrupa mensagens de uma conversa.
type ChatThread struct {
	ID           int64     `json:"id"`
	Title        string    `json:"title"`
	Pinned       bool      `json:"pinned"`
	Archived     bool      `json:"archived"`
	MessageCount int       `json:"message_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Hidratado pelo service quando útil.
	Messages []ChatMessage `json:"messages,omitempty"`
}

// ChatMessage é uma única mensagem dentro de uma thread.
type ChatMessage struct {
	ID        int64     `json:"id"`
	ThreadID  int64     `json:"thread_id"`
	Role      ChatRole  `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// Validate the message before insertion.
func (m *ChatMessage) Validate() error {
	switch m.Role {
	case ChatRoleUser, ChatRoleAssistant, ChatRoleSystem:
	default:
		return ErrInvalidArg
	}
	if m.Content == "" {
		return ErrInvalidArg
	}
	return nil
}
