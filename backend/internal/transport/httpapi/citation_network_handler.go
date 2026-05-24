package httpapi

import (
	"net/http"
	"strconv"

	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/pkg/httputil"
)

type CitationNetworkHandler struct{ svc *service.CitationNetworkService }

func NewCitationNetworkHandler(svc *service.CitationNetworkService) *CitationNetworkHandler {
	return &CitationNetworkHandler{svc: svc}
}

// Build — GET /api/v1/citations/network/{id}
func (h *CitationNetworkHandler) Build(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	resp, err := h.svc.Build(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}
