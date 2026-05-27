package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/LeoPani/argos/backend/internal/domain"
)

// ── Patent ────────────────────────────────────────────────────────────────────

func TestPatent_Validate_Valid(t *testing.T) {
	p := &domain.Patent{
		ApplicationNumber: "BR102024000001",
		Title:             "Sistema de purificação por ozônio",
		Abstract:          "Descreve um sistema portátil de purificação de água.",
	}
	if err := p.Validate(); err != nil {
		t.Errorf("valid patent should pass validation, got: %v", err)
	}
}

func TestPatent_Validate_MissingApplicationNumber(t *testing.T) {
	p := &domain.Patent{Title: "Test", Abstract: "Test abstract"}
	err := p.Validate()
	if err == nil {
		t.Fatal("expected error for missing application_number")
	}
	if !errors.Is(err, domain.ErrInvalidArg) {
		t.Errorf("expected ErrInvalidArg, got %v", err)
	}
}

func TestPatent_Validate_MissingTitle(t *testing.T) {
	p := &domain.Patent{ApplicationNumber: "BR1", Abstract: "Test abstract"}
	err := p.Validate()
	if err == nil {
		t.Fatal("expected error for missing title")
	}
	if !errors.Is(err, domain.ErrInvalidArg) {
		t.Errorf("expected ErrInvalidArg, got %v", err)
	}
}

func TestPatent_Validate_MissingAbstract(t *testing.T) {
	p := &domain.Patent{ApplicationNumber: "BR1", Title: "Test title"}
	err := p.Validate()
	if err == nil {
		t.Fatal("expected error for missing abstract")
	}
	if !errors.Is(err, domain.ErrInvalidArg) {
		t.Errorf("expected ErrInvalidArg, got %v", err)
	}
}

