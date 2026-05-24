package httpapi

import (
	"net/http"
	"strconv"

	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/pkg/httputil"
)

// MetricsHandler exposes academic IP intelligence indicators.
type MetricsHandler struct{ svc *service.MetricsService }

func NewMetricsHandler(svc *service.MetricsService) *MetricsHandler {
	return &MetricsHandler{svc: svc}
}

// Snapshot — GET /api/v1/metrics?scope=UFOP
func (h *MetricsHandler) Snapshot(w http.ResponseWriter, r *http.Request) {
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "UFOP"
	}
	resp, err := h.svc.Snapshot(r.Context(), scope)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// HealthScore — GET /api/v1/metrics/health-score?scope=UFOP
func (h *MetricsHandler) HealthScore(w http.ResponseWriter, r *http.Request) {
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "UFOP"
	}
	resp, err := h.svc.HealthScore(r.Context(), scope)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// PCI — GET /api/v1/metrics/patent/{id}/pci
func (h *MetricsHandler) PCI(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	resp, err := h.svc.PCI(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// Maintenance — GET /api/v1/metrics/patent/{id}/maintenance
func (h *MetricsHandler) Maintenance(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	resp, err := h.svc.MaintenanceFor(r.Context(), id)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "compute_failed", "message": err.Error()})
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// InventorProfile — GET /api/v1/metrics/inventors/{name}
func (h *MetricsHandler) InventorProfile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	resp, err := h.svc.InventorByName(r.Context(), name)
	if err != nil {
		httputil.JSON(w, http.StatusNotFound, map[string]any{"error": "inventor_not_found", "message": err.Error()})
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// HealthByDepartment — GET /api/v1/metrics/departments
func (h *MetricsHandler) HealthByDepartment(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.HealthByDepartment(r.Context())
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"departments": resp})
}

// KnowledgeStock — GET /api/v1/metrics/knowledge-stock?scope=UFOP
func (h *MetricsHandler) KnowledgeStock(w http.ResponseWriter, r *http.Request) {
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "UFOP"
	}
	resp, err := h.svc.KnowledgeStock(r.Context(), scope)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"series":      resp,
		"scope":       scope,
		"methodology": "Griliches_1990_perpetual_inventory",
		"depreciation_rate": 0.15,
	})
}

// Methodology — GET /api/v1/metrics/methodology
// Static metadata: formulas + references for each metric. Renderable by frontend.
func (h *MetricsHandler) Methodology(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, methodologyPayload())
}

