package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/pkg/httputil"
)

// WatchlistHandler exposes CRUD + check operations.
type WatchlistHandler struct{ svc *service.WatchlistService }

func NewWatchlistHandler(svc *service.WatchlistService) *WatchlistHandler {
	return &WatchlistHandler{svc: svc}
}

type createWatchlistRequest struct {
	Label               string           `json:"label"`
	Type                domain.WatchType `json:"watch_type"`
	Query               string           `json:"query"`
	AutoDispute         bool             `json:"auto_dispute"`
	SimilarityThreshold int              `json:"similarity_threshold"`
}

// List handles GET /api/v1/watchlists
func (h *WatchlistHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context())
	if err != nil {
		httputil.Error(w, err)
		return
	}
	if items == nil {
		items = []domain.Watchlist{}
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"items": items,
		"count": len(items),
	})
}

// Create handles POST /api/v1/watchlists
func (h *WatchlistHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req createWatchlistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json", "message": err.Error()})
		return
	}

	saved, err := h.svc.Create(r.Context(), &domain.Watchlist{
		Label:               req.Label,
		Type:                req.Type,
		Query:               req.Query,
		AutoDispute:         req.AutoDispute,
		SimilarityThreshold: req.SimilarityThreshold,
	})
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, saved)
}

// Delete handles DELETE /api/v1/watchlists/{id}
func (h *WatchlistHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Check handles POST /api/v1/watchlists/{id}/check — runs a single scan.
func (h *WatchlistHandler) Check(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	updated, err := h.svc.Check(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, updated)
}

// CheckAll handles POST /api/v1/watchlists/check-all — scans every watchlist.
func (h *WatchlistHandler) CheckAll(w http.ResponseWriter, r *http.Request) {
	n, err := h.svc.CheckAll(r.Context())
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"checked": n})
}
