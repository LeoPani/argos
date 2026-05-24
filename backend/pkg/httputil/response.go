// Package httputil holds small, reusable HTTP helpers. It lives under
// pkg/ (not internal/) on purpose: these helpers are intentionally
// generic enough to be reused across multiple services if we ever
// extract this code.
package httputil

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/LeoPani/argos/backend/internal/domain"
)

// JSON writes payload as a JSON response with the given HTTP status code.
// If encoding fails (rare, usually only with cyclic structs), the error
// is logged but the connection has already had headers flushed.
func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("httputil: failed to encode response", "err", err)
	}
}

// Error writes a structured JSON error response. The HTTP status code and
// machine-readable error code are derived from the domain sentinel inside
// err via errors.Is. Unknown errors map to 500 Internal Server Error.
//
// Example output:
//
//	{ "error": "not_found", "message": "get patent id=42: domain: resource not found" }
func Error(w http.ResponseWriter, err error) {
	status, code := mapError(err)
	JSON(w, status, map[string]any{
		"error":   code,
		"message": err.Error(),
	})
}

// mapError translates domain sentinels into HTTP status + machine code.
// Anything that isn't a recognized domain error becomes a 500.
func mapError(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, domain.ErrDuplicate):
		return http.StatusConflict, "duplicate"
	case errors.Is(err, domain.ErrInvalidArg):
		return http.StatusBadRequest, "invalid_argument"
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, "conflict"
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized, "unauthorized"
	default:
		return http.StatusInternalServerError, "internal_error"
	}
}