func methodologyPayload() map[string]any {
	return map[string]any{
		"version": "1.0",
		"metrics": []map[string]any{
			{
				"id":          "autm_health_score",
				"name":        "AUTM Health Score",
				"description": "Composite 5-indicator score per AUTM Survey FY2022 + FORTEC 2023 adaptation",
				"formula":     "score = mean(P1,P2,P3,P4,P5) where P1..P5 are normalized AUTM indicators",
				"components": []map[string]string{
					{"key": "p1_disclosures",      "label": "Disclosures per inventor",     "definition": "patents / unique inventors"},
					{"key": "p2_grant_rate",       "label": "Grant rate",                    "definition": "granted / (granted + pending + failed)"},
					{"key": "p3_license_rate",     "label": "License intensity",             "definition": "patents with TT contract / total patents"},
					{"key": "p4_revenue_per_asset","label": "Revenue per asset (BRL)",       "definition": "Σ(upfront + royalty_floor) / total patents"},
					{"key": "p5_time_to_grant",    "label": "Average time to grant (days)",  "definition": "mean(now - filing_date) for classified patents"},
				},
				"references": []string{
					"AUTM Licensing Activity Survey FY2022, Association of University Technology Managers.",
					"FORTEC. Indicadores de PI em ICTs brasileiras 2023.",
				},
			},
			{
				"id":          "tt_funnel",
				"name":        "Technology Transfer Conversion Funnel",
				"description": "Standard AUTM disclosures→patents→licenses→revenue funnel",
				"formula":     "rate(stage_i+1 / stage_i) per transition",
				"references": []string{
					"AUTM Licensing Activity Survey FY2022 — Section: Conversion Rates.",
				},
			},
			{
				"id":          "hjt_diversity",
				"name":        "HJT IPC Diversity (light)",
				"description": "Hall-Jaffe-Trajtenberg (2001) Originality Index, portfolio-adapted",
				"formula":     "D = 1 - Σⱼ sⱼ²  where sⱼ = share of patents in IPC class j",
				"interpretation": "0 = fully specialized in 1 class; 0.875 → uniform over 8 classes",
				"references": []string{
					"Hall, B. H., Jaffe, A. B., & Trajtenberg, M. (2001). The NBER patent citation data file: Lessons, insights and methodological tools. NBER Working Paper 8498.",
				},
			},
			{
				"id":          "triple_helix",
				"name":        "Triple Helix Engagement Score",
				"description": "Etzkowitz-Leydesdorff (2000) U-I-G interaction model",
				"formula":     "balance(U,I,G) × 0.5 + collaboration_rate × 0.5, scaled to 0-100",
				"references": []string{
					"Etzkowitz, H., & Leydesdorff, L. (2000). The dynamics of innovation: from National Systems and 'Mode 2' to a Triple Helix of university–industry–government relations. Research Policy, 29(2), 109-123.",
					"Almeida, M., Mello, J. M. C., Etzkowitz, H. (2012). Triple Helix in Latin America. Tecnología en Marcha.",
				},
			},
			{
				"id":          "inventor_h",
				"name":        "Inventor Productivity Score (Hirsch-adapted)",
				"description": "Patent-count h-index proxy (Wong & Pang 2011) — partial without citation data",
				"formula":     "h_proxy = min(total_patents, ipc_breadth × 2)",
				"references": []string{
					"Hirsch, J. E. (2005). An index to quantify an individual's scientific research output. PNAS, 102(46), 16569-16572.",
					"Wong, P. K., & Pang, R. K. M. (2011). The h-index, h-type indices, and the science citation index database. Scientometrics, 87(1), 165-176.",
				},
			},
			{
				"id":          "maintenance_decision",
				"name":        "Patent Maintenance Decision (renewal economics)",
				"description": "Recomendação keep/license/abandon por análise de NPV vs custo de anuidades",
				"formula":     "if NPV > Σ remaining_annuities: keep; else if age<10 & licenses=0: license; else: abandon",
				"references": []string{
					"Schankerman, M., & Pakes, A. (1986). Estimates of the value of patent rights in European countries during the post-1950 period. The Economic Journal, 96(384), 1052-1076.",
					"Pakes, A. (1986). Patents as options: some estimates of the value of holding European patent stocks. Econometrica, 54(4), 755-784.",
				},
			},
			{
				"id":          "knowledge_stock",
				"name":        "Knowledge Stock (Perpetual Inventory)",
				"description": "Capital de R&D acumulado da instituição via método de inventário perpétuo",
				"formula":     "S(t) = (1 − δ) · S(t−1) + N(t)   onde δ = 0.15 (taxa de depreciação)",
				"references": []string{
					"Griliches, Z. (1990). Patent statistics as economic indicators: A survey. Journal of Economic Literature, 28(4), 1661-1707.",
				},
			},
			{
				"id":          "inventor_profile",
				"name":        "Inventor Profile (Hirsch + Lei 10.973)",
				"description": "Perfil produtivo do inventor com h-index proxy + estimativa de royalty devido por Lei 10.973/2004",
				"formula":     "royalty_devido = Σ (contract.royalty × inventor_share_pct)  por patente do inventor",
				"references": []string{
					"Lei n. 10.973/2004 (Marco Legal da Ciência, Tecnologia e Inovação) — artigos 11-13.",
					"Hirsch, J. E. (2005). An index to quantify an individual's scientific research output. PNAS, 102(46), 16569-16572.",
				},
			},
			{
				"id":          "pci_lanjouw_schankerman",
				"name":        "Patent Composite Index (PCI)",
				"description": "Lanjouw-Schankerman (2004) weighted multi-indicator patent quality",
				"formula":     "PCI = 0.46·n(fwd_cites) + 0.27·n(family_size) + 0.16·n(claims) + 0.11·n(bwd_cites)",
				"normalization": "min-max in [0,1] against L-S 2004 typical ranges",
				"references": []string{
					"Lanjouw, J. O., & Schankerman, M. (2004). Patent quality and research productivity: Measuring innovation with multiple indicators. The Economic Journal, 114(495), 441-465.",
				},
				"data_requirements": "Forward & backward citations (Lens.org Patent API). Partial mode if absent.",
			},
		},
	}
}
