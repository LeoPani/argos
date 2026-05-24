package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/pkg/httputil"
)

// ChatHandler exposes thread + message endpoints.
type ChatHandler struct{ svc *service.ChatService }

func NewChatHandler(svc *service.ChatService) *ChatHandler {
	return &ChatHandler{svc: svc}
}

// ListThreads — GET /api/v1/chat/threads
func (h *ChatHandler) ListThreads(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListThreads(r.Context())
	if err != nil {
		httputil.Error(w, err)
		return
	}
	if items == nil {
		items = []domain.ChatThread{}
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
}

type createThreadRequest struct {
	FirstMessage string `json:"first_message"`
}

// CreateThread — POST /api/v1/chat/threads
func (h *ChatHandler) CreateThread(w http.ResponseWriter, r *http.Request) {
	var req createThreadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	t, err := h.svc.CreateThread(r.Context(), req.FirstMessage)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, t)
}

// GetThread — GET /api/v1/chat/threads/{id} (with messages hydrated)
func (h *ChatHandler) GetThread(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	t, err := h.svc.GetThread(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, t)
}

// DeleteThread — DELETE /api/v1/chat/threads/{id}
func (h *ChatHandler) DeleteThread(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	if err := h.svc.DeleteThread(r.Context(), id); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type appendMessageRequest struct {
	Role    domain.ChatRole `json:"role"`
	Content string          `json:"content"`
}

// AppendMessage — POST /api/v1/chat/threads/{id}/messages
func (h *ChatHandler) AppendMessage(w http.ResponseWriter, r *http.Request) {
	threadID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	var req appendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	m := &domain.ChatMessage{
		ThreadID: threadID,
		Role:     req.Role,
		Content:  req.Content,
	}
	if err := h.svc.AppendMessage(r.Context(), m); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, m)
}

type updateTitleRequest struct {
	Title string `json:"title"`
}

// UpdateTitle — PATCH /api/v1/chat/threads/{id}/title
func (h *ChatHandler) UpdateTitle(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	var req updateTitleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	if err := h.svc.UpdateTitle(r.Context(), id, req.Title); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"title": req.Title})
}
