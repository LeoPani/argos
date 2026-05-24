package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/pkg/httputil"
)

// TTContractHandler exposes TT contract endpoints.
type TTContractHandler struct{ svc *service.TTContractService }

func NewTTContractHandler(svc *service.TTContractService) *TTContractHandler {
	return &TTContractHandler{svc: svc}
}

type createTTContractRequest struct {
	ContractNumber     string             `json:"contract_number"`
	PatentID           *int64             `json:"patent_id,omitempty"`
	PoolID             *int64             `json:"pool_id,omitempty"`
	Licensor           string             `json:"licensor"`
	Licensee           string             `json:"licensee"`
	LicenseeCNPJ       string             `json:"licensee_cnpj"`
	LicenseKind        domain.LicenseKind `json:"license_kind"`
	Sublicensable      bool               `json:"sublicensable"`
	Territory          string             `json:"territory"`
	FieldOfUse         string             `json:"field_of_use"`
	RoyaltyRate        float64            `json:"royalty_rate"`
	RoyaltyFloorAnnual float64            `json:"royalty_floor_annual"`
	UpfrontFee         float64            `json:"upfront_fee"`
	InventorSharePct   int                `json:"inventor_share_pct"`
	Milestones         json.RawMessage    `json:"milestones,omitempty"`
	Notes              string             `json:"notes"`
}

// Create — POST /api/v1/tt-contracts
func (h *TTContractHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req createTTContractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json", "message": err.Error()})
		return
	}

	c := &domain.TTContract{
		ContractNumber:     req.ContractNumber,
		PatentID:           req.PatentID,
		PoolID:             req.PoolID,
		Licensor:           req.Licensor,
		Licensee:           req.Licensee,
		LicenseeCNPJ:       req.LicenseeCNPJ,
		LicenseKind:        req.LicenseKind,
		Sublicensable:      req.Sublicensable,
		Territory:          req.Territory,
		FieldOfUse:         req.FieldOfUse,
		RoyaltyRate:        req.RoyaltyRate,
		RoyaltyFloorAnnual: req.RoyaltyFloorAnnual,
		UpfrontFee:         req.UpfrontFee,
		InventorSharePct:   req.InventorSharePct,
		Milestones:         req.Milestones,
		AuditRights:        true,
		Notes:              req.Notes,
	}
	saved, err := h.svc.Create(r.Context(), c)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, saved)
}

// List — GET /api/v1/tt-contracts
func (h *TTContractHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := domain.TTContractFilter{
		Status: domain.ContractStatus(q.Get("status")),
		Search: q.Get("q"),
	}
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			f.Limit = v
		}
	}
	if p := q.Get("patent_id"); p != "" {
		if v, err := strconv.ParseInt(p, 10, 64); err == nil {
			f.PatentID = &v
		}
	}
	if p := q.Get("pool_id"); p != "" {
		if v, err := strconv.ParseInt(p, 10, 64); err == nil {
			f.PoolID = &v
		}
	}

	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	if items == nil {
		items = []domain.TTContract{}
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"items":      items,
		"pagination": map[string]any{"total": total, "limit": f.Limit, "offset": f.Offset},
	})
}

// GetByID — GET /api/v1/tt-contracts/{id}
func (h *TTContractHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	c, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, c)
}

// UpdateStatus — PATCH /api/v1/tt-contracts/{id}/status
func (h *TTContractHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	var body struct {
		Status domain.ContractStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	if err := h.svc.UpdateStatus(r.Context(), id, body.Status); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// Delete — DELETE /api/v1/tt-contracts/{id}
func (h *TTContractHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
