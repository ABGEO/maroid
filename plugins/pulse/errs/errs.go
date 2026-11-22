// Package errs defines the errors that the packages of pulse share.
package errs

import "errors"

// ErrPromptAnswered is returned when a value names a prompt that already holds one.
var ErrPromptAnswered = errors.New("value: the prompt already holds a value")
