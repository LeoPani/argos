package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/pkg/httputil"
)

// DisputeHandler handles arbitration HTTP endpoints.
type DisputeHandler struct{ svc *service.DisputeService }

func NewDisputeHandler(svc *service.DisputeService) *DisputeHandler {
	return &DisputeHandler{svc: svc}
}

type openDisputeRequest struct {
	CaseNumber  string               `json:"case_number"`
	Title       string               `json:"title"`
	Summary     string               `json:"summary"`
	Kind        domain.DisputeKind   `json:"kind"`
	PatentID    *int64               `json:"patent_id,omitempty"`
	TrademarkID *int64               `json:"trademark_id,omitempty"`
}

// POST /api/v1/disputes
func (h *DisputeHandler) Open(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req openDisputeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json", "message": err.Error()})
		return
	}

	d := &domain.Dispute{
		CaseNumber:  req.CaseNumber,
		Title:       req.Title,
		Summary:     req.Summary,
		Kind:        req.Kind,
		PatentID:    req.PatentID,
		TrademarkID: req.TrademarkID,
	}

	saved, err := h.svc.Open(r.Context(), d)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, saved)
}

// GET /api/v1/disputes/{id}
func (h *DisputeHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	d, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, d)
}

// GET /api/v1/disputes
func (h *DisputeHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := domain.DisputeFilter{
		Status: domain.DisputeStatus(q.Get("status")),
		Kind:   domain.DisputeKind(q.Get("kind")),
		Search: q.Get("search"),
	}
	if ls := q.Get("limit"); ls != "" {
		if n, _ := strconv.Atoi(ls); n > 0 {
			f.Limit = n
		}
	}
	f.Normalize()

	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"items":      items,
		"pagination": map[string]any{"total": total, "limit": f.Limit, "offset": f.Offset},
	})
}

// PATCH /api/v1/disputes/{id}/status
func (h *DisputeHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	var body struct {
		Status domain.DisputeStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	if err := h.svc.Transition(r.Context(), id, body.Status); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
