package httpapi

import (
	"net/http"
	"strconv"

	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/pkg/httputil"
)

type TTTemplateHandler struct{ svc *service.TTTemplateService }

func NewTTTemplateHandler(svc *service.TTTemplateService) *TTTemplateHandler {
	return &TTTemplateHandler{svc: svc}
}

// FromUFOP — GET /api/v1/tt-template/from-ufop/{oppID}
func (h *TTTemplateHandler) FromUFOP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("oppID"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	tpl, err := h.svc.FromUFOPOpportunity(r.Context(), id)
	if err != nil {
		httputil.JSON(w, http.StatusNotFound, map[string]any{"error": "not_found", "message": err.Error()})
		return
	}
	httputil.JSON(w, http.StatusOK, tpl)
}
