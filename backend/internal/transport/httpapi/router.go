package httpapi

import (
	"database/sql"
	"net/http"

	"github.com/LeoPani/argos/backend/internal/service"
)

// Deps bundles everything NewRouter needs to construct the handler chain.
// Grouping dependencies in a struct (instead of a long parameter list)
// keeps main.go readable and makes future additions painless.
type Deps struct {
	DB            *sql.DB
	PatentService *service.PatentService
}

// NewRouter assembles the application's HTTP handler chain.
//
// Route map:
//
//	GET  /health                       -> liveness probe
//	POST /api/v1/patents               -> create + classify a patent
//	GET  /api/v1/patents               -> list (with filters/pagination)
//	GET  /api/v1/patents/{id}          -> single patent by id
//
// Middleware order (outermost first):
//   1. RecoverMiddleware  — catches panics before they kill the process
//   2. LoggingMiddleware  — observes the recovered request/response
//
// Recovery wraps Logging so that even panicking handlers still emit a
// structured log line (the panic shows up as status=500).
func NewRouter(deps Deps) http.Handler {
	mux := http.NewServeMux()

	// Build handlers
	health := NewHealthHandler(deps.DB)
	patents := NewPatentHandler(deps.PatentService)

	// Health
	mux.HandleFunc("GET /health", health.Get)

	// Patents (Phase 1)
	mux.HandleFunc("POST /api/v1/patents", patents.Create)
	mux.HandleFunc("GET /api/v1/patents", patents.List)
	mux.HandleFunc("GET /api/v1/patents/{id}", patents.GetByID)

	// Apply middleware (outermost first).
	return RecoverMiddleware(LoggingMiddleware(mux))
}
