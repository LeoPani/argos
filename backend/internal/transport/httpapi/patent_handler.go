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

type PatentHandler struct {
	svc *service.PatentService
}

func NewPatentHandler(svc *service.PatentService) *PatentHandler {
	return &PatentHandler{svc: svc}
}

// ---- POST /api/v1/patents ----

type createPatentRequest struct {
	ApplicationNumber string     `json:"application_number"`
	Title             string     `json:"title"`
	Abstract          string     `json:"abstract"`
	Applicant         string     `json:"applicant"`
	Inventors         []string   `json:"inventors"`
	FilingDate        *time.Time `json:"filing_date"`
	PublicationDate   *time.Time `json:"publication_date"`
	IPCCode           string     `json:"ipc_code"`
	RPIIssue          string     `json:"rpi_issue"`
}

func (h *PatentHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req createPatentRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{
			"error":   "invalid_json",
			"message": err.Error(),
		})
		return
	}

	p := &domain.Patent{
		ApplicationNumber: req.ApplicationNumber,
		Title:             req.Title,
		Abstract:          req.Abstract,
		Applicant:         req.Applicant,
		Inventors:         req.Inventors,
		FilingDate:        req.FilingDate,
		PublicationDate:   req.PublicationDate,
		IPCCode:           req.IPCCode,
		RPIIssue:          req.RPIIssue,
	}

	saved, err := h.svc.Ingest(r.Context(), p)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, saved)
}

// ---- GET /api/v1/patents/{id} ----

func (h *PatentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{
			"error":   "invalid_id",
			"message": "id must be a positive integer",
		})
		return
	}

	p, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, p)
}

// ---- GET /api/v1/patents ----

func (h *PatentHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := domain.PatentFilter{
		Search:   q.Get("search"),
		RPIIssue: q.Get("rpi"),
		Status:   domain.PatentStatus(q.Get("status")),
	}

	if catStr := q.Get("category"); catStr != "" {
		if n, err := strconv.Atoi(catStr); err == nil {
			cat := domain.IPCCategory(n)
			f.Category = &cat
		}
	}
	if limitStr := q.Get("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil {
			f.Limit = n
		}
	}
	if offsetStr := q.Get("offset"); offsetStr != "" {
		if n, err := strconv.Atoi(offsetStr); err == nil {
			f.Offset = n
		}
	}

	// Normalize HERE so the response carries the actual (clamped) values
	// in pagination metadata. The service normalizes its own copy too,
	// but Go passes structs by value — the handler's copy stays raw.
	f.Normalize()

	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]any{
		"items": items,
		"pagination": map[string]any{
			"total":  total,
			"limit":  f.Limit,
			"offset": f.Offset,
		},
	})
}
