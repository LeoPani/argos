package httpapi

import (
	"database/sql"
	"net/http"

	"github.com/LeoPani/argos/backend/internal/service"
)

// Deps bundles all services the router needs.
type Deps struct {
	DB                *sql.DB
	PatentService     *service.PatentService
	TrademarkService  *service.TrademarkService
	DisputeService    *service.DisputeService
	PriorArtService   *service.PriorArtService
	UFOPService       *service.UFOPService
	PortfolioService  *service.PortfolioService
	StatsService      *service.StatsService
	WatchlistService  *service.WatchlistService
	ArbitrationAI     *service.ArbitrationAI
	TTContractService *service.TTContractService
	PoolService       *service.PoolService
	ChatService       *service.ChatService
	SearchService     *service.SearchService
	MetricsService    *service.MetricsService
	EnrichmentService *service.EnrichmentService
}

// NewRouter assembles the full HTTP handler chain.
//
// Route map:
//
//	GET    /health
//	POST   /api/v1/patents
//	GET    /api/v1/patents
//	GET    /api/v1/patents/{id}
//	POST   /api/v1/trademarks
//	GET    /api/v1/trademarks
//	GET    /api/v1/trademarks/{id}
//	POST   /api/v1/disputes
//	GET    /api/v1/disputes
//	GET    /api/v1/disputes/{id}
//	PATCH  /api/v1/disputes/{id}/status
//	GET    /api/v1/prior-art
func NewRouter(deps Deps) http.Handler {
	mux := http.NewServeMux()

	// ── Health ────────────────────────────────────────────────────────────
	health := NewHealthHandler(deps.DB)
	mux.HandleFunc("GET /health", health.Get)

	// ── Patents (Phase 1) ─────────────────────────────────────────────────
	patents := NewPatentHandler(deps.PatentService)
	mux.HandleFunc("POST /api/v1/patents", patents.Create)
	mux.HandleFunc("GET /api/v1/patents", patents.List)
	mux.HandleFunc("GET /api/v1/patents/{id}", patents.GetByID)

	// ── Trademarks (Phase C) ──────────────────────────────────────────────
	if deps.TrademarkService != nil {
		trademarks := NewTrademarkHandler(deps.TrademarkService)
		mux.HandleFunc("POST /api/v1/trademarks", trademarks.Create)
		mux.HandleFunc("GET /api/v1/trademarks", trademarks.List)
		mux.HandleFunc("GET /api/v1/trademarks/{id}", trademarks.GetByID)
	}

	// ── Disputes / Arbitration (Phase 5) ─────────────────────────────────
	if deps.DisputeService != nil {
		disputes := NewDisputeHandler(deps.DisputeService)
		mux.HandleFunc("POST /api/v1/disputes", disputes.Open)
		mux.HandleFunc("GET /api/v1/disputes", disputes.List)
		mux.HandleFunc("GET /api/v1/disputes/{id}", disputes.GetByID)
		mux.HandleFunc("PATCH /api/v1/disputes/{id}/status", disputes.UpdateStatus)
	}

	// ── Prior Art Search ──────────────────────────────────────────────────
	if deps.PriorArtService != nil {
		priorArt := NewPriorArtHandler(deps.PriorArtService)
		mux.HandleFunc("GET /api/v1/prior-art", priorArt.Search)
	}

	// ── Portfolio aggregation ─────────────────────────────────────────────
	if deps.PortfolioService != nil {
		portfolio := NewPortfolioHandler(deps.PortfolioService)
		mux.HandleFunc("GET /api/v1/portfolio", portfolio.Get)
	}

	// ── Stats / Dashboard BI ──────────────────────────────────────────────
	if deps.StatsService != nil {
		stats := NewStatsHandler(deps.StatsService)
		mux.HandleFunc("GET /api/v1/stats", stats.Get)
	}

	// ── Arbitration AI (subjects + verdict) ───────────────────────────────
	if deps.ArbitrationAI != nil {
		arb := NewArbitrationHandler(deps.ArbitrationAI)
		mux.HandleFunc("GET /api/v1/disputes/{id}/subjects",       arb.ListSubjects)
		mux.HandleFunc("POST /api/v1/disputes/{id}/subjects",      arb.AddSubject)
		mux.HandleFunc("DELETE /api/v1/disputes/subjects/{subjectId}", arb.DeleteSubject)
		mux.HandleFunc("POST /api/v1/disputes/{id}/analyze",       arb.Analyze)
		mux.HandleFunc("GET /api/v1/disputes/{id}/verdict",        arb.LatestVerdict)
	}

	// ── TT Contracts ──────────────────────────────────────────────────────
	if deps.TTContractService != nil {
		tt := NewTTContractHandler(deps.TTContractService)
		mux.HandleFunc("GET /api/v1/tt-contracts",                 tt.List)
		mux.HandleFunc("POST /api/v1/tt-contracts",                tt.Create)
		mux.HandleFunc("GET /api/v1/tt-contracts/{id}",            tt.GetByID)
		mux.HandleFunc("PATCH /api/v1/tt-contracts/{id}/status",   tt.UpdateStatus)
		mux.HandleFunc("DELETE /api/v1/tt-contracts/{id}",         tt.Delete)
	}

	// ── Patent Pools ──────────────────────────────────────────────────────
	if deps.PoolService != nil {
		pool := NewPoolHandler(deps.PoolService)
		mux.HandleFunc("GET /api/v1/pools",                        pool.List)
		mux.HandleFunc("POST /api/v1/pools",                       pool.Create)
		mux.HandleFunc("GET /api/v1/pools/{id}",                   pool.GetByID)
		mux.HandleFunc("DELETE /api/v1/pools/{id}",                pool.Delete)
		mux.HandleFunc("POST /api/v1/pools/{id}/members",          pool.AddMember)
		mux.HandleFunc("DELETE /api/v1/pools/{id}/members/{patentId}", pool.RemoveMember)
	}

	// ── Academic IP metrics ───────────────────────────────────────────────
	if deps.MetricsService != nil {
		m := NewMetricsHandler(deps.MetricsService)
		mux.HandleFunc("GET /api/v1/metrics",                m.Snapshot)
		mux.HandleFunc("GET /api/v1/metrics/health-score",   m.HealthScore)
		mux.HandleFunc("GET /api/v1/metrics/patent/{id}/pci",         m.PCI)
		mux.HandleFunc("GET /api/v1/metrics/patent/{id}/maintenance", m.Maintenance)
		mux.HandleFunc("GET /api/v1/metrics/inventors/{name}",        m.InventorProfile)
		mux.HandleFunc("GET /api/v1/metrics/departments",             m.HealthByDepartment)
		mux.HandleFunc("GET /api/v1/metrics/knowledge-stock",         m.KnowledgeStock)
		mux.HandleFunc("GET /api/v1/metrics/methodology",             m.Methodology)
	}

	// ── Lens.org enrichment ───────────────────────────────────────────────
	if deps.EnrichmentService != nil {
		e := NewEnrichmentHandler(deps.EnrichmentService)
		mux.HandleFunc("POST /api/v1/metrics/enrich-all",      e.EnrichAll)
		mux.HandleFunc("POST /api/v1/metrics/enrich/{id}",     e.EnrichOne)
	}

	// ── Global federated search ───────────────────────────────────────────
	if deps.SearchService != nil {
		s := NewSearchHandler(deps.SearchService)
		mux.HandleFunc("GET /api/v1/search", s.Search)
	}

	// ── Chat threads + messages ───────────────────────────────────────────
	if deps.ChatService != nil {
		chat := NewChatHandler(deps.ChatService)
		mux.HandleFunc("GET /api/v1/chat/threads",                  chat.ListThreads)
		mux.HandleFunc("POST /api/v1/chat/threads",                 chat.CreateThread)
		mux.HandleFunc("GET /api/v1/chat/threads/{id}",             chat.GetThread)
		mux.HandleFunc("DELETE /api/v1/chat/threads/{id}",          chat.DeleteThread)
		mux.HandleFunc("POST /api/v1/chat/threads/{id}/messages",   chat.AppendMessage)
		mux.HandleFunc("PATCH /api/v1/chat/threads/{id}/title",     chat.UpdateTitle)
	}

	// ── Watchlists / Alerts ───────────────────────────────────────────────
	if deps.WatchlistService != nil {
		watch := NewWatchlistHandler(deps.WatchlistService)
		mux.HandleFunc("GET /api/v1/watchlists", watch.List)
		mux.HandleFunc("POST /api/v1/watchlists", watch.Create)
		mux.HandleFunc("DELETE /api/v1/watchlists/{id}", watch.Delete)
		mux.HandleFunc("POST /api/v1/watchlists/{id}/check", watch.Check)
		mux.HandleFunc("POST /api/v1/watchlists/check-all", watch.CheckAll)
	}

	// ── UFOP Intelligence (Phase E) ───────────────────────────────────────
	if deps.UFOPService != nil {
		ufop := NewUFOPHandler(deps.UFOPService)
		mux.HandleFunc("GET /api/v1/ufop/opportunities", ufop.List)
		mux.HandleFunc("GET /api/v1/ufop/opportunities/{id}", ufop.GetByID)
		mux.HandleFunc("PATCH /api/v1/ufop/opportunities/{id}/status", ufop.UpdateStatus)
	}

	// Middleware stack (outermost → innermost):
	// CORS → Recover (catches panics) → Logging → handler
	return CORSMiddleware(RecoverMiddleware(LoggingMiddleware(mux)))
}
