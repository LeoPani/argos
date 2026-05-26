package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/pkg/httputil"
)

// UFOPHandler handles UFOP Intelligence endpoints.
type UFOPHandler struct{ svc *service.UFOPService }

// NewUFOPHandler creates a new UFOP handler.
func NewUFOPHandler(svc *service.UFOPService) *UFOPHandler {
	return &UFOPHandler{svc: svc}
}

// List handles GET /api/v1/ufop/opportunities
//
// Query params:
//
//	source           = oai | portal | lens
//	level            = high | medium | low
//	status           = new | reviewed | converted | dismissed
//	q                = full-text search (title / abstract ILIKE)
//	department       = substring (case-insensitive)
//	patentable_only  = "true" → exclui is_patentable=false (Art. 10 LPI)
//	limit            = default 50, max 500
//	offset           = default 0
func (h *UFOPHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	f := domain.UFOPFilter{
		Source:         domain.UFOPSource(q.Get("source")),
		Level:          domain.UFOPOpportunityLevel(q.Get("level")),
		Status:         domain.UFOPOpportunityStatus(q.Get("status")),
		Search:         q.Get("q"),
		DepartmentLike: q.Get("department"),
		PatentableOnly: q.Get("patentable_only") == "true",
	}
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			f.Limit = v
		}
	}
	if o := q.Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil {
			f.Offset = v
		}
	}

	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	if items == nil {
		items = []domain.UFOPOpportunity{}
	}

	httputil.JSON(w, http.StatusOK, map[string]any{
		"items": items,
		"pagination": map[string]any{
			"total":  total,
			"limit":  f.Limit,
			"offset": f.Offset,
		},
	})
}

// GetByID handles GET /api/v1/ufop/opportunities/{id}
func (h *UFOPHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}

	opp, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, opp)
}

// UpdateStatus handles PATCH /api/v1/ufop/opportunities/{id}/status
//
// Body: {"status": "reviewed" | "converted" | "dismissed" | "new"}
func (h *UFOPHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}

	var body struct {
		Status domain.UFOPOpportunityStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json", "message": err.Error()})
		return
	}

	if err := h.svc.UpdateStatus(r.Context(), id, body.Status); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]any{"status": string(body.Status)})
}
