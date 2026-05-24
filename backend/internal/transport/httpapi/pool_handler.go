package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/pkg/httputil"
)

// PoolHandler exposes patent pool endpoints.
type PoolHandler struct{ svc *service.PoolService }

func NewPoolHandler(svc *service.PoolService) *PoolHandler {
	return &PoolHandler{svc: svc}
}

type createPoolRequest struct {
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Kind          domain.PoolKind `json:"pool_kind"`
	RoyaltyRate   float64         `json:"royalty_rate"`
	Territory     string          `json:"territory"`
	DurationYears int             `json:"duration_years"`
}

// Create — POST /api/v1/pools
func (h *PoolHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createPoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	p := &domain.PatentPool{
		Name:          req.Name,
		Description:   req.Description,
		Kind:          req.Kind,
		RoyaltyRate:   req.RoyaltyRate,
		Territory:     req.Territory,
		DurationYears: req.DurationYears,
	}
	saved, err := h.svc.Create(r.Context(), p)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, saved)
}

// List — GET /api/v1/pools
func (h *PoolHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context())
	if err != nil {
		httputil.Error(w, err)
		return
	}
	if items == nil {
		items = []domain.PatentPool{}
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
}

// GetByID — GET /api/v1/pools/{id}
func (h *PoolHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	p, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, p)
}

// Delete — DELETE /api/v1/pools/{id}
func (h *PoolHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

type addMemberRequest struct {
	PatentID int64   `json:"patent_id"`
	SharePct float64 `json:"share_pct"`
}

// AddMember — POST /api/v1/pools/{id}/members
func (h *PoolHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	var req addMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	m, err := h.svc.AddPatent(r.Context(), id, req.PatentID, req.SharePct)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, m)
}

// RemoveMember — DELETE /api/v1/pools/{id}/members/{patentId}
func (h *PoolHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	poolID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_pool_id"})
		return
	}
	patentID, err := strconv.ParseInt(r.PathValue("patentId"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_patent_id"})
		return
	}
	if err := h.svc.RemovePatent(r.Context(), poolID, patentID); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
