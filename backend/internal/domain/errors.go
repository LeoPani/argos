// Package domain holds the core business entities and rules for Argos.
// This file collects sentinel errors that adapters wrap so service-layer
// and transport-layer code can branch on outcome without importing any
// driver-specific package.
package domain

import "errors"

// Sentinel errors. Compare with errors.Is, never with ==.
//
// Typical usage at the repository boundary:
//
//	if pqErr.Code == "23505" {
//	    return fmt.Errorf("insert patent %q: %w", id, domain.ErrDuplicate)
//	}
//
// And at the HTTP handler boundary:
//
//	switch {
//	case errors.Is(err, domain.ErrNotFound):    w.WriteHeader(404)
//	case errors.Is(err, domain.ErrDuplicate):   w.WriteHeader(409)
//	case errors.Is(err, domain.ErrInvalidArg):  w.WriteHeader(400)
//	default:                                    w.WriteHeader(500)
//	}
var (
	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("domain: resource not found")

	// ErrDuplicate is returned when uniqueness is violated (e.g. inserting
	// a patent with an application number that already exists).
	ErrDuplicate = errors.New("domain: resource already exists")

	// ErrInvalidArg signals that a caller supplied a malformed value
	// (empty required field, value out of range, etc.).
	ErrInvalidArg = errors.New("domain: invalid argument")

	// ErrConflict is returned when an operation cannot proceed because
	// of the entity's current state (e.g. closing an already-closed dispute).
	ErrConflict = errors.New("domain: state conflict")

	// ErrUnauthorized is returned when the caller lacks permission to
	// perform the requested operation. Reserved for later phases.
	ErrUnauthorized = errors.New("domain: unauthorized")
)

// wrapInvalid is a tiny helper used by entity Validate methods to keep
// error wrapping consistent. Unexported on purpose.
func wrapInvalid(reason string) error {
	return &invalidArgError{reason: reason}
}

// invalidArgError implements error and wraps ErrInvalidArg so callers
// can use errors.Is(err, domain.ErrInvalidArg) AND still read the
// human-friendly reason via err.Error().
type invalidArgError struct {
	reason string
}

func (e *invalidArgError) Error() string { return "domain: invalid argument: " + e.reason }
func (e *invalidArgError) Unwrap() error { return ErrInvalidArg }
