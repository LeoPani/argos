package httpapi

import (
	"net/http"
	"strconv"

	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/pkg/httputil"
)

type MarketplaceHandler struct{ svc *service.MarketplaceService }

func NewMarketplaceHandler(svc *service.MarketplaceService) *MarketplaceHandler {
	return &MarketplaceHandler{svc: svc}
}

// List — GET /api/v1/marketplace?ipc=C&q=lithium&limit=20
func (h *MarketplaceHandler) List(w http.ResponseWriter, r *http.Request) {
	ipc := r.URL.Query().Get("ipc")
	q := r.URL.Query().Get("q")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}
	resp, err := h.svc.List(r.Context(), ipc, q, limit)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}
