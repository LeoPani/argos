package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/pkg/httputil"
)

// TrademarkHandler handles trademark HTTP endpoints.
type TrademarkHandler struct{ svc *service.TrademarkService }

func NewTrademarkHandler(svc *service.TrademarkService) *TrademarkHandler {
	return &TrademarkHandler{svc: svc}
}

type createTrademarkRequest struct {
	ProcessNumber   string                `json:"process_number"`
	Name            string                `json:"name"`
	Kind            domain.TrademarkKind  `json:"kind"`
	Status          domain.TrademarkStatus `json:"status"`
	Owner           string                `json:"owner"`
	NiceClasses     []int                 `json:"nice_classes"`
	ImageURL        string                `json:"image_url"`
	FilingDate      *time.Time            `json:"filing_date"`
	PublicationDate *time.Time            `json:"publication_date"`
	GrantedDate     *time.Time            `json:"granted_date"`
	RPIIssue        string                `json:"rpi_issue"`
}

// POST /api/v1/trademarks
func (h *TrademarkHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req createTrademarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json", "message": err.Error()})
		return
	}

	t := &domain.Trademark{
		ProcessNumber:   req.ProcessNumber,
		Name:            req.Name,
		Kind:            req.Kind,
		Status:          req.Status,
		Owner:           req.Owner,
		NiceClasses:     req.NiceClasses,
		ImageURL:        req.ImageURL,
		FilingDate:      req.FilingDate,
		PublicationDate: req.PublicationDate,
		GrantedDate:     req.GrantedDate,
		RPIIssue:        req.RPIIssue,
	}

	saved, err := h.svc.Create(r.Context(), t)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, saved)
}

// GET /api/v1/trademarks/{id}
func (h *TrademarkHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	t, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, t)
}

// GET /api/v1/trademarks
func (h *TrademarkHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := domain.TrademarkFilter{
		Kind:   domain.TrademarkKind(q.Get("kind")),
		Status: domain.TrademarkStatus(q.Get("status")),
		Search: q.Get("search"),
	}
	if ls := q.Get("limit"); ls != "" {
		if n, err := strconv.Atoi(ls); err == nil {
			f.Limit = n
		}
	}
	if os := q.Get("offset"); os != "" {
		if n, err := strconv.Atoi(os); err == nil {
			f.Offset = n
		}
	}
	f.Normalize()

	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"items":      items,
		"pagination": map[string]any{"total": total, "limit": f.Limit, "offset": f.Offset},
	})
}
