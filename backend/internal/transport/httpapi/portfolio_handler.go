package httpapi

import (
	"net/http"

	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/pkg/httputil"
)

// PortfolioHandler serves GET /api/v1/portfolio.
type PortfolioHandler struct{ svc *service.PortfolioService }

func NewPortfolioHandler(svc *service.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{svc: svc}
}

// Get returns the full portfolio aggregation (patents + trademarks + costs + AI opps).
func (h *PortfolioHandler) Get(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Get(r.Context())
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}
