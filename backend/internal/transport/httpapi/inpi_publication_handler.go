package httpapi

import (
	"net/http"
	"strconv"

	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/pkg/httputil"
)


type INPIPublicationHandler struct {
	svc *service.INPIPublicationService
}

func NewINPIPublicationHandler(svc *service.INPIPublicationService) *INPIPublicationHandler {
	return &INPIPublicationHandler{svc: svc}
}

// Stats — GET /api/v1/inpi-publications/stats
func (h *INPIPublicationHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.Stats(r.Context())
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, stats)
}

// ListUFOP — GET /api/v1/inpi-publications/ufop?limit=100
func (h *INPIPublicationHandler) ListUFOP(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	items, err := h.svc.ListUFOP(r.Context(), limit)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"count": len(items),
		"items": items,
	})
}

// Timeline — GET /api/v1/inpi-publications/timeline?limit=50
// Returns despacho counts grouped by RPI number, sorted ascending.
func (h *INPIPublicationHandler) Timeline(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	points, err := h.svc.Timeline(r.Context(), limit)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"count":  len(points),
		"points": points,
	})
}
