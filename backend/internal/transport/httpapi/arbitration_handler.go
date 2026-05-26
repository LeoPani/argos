package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/pkg/httputil"
)

// ArbitrationHandler exposes subject + verdict endpoints.
type ArbitrationHandler struct{ svc *service.ArbitrationAI }

func NewArbitrationHandler(svc *service.ArbitrationAI) *ArbitrationHandler {
	return &ArbitrationHandler{svc: svc}
}

type addSubjectRequest struct {
	Kind    domain.SubjectKind `json:"kind"`
	RefID   *int64             `json:"ref_id,omitempty"`
	Label   string             `json:"label"`
	PartyID *int64             `json:"party_id,omitempty"`
}

// AddSubject — POST /api/v1/disputes/{id}/subjects
func (h *ArbitrationHandler) AddSubject(w http.ResponseWriter, r *http.Request) {
	did, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}

	var req addSubjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json", "message": err.Error()})
		return
	}

	s := &domain.DisputeSubject{
		DisputeID: did,
		Kind:      req.Kind,
		RefID:     req.RefID,
		Label:     req.Label,
		PartyID:   req.PartyID,
	}
	if err := h.svc.AddSubject(r.Context(), s); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, s)
}

// ListSubjects — GET /api/v1/disputes/{id}/subjects
func (h *ArbitrationHandler) ListSubjects(w http.ResponseWriter, r *http.Request) {
	did, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}

	subjects, err := h.svc.ListSubjects(r.Context(), did)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	if subjects == nil {
		subjects = []domain.DisputeSubject{}
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"items": subjects, "count": len(subjects)})
}

// DeleteSubject — DELETE /api/v1/disputes/subjects/{subjectId}
func (h *ArbitrationHandler) DeleteSubject(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("subjectId"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}
	if err := h.svc.DeleteSubject(r.Context(), id); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Analyze — POST /api/v1/disputes/{id}/analyze
func (h *ArbitrationHandler) Analyze(w http.ResponseWriter, r *http.Request) {
	did, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}

	verdict, err := h.svc.Analyze(r.Context(), did)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "analyze_failed", "message": err.Error()})
		return
	}
	httputil.JSON(w, http.StatusOK, verdict)
}

// ComparePatents — POST /api/v1/disputes/compare
// Lightweight quick-compare — no dispute required.
func (h *ArbitrationHandler) ComparePatents(w http.ResponseWriter, r *http.Request) {
	var req service.PIComparisonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json", "message": err.Error()})
		return
	}
	if req.PatentAID == 0 || req.PatentBID == 0 {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "patent_a_id and patent_b_id required"})
		return
	}

	result, err := h.svc.ComparePatents(r.Context(), req.PatentAID, req.PatentBID)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "compare_failed", "message": err.Error()})
		return
	}
	httputil.JSON(w, http.StatusOK, result)
}

// LatestVerdict — GET /api/v1/disputes/{id}/verdict
func (h *ArbitrationHandler) LatestVerdict(w http.ResponseWriter, r *http.Request) {
	did, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_id"})
		return
	}

	v, err := h.svc.LatestVerdict(r.Context(), did)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			httputil.JSON(w, http.StatusOK, map[string]any{"verdict": nil})
			return
		}
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"verdict": v})
}
