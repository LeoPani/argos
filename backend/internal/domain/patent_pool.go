package domain

import "time"

// PoolKind tipifica o pool.
type PoolKind string

const (
	PoolKindOffensive          PoolKind = "offensive"           // licenciamento conjunto a terceiros
	PoolKindDefensive          PoolKind = "defensive"           // proteção mútua entre titulares
	PoolKindStandardEssential  PoolKind = "standard_essential"  // SEP / FRAND
)

// PoolStatus rastreia o ciclo do pool.
type PoolStatus string

const (
	PoolStatusForming PoolStatus = "forming"
	PoolStatusActive  PoolStatus = "active"
	PoolStatusClosed  PoolStatus = "closed"
)

// PoolMember é uma patente no pool com sua participação.
type PoolMember struct {
	ID        int64     `json:"id"`
	PoolID    int64     `json:"pool_id"`
	PatentID  int64     `json:"patent_id"`
	SharePct  float64   `json:"share_pct"`
	AddedAt   time.Time `json:"added_at"`

	// Snapshot hydratado pelo service (sem persistência).
	PatentNumber string `json:"patent_number,omitempty" db:"-"`
	PatentTitle  string `json:"patent_title,omitempty"  db:"-"`
}

// PatentPool é o agregado.
type PatentPool struct {
	ID            int64        `json:"id"`
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	Kind          PoolKind     `json:"pool_kind"`
	RoyaltyRate   float64      `json:"royalty_rate"`
	Territory     string       `json:"territory"`
	DurationYears int          `json:"duration_years"`
	Administrator string       `json:"administrator"`
	Status        PoolStatus   `json:"status"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`

	Members []PoolMember `json:"members,omitempty"`
}

// Validate enforces invariants for new pools.
func (p *PatentPool) Validate() error {
	if p.Name == "" {
		return ErrInvalidArg
	}
	switch p.Kind {
	case PoolKindOffensive, PoolKindDefensive, PoolKindStandardEssential:
	case "":
		p.Kind = PoolKindOffensive
	default:
		return ErrInvalidArg
	}
	if p.Status == "" {
		p.Status = PoolStatusForming
	}
	if p.Territory == "" {
		p.Territory = "BR"
	}
	if p.Administrator == "" {
		p.Administrator = "NIT-UFOP"
	}
	if p.DurationYears == 0 {
		p.DurationYears = 10
	}
	return nil
}
