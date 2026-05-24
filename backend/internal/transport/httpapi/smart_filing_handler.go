package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/pkg/httputil"
)

// SmartFilingHandler exposes the NIT-UFOP filing assistant.
type SmartFilingHandler struct{ svc *service.SmartFilingService }

func NewSmartFilingHandler(svc *service.SmartFilingService) *SmartFilingHandler {
	return &SmartFilingHandler{svc: svc}
}

// Analyze — POST /api/v1/smart-filing
//
//	body: { "title": "...", "abstract": "...", "field": "..." }
func (h *SmartFilingHandler) Analyze(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB cap
	var in service.FilingInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json", "message": err.Error()})
		return
	}

	resp, err := h.svc.Analyze(r.Context(), in)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "analyze_failed", "message": err.Error()})
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}