func TestPatent_Validate_ErrorMessage(t *testing.T) {
	p := &domain.Patent{ApplicationNumber: "BR1"}
	err := p.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

// ── IPCCategory ───────────────────────────────────────────────────────────────

func TestIPCCategory_IsValid(t *testing.T) {
	for cat := 0; cat <= 7; cat++ {
		c := domain.IPCCategory(cat)
		if !c.IsValid() {
			t.Errorf("IPCCategory(%d) should be valid", cat)
		}
	}
}

func TestIPCCategory_IsValid_Unknown(t *testing.T) {
	if domain.IPCCategoryUnknown.IsValid() {
		t.Error("IPCCategoryUnknown (-1) should not be valid")
	}
}

func TestIPCCategory_IsValid_OutOfRange(t *testing.T) {
	for _, c := range []int{-2, 8, 9, 100} {
		if domain.IPCCategory(c).IsValid() {
			t.Errorf("IPCCategory(%d) should not be valid", c)
		}
	}
}

func TestIPCCategoryUnknown_Value(t *testing.T) {
	if domain.IPCCategoryUnknown != -1 {
		t.Errorf("IPCCategoryUnknown should be -1, got %d", domain.IPCCategoryUnknown)
	}
}

// ── PatentStatus ──────────────────────────────────────────────────────────────

func TestPatentStatus_KnownValues(t *testing.T) {
	statuses := []domain.PatentStatus{
		domain.PatentStatusPending,
		domain.PatentStatusClassified,
		domain.PatentStatusFailed,
		domain.PatentStatusReclassified,
	}
	for _, s := range statuses {
		if string(s) == "" {
			t.Errorf("status %q should not be empty", s)
		}
	}
}

// ── Trademark ─────────────────────────────────────────────────────────────────

func TestTrademark_Validate_Valid(t *testing.T) {
	tm := &domain.Trademark{
		ProcessNumber: "BR502024000001",
		Name:          "ARGOS",
		Kind:          "word",
	}
	if err := tm.Validate(); err != nil {
		t.Errorf("valid trademark should pass: %v", err)
	}
}

func TestTrademark_Validate_MissingProcessNumber(t *testing.T) {
	tm := &domain.Trademark{Name: "ARGOS", Kind: "word"}
	err := tm.Validate()
	if !errors.Is(err, domain.ErrInvalidArg) {
		t.Errorf("expected ErrInvalidArg for missing process_number, got %v", err)
	}
}

func TestTrademark_Validate_MissingName(t *testing.T) {
	tm := &domain.Trademark{ProcessNumber: "BR1", Kind: "word"}
	err := tm.Validate()
	if !errors.Is(err, domain.ErrInvalidArg) {
		t.Errorf("expected ErrInvalidArg for missing name, got %v", err)
	}
}

func TestTrademark_Validate_MissingKind(t *testing.T) {
	tm := &domain.Trademark{ProcessNumber: "BR1", Name: "ARGOS"}
	err := tm.Validate()
	if !errors.Is(err, domain.ErrInvalidArg) {
		t.Errorf("expected ErrInvalidArg for missing kind, got %v", err)
	}
}

// ── Dispute ───────────────────────────────────────────────────────────────────

func TestDispute_Validate_Valid(t *testing.T) {
	d := &domain.Dispute{
		CaseNumber: "ARB-2024-001",
		Title:      "Disputa de anterioridade PI-ALFA vs PI-BETA",
		Kind:       "patent_conflict",
	}
	if err := d.Validate(); err != nil {
		t.Errorf("valid dispute should pass: %v", err)
	}
}

func TestDispute_Validate_MissingCaseNumber(t *testing.T) {
	d := &domain.Dispute{Title: "Test", Kind: "patent_conflict"}
	if !errors.Is(d.Validate(), domain.ErrInvalidArg) {
		t.Error("expected ErrInvalidArg for missing case_number")
	}
}

func TestDispute_Validate_MissingTitle(t *testing.T) {
	d := &domain.Dispute{CaseNumber: "ARB-001", Kind: "patent_conflict"}
	if !errors.Is(d.Validate(), domain.ErrInvalidArg) {
		t.Error("expected ErrInvalidArg for missing title")
	}
}

// ── Sentinel errors ───────────────────────────────────────────────────────────

func TestSentinelErrors_AreDistinct(t *testing.T) {
	sentinels := []error{
		domain.ErrNotFound,
		domain.ErrDuplicate,
		domain.ErrInvalidArg,
		domain.ErrConflict,
		domain.ErrUnauthorized,
	}
	for i, a := range sentinels {
		for j, b := range sentinels {
			if i != j && errors.Is(a, b) {
				t.Errorf("sentinel %v should not match %v", a, b)
			}
		}
	}
}

func TestSentinelErrors_ErrorMessagesNonEmpty(t *testing.T) {
	for _, err := range []error{
		domain.ErrNotFound,
		domain.ErrDuplicate,
		domain.ErrInvalidArg,
		domain.ErrConflict,
		domain.ErrUnauthorized,
	} {
		if err.Error() == "" {
			t.Errorf("sentinel error message should not be empty: %v", err)
		}
	}
}

func TestErrInvalidArg_Wrapping(t *testing.T) {
	// Validate returns an *invalidArgError that wraps ErrInvalidArg.
	// Callers must be able to use errors.Is to detect it.
	p := &domain.Patent{} // all required fields missing
	err := p.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, domain.ErrInvalidArg) {
		t.Errorf("errors.Is(err, ErrInvalidArg) should be true, got err=%v", err)
	}
	// Must NOT match other sentinels
	if errors.Is(err, domain.ErrNotFound) {
		t.Error("invalid arg error should not match ErrNotFound")
	}
}

// ── PatentFilter ──────────────────────────────────────────────────────────────

func TestPatentFilter_Normalize_DefaultLimit(t *testing.T) {
	f := domain.PatentFilter{}
	f.Normalize()
	if f.Limit <= 0 {
		t.Errorf("Normalize() should set a positive default limit, got %d", f.Limit)
	}
}

func TestPatentFilter_Normalize_ClampLimit(t *testing.T) {
	f := domain.PatentFilter{Limit: 99999}
	f.Normalize()
	const maxLimit = 500
	if f.Limit > maxLimit {
		t.Errorf("Normalize() should clamp limit to max %d, got %d", maxLimit, f.Limit)
	}
}

func TestPatentFilter_DateRange(t *testing.T) {
	from  := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	f := domain.PatentFilter{FilingFrom: &from, FilingUntil: &until}
	if f.FilingFrom == nil || f.FilingUntil == nil {
		t.Error("date range fields should be set")
	}
	if !f.FilingFrom.Before(*f.FilingUntil) {
		t.Error("FilingFrom should be before FilingUntil")
	}
}
