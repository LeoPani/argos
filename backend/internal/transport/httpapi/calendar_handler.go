package httpapi

import (
	"net/http"
	"time"

	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/pkg/httputil"
)

type CalendarHandler struct{ svc *service.CalendarService }

func NewCalendarHandler(svc *service.CalendarService) *CalendarHandler {
	return &CalendarHandler{svc: svc}
}

// Get — GET /api/v1/calendar?from=2026-01-01&to=2026-12-31
func (h *CalendarHandler) Get(w http.ResponseWriter, r *http.Request) {
	parse := func(k string) time.Time {
		v := r.URL.Query().Get(k)
		if v == "" {
			return time.Time{}
		}
		t, _ := time.Parse("2006-01-02", v)
		return t
	}
	resp, err := h.svc.Get(r.Context(), parse("from"), parse("to"))
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}
