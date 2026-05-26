package domain

import "time"

// IPTimestamp representa um registro de anterioridade com prova de existência
// baseada em cadeia de hashes SHA-256 (proof-of-existence sem blockchain real).
type IPTimestamp struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Authors     []string  `json:"authors"`
	Category    string    `json:"category"` // "invenção" | "software" | "design" | "segredo industrial"
	ContentHash string    `json:"content_hash"` // SHA-256 canônico
	PrevHash    string    `json:"prev_hash"`    // hash do registro anterior (cadeia)
	ChainIndex  int64     `json:"chain_index"`  // posição na cadeia
	CreatedAt   time.Time `json:"created_at"`
}

// IPTimestampFilter para listagem paginada.
type IPTimestampFilter struct {
	Limit  int
	Offset int
}
