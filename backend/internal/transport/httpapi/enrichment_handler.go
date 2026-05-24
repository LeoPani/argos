package httpapi

import (
	"net/http"
	"strconv"

	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/pkg/httputil"
)

// EnrichmentHandler exposes Lens.org enrichment endpoints.
type EnrichmentHandler struct{ svc *service.EnrichmentService }

func NewEnrichmentHandler(svc *service.EnrichmentService) *EnrichmentHandler {
	return &EnrichmentHandler{svc: svc}
}

// EnrichAll — POST /api/v1/metrics/enrich-all?limit=50
func (h *EnrichmentHandler) EnrichAll(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}
	stats, err := h.svc.EnrichAll(r.Context(), limit)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, stats)
}

// EnrichOne — POST /api/v1/metrics/enrich/{patent_id}
func (h *EnrichmentHandler) EnrichOne(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	result, err := h.svc.EnrichOne(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"patent_id":          result.PatentID,
		"lens_id":            result.LensID,
		"forward_citations":  len(result.ForwardCitations),
		"backward_citations": len(result.BackwardCitations),
		"family_size":        result.FamilySize,
		"claims_count":       result.ClaimsCount,
		"source":             result.Source,
	})
}
