package domain

import (
	"encoding/json"
	"time"
)

// LicenseKind tipifica a licença.
type LicenseKind string

const (
	LicenseExclusive    LicenseKind = "exclusive"
	LicenseNonExclusive LicenseKind = "non_exclusive"
	LicenseSole         LicenseKind = "sole"
)

// ContractStatus rastreia o ciclo do contrato.
type ContractStatus string

const (
	ContractDraft       ContractStatus = "draft"
	ContractNegotiating ContractStatus = "negotiating"
	ContractActive      ContractStatus = "active"
	ContractExpired     ContractStatus = "expired"
	ContractTerminated  ContractStatus = "terminated"
)

// Milestone é um gatilho de pagamento ou entrega.
type Milestone struct {
	Label    string  `json:"label"`
	DueDate  string  `json:"due_date,omitempty"` // ISO-8601 date
	FeeBRL   float64 `json:"fee_brl,omitempty"`
	Done     bool    `json:"done"`
}

// TTContract é um contrato de transferência tecnológica.
type TTContract struct {
	ID              int64           `json:"id"`
	ContractNumber  string          `json:"contract_number"`
	PatentID        *int64          `json:"patent_id,omitempty"`
	PoolID          *int64          `json:"pool_id,omitempty"`

	Licensor        string          `json:"licensor"`
	Licensee        string          `json:"licensee"`
	LicenseeCNPJ    string          `json:"licensee_cnpj"`

	LicenseKind     LicenseKind     `json:"license_kind"`
	Sublicensable   bool            `json:"sublicensable"`
	Territory       string          `json:"territory"`
	FieldOfUse      string          `json:"field_of_use"`

	RoyaltyRate        float64      `json:"royalty_rate"`         // %
	RoyaltyFloorAnnual float64      `json:"royalty_floor_annual"` // BRL
	UpfrontFee         float64      `json:"upfront_fee"`          // BRL
	InventorSharePct   int          `json:"inventor_share_pct"`   // 0..50

	Milestones      json.RawMessage `json:"milestones"`           // []Milestone serializado

	SignedAt        *time.Time      `json:"signed_at,omitempty"`
	ExpiresAt       *time.Time      `json:"expires_at,omitempty"`

	Status          ContractStatus  `json:"status"`
	NITApproved     bool            `json:"nit_approved"`
	AuditRights     bool            `json:"audit_rights"`

	Notes           string          `json:"notes"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// Validate enforces invariants for new contracts.
func (c *TTContract) Validate() error {
	switch {
	case c.ContractNumber == "":
		return ErrInvalidArg
	case c.Licensee == "":
		return ErrInvalidArg
	case c.PatentID == nil && c.PoolID == nil:
		return ErrInvalidArg
	}
	switch c.LicenseKind {
	case LicenseExclusive, LicenseNonExclusive, LicenseSole:
	default:
		return ErrInvalidArg
	}
	if c.InventorSharePct < 0 || c.InventorSharePct > 50 {
		return ErrInvalidArg
	}
	if c.Status == "" {
		c.Status = ContractDraft
	}
	if c.Licensor == "" {
		c.Licensor = "Universidade Federal de Ouro Preto"
	}
	if c.Territory == "" {
		c.Territory = "BR"
	}
	if c.InventorSharePct == 0 {
		c.InventorSharePct = 33
	}
	if len(c.Milestones) == 0 {
		c.Milestones = []byte("[]")
	}
	return nil
}

// TTContractFilter é o filtro de listagem.
type TTContractFilter struct {
	Status    ContractStatus
	PatentID  *int64
	PoolID    *int64
	Search    string
	Limit     int
	Offset    int
}

func (f *TTContractFilter) Normalize() {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 200 {
		f.Limit = 200
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
}
