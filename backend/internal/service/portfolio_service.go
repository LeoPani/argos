// Package service — PortfolioService aggregates patents and trademarks
// into a unified portfolio view with cost projections.
//
// INPI annuity schedule (simplified brackets, 2024 table):
//
//	Year 3-4:   R$ 230/year
//	Year 5-6:   R$ 350/year
//	Year 7-8:   R$ 550/year
//	Year 9-10:  R$ 800/year
//	Year 11-13: R$ 1200/year
//	Year 14-16: R$ 1800/year
//	Year 17-20: R$ 2500/year
//
// Trademark renewal: R$ 855/class every 10 years ≈ R$ 85.50/year per class.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/repository"
)

// ─── Response types ───────────────────────────────────────────────────────────

// PortfolioAsset is a unified view of a patent or trademark.
type PortfolioAsset struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`          // PI | MU | TM | DP
	Number      string  `json:"number"`
	Title       string  `json:"title"`
	Owner       string  `json:"owner"`
	Status      string  `json:"status"`        // active | pending | expired | opposition
	FilingDate  string  `json:"filing_date"`
	ExpiryDate  string  `json:"expiry_date"`
	NextFeeDate *string `json:"next_fee_date"` // null if expired / no upcoming fee
	CostAnnual  float64 `json:"cost_annual"`
	CostMonthly float64 `json:"cost_monthly"`
	CostTotal   float64 `json:"cost_total"`
	IPCCode     string  `json:"ipc_code,omitempty"`
}

// PortfolioCostSummary aggregates costs across all assets.
type PortfolioCostSummary struct {
	Monthly float64 `json:"monthly"`
	Annual  float64 `json:"annual"`
	Total   float64 `json:"total"`
}

// PortfolioCostPoint is a single entry in the 5-year cost timeline.
type PortfolioCostPoint struct {
	Year  string  `json:"year"`
	Value float64 `json:"value"`
}

// PortfolioAIOpportunity wraps a UFOP opportunity as a portfolio suggestion.
type PortfolioAIOpportunity struct {
	ID            string  `json:"id"`
	Type          string  `json:"type"` // "opportunity" | "risk"
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	IPCClass      string  `json:"ipc_class,omitempty"`
	EstimatedCost float64 `json:"estimated_cost,omitempty"`
	Confidence    int     `json:"confidence"`
	ActionLabel   string  `json:"action_label"`
}

// PortfolioResponse is the full payload returned by GET /api/v1/portfolio.
type PortfolioResponse struct {
	Assets          []PortfolioAsset         `json:"assets"`
	CostSummary     PortfolioCostSummary     `json:"cost_summary"`
	CostTimeline    []PortfolioCostPoint     `json:"cost_timeline"`
	AIOpportunities []PortfolioAIOpportunity `json:"ai_opportunities"`
}

// ─── Service ──────────────────────────────────────────────────────────────────

// PortfolioService assembles the portfolio view from patents, trademarks,
// and UFOP opportunities.
type PortfolioService struct {
	patents    repository.PatentRepository
	trademarks repository.TrademarkRepository
	ufop       repository.UFOPRepository
}

// NewPortfolioService creates the service.
func NewPortfolioService(
	patents repository.PatentRepository,
	trademarks repository.TrademarkRepository,
	ufop repository.UFOPRepository,
) *PortfolioService {
	return &PortfolioService{patents: patents, trademarks: trademarks, ufop: ufop}
}

// Get returns the full portfolio view.
func (s *PortfolioService) Get(ctx context.Context) (*PortfolioResponse, error) {
	now := time.Now()
	var assets []PortfolioAsset

	// ── Patents ───────────────────────────────────────────────────────────
	patents, err := s.patents.List(ctx, domain.PatentFilter{Limit: 200})
	if err != nil {
		return nil, fmt.Errorf("portfolio: list patents: %w", err)
	}
	for _, p := range patents {
		assets = append(assets, patentToAsset(p, now))
	}

	// ── Trademarks ────────────────────────────────────────────────────────
	trademarks, err := s.trademarks.List(ctx, domain.TrademarkFilter{Limit: 200})
	if err != nil {
		return nil, fmt.Errorf("portfolio: list trademarks: %w", err)
	}
	for _, t := range trademarks {
		assets = append(assets, trademarkToAsset(t, now))
	}

	if assets == nil {
		assets = []PortfolioAsset{}
	}

	// ── Cost summary ──────────────────────────────────────────────────────
	summary := computeSummary(assets)

	// ── Cost timeline (5 years from now) ──────────────────────────────────
	timeline := computeTimeline(assets, now)

	// ── AI opportunities from UFOP ────────────────────────────────────────
	opps, err := s.ufop.List(ctx, domain.UFOPFilter{
		Level:  domain.UFOPLevelHigh,
		Status: domain.UFOPStatusNew,
		Limit:  5,
	})
	if err != nil {
		opps = nil // non-fatal
	}
	aiOpps := ufopToAIOpportunities(opps)

	return &PortfolioResponse{
		Assets:          assets,
		CostSummary:     summary,
		CostTimeline:    timeline,
		AIOpportunities: aiOpps,
	}, nil
}

// ─── Patent → PortfolioAsset ──────────────────────────────────────────────────

func patentToAsset(p domain.Patent, now time.Time) PortfolioAsset {
	filing := ""
	expiry := ""
	var nextFee *string

	if p.FilingDate != nil {
		filing = p.FilingDate.Format("2006-01-02")
		exp := p.FilingDate.AddDate(20, 0, 0)
		expiry = exp.Format("2006-01-02")

		nf := nextAnnuityDate(*p.FilingDate, now)
		if nf != nil {
			s := nf.Format("2006-01-02")
			nextFee = &s
		}
	}

	ageYears := 0
	if p.FilingDate != nil {
		ageYears = int(now.Sub(*p.FilingDate).Hours() / (24 * 365.25))
	}
	annual := patentAnnuity(ageYears + 1) // next year's fee
	remaining := 0
	if p.FilingDate != nil {
		exp := p.FilingDate.AddDate(20, 0, 0)
		remaining = int(exp.Sub(now).Hours() / (24 * 365.25))
		if remaining < 0 {
			remaining = 0
		}
	}
	total := accumulatedCost(ageYears, remaining)

	status := patentStatus(p)

	// Prefer the raw IPC code from INPI; fall back to the AI category letter.
	ipcCode := p.IPCCode
	if ipcCode == "" && p.IPCCategory.IsValid() {
		ipcCode = ipcCategoryLetter(p.IPCCategory)
	}

	return PortfolioAsset{
		ID:          fmt.Sprintf("pat-%d", p.ID),
		Type:        "PI",
		Number:      p.ApplicationNumber,
		Title:       p.Title,
		Owner:       p.Applicant,
		Status:      status,
		FilingDate:  filing,
		ExpiryDate:  expiry,
		NextFeeDate: nextFee,
		CostAnnual:  annual,
		CostMonthly: round2(annual / 12),
		CostTotal:   total,
		IPCCode:     ipcCode,
	}
}

func patentStatus(p domain.Patent) string {
	switch p.Status {
	case "classified":
		return "active"
	case "pending", "failed":
		return "pending"
	default:
		return "pending"
	}
}

// patentAnnuity returns the INPI annuity for a patent in a given year of life.
func patentAnnuity(yearOfLife int) float64 {
	switch {
	case yearOfLife <= 2:
		return 0
	case yearOfLife <= 4:
		return 230
	case yearOfLife <= 6:
		return 350
	case yearOfLife <= 8:
		return 550
	case yearOfLife <= 10:
		return 800
	case yearOfLife <= 13:
		return 1200
	case yearOfLife <= 16:
		return 1800
	default:
		return 2500
	}
}

// accumulatedCost estimates total remaining cost from current age.
func accumulatedCost(currentAge, remainingYears int) float64 {
	total := 0.0
	for i := 0; i < remainingYears; i++ {
		total += patentAnnuity(currentAge + i + 1)
	}
	return total
}

// nextAnnuityDate returns the date of the next annuity payment due.
// Annuities start in year 3 and are due on the filing date anniversary.
func nextAnnuityDate(filing, now time.Time) *time.Time {
	for y := 3; y <= 20; y++ {
		due := filing.AddDate(y, 0, 0)
		if due.After(now) {
			return &due
		}
	}
	return nil
}

// ─── Trademark → PortfolioAsset ───────────────────────────────────────────────

func trademarkToAsset(t domain.Trademark, now time.Time) PortfolioAsset {
	filing := ""
	if t.FilingDate != nil {
		filing = t.FilingDate.Format("2006-01-02")
	}

	// Expiry = granted date + 10 years; fall back to filing date + 10 years.
	var expiryTime *time.Time
	base := t.GrantedDate
	if base == nil {
		base = t.FilingDate
	}
	if base != nil {
		exp := base.AddDate(10, 0, 0)
		expiryTime = &exp
	}

	expiry := ""
	var nextFee *string
	remainingYears := 0

	if expiryTime != nil {
		expiry = expiryTime.Format("2006-01-02")
		// Next renewal due: roll forward in 10-year steps until future.
		next := *expiryTime
		for !next.After(now) {
			next = next.AddDate(10, 0, 0)
		}
		s := next.Format("2006-01-02")
		nextFee = &s

		rem := expiryTime.Sub(now).Hours() / (24 * 365.25)
		if rem > 0 {
			remainingYears = int(rem)
		}
	}

	nClasses := len(t.NiceClasses)
	if nClasses == 0 {
		nClasses = 1
	}
	annualCost := 85.50 * float64(nClasses) // R$ 855/class every 10 yr ≈ R$ 85.50/yr

	return PortfolioAsset{
		ID:          fmt.Sprintf("tm-%d", t.ID),
		Type:        "TM",
		Number:      t.ProcessNumber,
		Title:       t.Name,
		Owner:       t.Owner,
		Status:      trademarkStatus(t),
		FilingDate:  filing,
		ExpiryDate:  expiry,
		NextFeeDate: nextFee,
		CostAnnual:  annualCost,
		CostMonthly: round2(annualCost / 12),
		CostTotal:   round2(annualCost * float64(remainingYears)),
	}
}

func trademarkStatus(t domain.Trademark) string {
	switch t.Status {
	case domain.TrademarkStatusGranted:
		return "active"
	case domain.TrademarkStatusFiled:
		return "pending"
	case domain.TrademarkStatusPublished:
		return "opposition" // under opposition window
	case domain.TrademarkStatusExpired, domain.TrademarkStatusDenied, domain.TrademarkStatusArchived:
		return "expired"
	default:
		return "pending"
	}
}

// ─── Cost aggregation ────────────────────────────────────────────────────────

func computeSummary(assets []PortfolioAsset) PortfolioCostSummary {
	var monthly, annual, total float64
	for _, a := range assets {
		monthly += a.CostMonthly
		annual += a.CostAnnual
		total += a.CostTotal
	}
	return PortfolioCostSummary{
		Monthly: round2(monthly),
		Annual:  round2(annual),
		Total:   round2(total),
	}
}

// computeTimeline projects annual costs for the next 5 years.
func computeTimeline(assets []PortfolioAsset, now time.Time) []PortfolioCostPoint {
	points := make([]PortfolioCostPoint, 5)
	for i := range points {
		yr := now.Year() + i
		points[i].Year = fmt.Sprintf("%d", yr)

		for _, a := range assets {
			if a.Type == "PI" {
				filing, err := time.Parse("2006-01-02", a.FilingDate)
				if err != nil {
					continue
				}
				age := yr - filing.Year()
				points[i].Value += patentAnnuity(age)
			} else {
				points[i].Value += a.CostAnnual
			}
		}
		points[i].Value = round2(points[i].Value)
	}
	return points
}

// ─── UFOP → AI opportunities ──────────────────────────────────────────────────

func ufopToAIOpportunities(opps []domain.UFOPOpportunity) []PortfolioAIOpportunity {
	var result []PortfolioAIOpportunity
	for _, o := range opps {
		confidence := int(o.PIScore * 10)
		if confidence > 99 {
			confidence = 99
		}

		ipcClass := ""
		if o.IPCSuggestion != "" && len(o.IPCSuggestion) > 0 {
			ipcClass = string(o.IPCSuggestion[0]) // "C — Química..." → "C"
		}

		desc := o.AIAnalysis
		if desc == "" {
			desc = o.Abstract
		}
		if len(desc) > 200 {
			desc = desc[:197] + "…"
		}

		result = append(result, PortfolioAIOpportunity{
			ID:            fmt.Sprintf("opp-%d", o.ID),
			Type:          "opportunity",
			Title:         truncateStr(o.Title, 80),
			Description:   desc,
			IPCClass:      ipcClass,
			EstimatedCost: 3500, // INPI filing fee estimate (PI + taxes)
			Confidence:    confidence,
			ActionLabel:   "Iniciar consulta",
		})
	}
	if result == nil {
		result = []PortfolioAIOpportunity{}
	}
	return result
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// ipcCategoryLetter maps a BERT category 0-7 to the IPC section letter.
func ipcCategoryLetter(c domain.IPCCategory) string {
	letters := [8]string{"A", "B", "C", "D", "E", "F", "G", "H"}
	i := int(c)
	if i < 0 || i >= len(letters) {
		return ""
	}
	return letters[i]
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
