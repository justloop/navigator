package navigator

import "errors"

var (
	// ErrNotReady indicates this navigator is not ready
	ErrNotReady = errors.New("Navigator is not ready yet")
)
