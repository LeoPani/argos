package domain

import "time"

// WatchType identifies what kind of entity an alert monitors.
type WatchType string

const (
	WatchTypeTerm    WatchType = "term"    // free-text search
	WatchTypeBrand   WatchType = "brand"   // trademark name
	WatchTypeCompany WatchType = "company" // applicant / holder name
	WatchTypePatent  WatchType = "patent"  // specific patent number
)

// WatchStatus reflects whether the watch has new matches awaiting review.
type WatchStatus string

const (
	WatchStatusOK    WatchStatus = "ok"
	WatchStatusAlert WatchStatus = "alert"
)

// Watchlist is a saved monitoring subscription.
type Watchlist struct {
	ID        int64       `json:"id"`
	Label     string      `json:"label"`
	Type      WatchType   `json:"watch_type"`
	Query     string      `json:"query"`
	LastCheck *time.Time  `json:"last_check"`
	NewCount  int         `json:"new_count"`
	Status    WatchStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// Validate enforces invariants before insertion.
func (w *Watchlist) Validate() error {
	switch {
	case w.Label == "":
		return ErrInvalidArg
	case w.Type == "":
		return ErrInvalidArg
	}
	switch w.Type {
	case WatchTypeTerm, WatchTypeBrand, WatchTypeCompany, WatchTypePatent:
	default:
		return ErrInvalidArg
	}
	// Query defaults to label when caller didn't set one explicitly.
	if w.Query == "" {
		w.Query = w.Label
	}
	if w.Status == "" {
		w.Status = WatchStatusOK
	}
	return nil
}
