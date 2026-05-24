package httpapi

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// statusRecorder intercepts the response status so middleware can log it.
// Without this, http.ResponseWriter's status is opaque.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader records the status and forwards the call. We don't override
// Write — Go's net/http calls WriteHeader(200) implicitly before the first
// Write, but we still catch that path via the init value below.
func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

// LoggingMiddleware emits one structured log line per request.
//
// Example: {"msg":"request","method":"GET","path":"/health","status":200,"duration_ms":3}
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// Default to 200 in case the handler never calls WriteHeader
		// explicitly (e.g. on a body-only response).
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", r.RemoteAddr,
		)
	})
}

// RecoverMiddleware turns a panic into a 500 response and a structured
// log. Without this, a panic in a handler crashes the entire process.
func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered",
					"panic", rec,
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal_error","message":"server panic"}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
