package apperr

import (
	"errors"
	"time"
)

var (
	ErrInvalid        = errors.New("invalid input")
	ErrNotFound       = errors.New("not found")
	ErrConflict       = errors.New("conflict")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")
	ErrTooEarly       = errors.New("too early")
	ErrNotImplemented = errors.New("not implemented yet")
)

type RetryAfter interface {
	RetryAfter() time.Duration
}
