// Package errs defines common error variables used across the plugin.
package errs

import (
	"errors"
)

// ErrUnknownSourceType is returned when a string does not name a known source type.
var ErrUnknownSourceType = errors.New("unknown source type")
