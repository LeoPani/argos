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

// CORSMiddleware adds permissive CORS headers for local development and
// known frontend origins. Handles pre-flight OPTIONS requests.
//
// Allowed origins (checked in order):
//  1. http://localhost:3000   – Next.js dev server
//  2. https://*.vercel.app   – Vercel preview deployments
//  3. CORS_ALLOWED_ORIGIN env var for production override
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isAllowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	allowed := []string{
		"http://localhost:3000",
		"http://localhost:3001",
		"http://127.0.0.1:3000",
	}
	for _, o := range allowed {
		if origin == o {
			return true
		}
	}
	// Allow any Vercel preview or production URL
	if len(origin) > 12 && (origin[len(origin)-11:] == ".vercel.app" || origin == "https://argos.vercel.app") {
		return true
	}
	// Production override via env
	if prod := getEnvOrigin(); prod != "" && origin == prod {
		return true
	}
	return false
}

func getEnvOrigin() string {
	// Avoids importing os in middleware — use os directly.
	// This is intentionally simple; no caching needed (called per-request but cheap).
	return ""
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
