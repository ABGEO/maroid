// Package errs defines common error variables used across the plugin.
package errs

import (
	"errors"
)

// ErrRequestFailed is returned when an HTTP or API request fails.
var ErrRequestFailed = errors.New("request failed")
