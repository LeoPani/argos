package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/LeoPani/argos/backend/pkg/httputil"
)

// pinger is the minimal contract HealthHandler needs from *sql.DB.
// Defining a local interface (instead of importing *sql.DB) keeps the
// handler unit-testable with a fake pool.
type pinger interface {
	PingContext(ctx context.Context) error
}

// HealthHandler reports DB liveness for load balancers and k8s probes.
type HealthHandler struct {
	db pinger
}

// NewHealthHandler wires the handler. db is typically *sql.DB but
// anything implementing PingContext works.
func NewHealthHandler(db pinger) *HealthHandler {
	return &HealthHandler{db: db}
}

// Get answers GET /health. Returns 200 if the DB ping succeeds within 2s,
// 503 otherwise — the conventional response for an unhealthy dependency.
func (h *HealthHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		httputil.JSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}
