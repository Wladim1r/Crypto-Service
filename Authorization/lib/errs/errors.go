// Package errs
package errs

import "errors"

var (
	ErrDB           = errors.New("database error")
	ErrRecordingWNC = errors.New("recording wasn't created")
	ErrRecordingWND = errors.New("recording wasn't deleted")
	ErrRecordingWNF = errors.New("recording wasn't found")
	ErrTokenTTL     = errors.New("token's ttl is over")
)
