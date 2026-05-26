package httpapi

import (
	"net/http"

	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/pkg/httputil"
)

// PriorArtHandler handles prior art search endpoints.
type PriorArtHandler struct{ svc *service.PriorArtService }

func NewPriorArtHandler(svc *service.PriorArtService) *PriorArtHandler {
	return &PriorArtHandler{svc: svc}
}

// GET /api/v1/prior-art?q=...&kind=patent|trademark|both
func (h *PriorArtHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "q is required"})
		return
	}
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = "both"
	}

	result, err := h.svc.Search(r.Context(), q, kind)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, result)
}
