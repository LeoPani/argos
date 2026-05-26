package httpapi

import (
	"net/http"
	"strconv"

	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/pkg/httputil"
)

// SemanticSearchHandler expõe /api/v1/semantic-search
// TF-IDF + cosine, sem deps externas. Documentado em METHODOLOGY.md.
type SemanticSearchHandler struct {
	svc *service.SemanticSearchService
}

func NewSemanticSearchHandler(svc *service.SemanticSearchService) *SemanticSearchHandler {
	return &SemanticSearchHandler{svc: svc}
}

// Search — GET /api/v1/semantic-search?q=...&top_k=20
func (h *SemanticSearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	topK := 20
	if raw := r.URL.Query().Get("top_k"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			topK = n
		}
	}
	resp, err := h.svc.Search(r.Context(), q, topK)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}
