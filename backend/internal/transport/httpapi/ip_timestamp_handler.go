package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/service"
)

// IPTimestampHandler exposes proof-of-existence endpoints.
type IPTimestampHandler struct{ svc *service.IPTimestampService }

func NewIPTimestampHandler(svc *service.IPTimestampService) *IPTimestampHandler {
	return &IPTimestampHandler{svc: svc}
}

// Create  POST /api/v1/timestamps
func (h *IPTimestampHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req service.IPTimestampCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	res, err := h.svc.Create(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

// List  GET /api/v1/timestamps
func (h *IPTimestampHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	items, total, err := h.svc.List(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	if items == nil {
		items = []domain.IPTimestamp{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"items": items,
		"pagination": map[string]any{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetByID  GET /api/v1/timestamps/{id}
func (h *IPTimestampHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	t, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

// Verify  GET /api/v1/timestamps/{id}/verify
func (h *IPTimestampHandler) Verify(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	ok, recomputed, err := h.svc.Verify(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"valid":              ok,
		"recomputed_hash":    recomputed,
		"integrity_message":  map[bool]string{true: "✓ Integridade confirmada", false: "✗ Hash não confere — registro adulterado"}[ok],
	})
}
