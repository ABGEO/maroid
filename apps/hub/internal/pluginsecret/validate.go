package pluginsecret

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	maxNameLength  = 64
	maxValueLength = 16384
)

func validateName(name string) error {
	if name == "" || len(name) > maxNameLength {
		return fmt.Errorf(
			"%w: the name holds %d characters, not 1 to %d",
			pluginapi.ErrSecretInvalid, len(name), maxNameLength,
		)
	}

	for _, char := range name {
		if !isNameCharacter(char) {
			return fmt.Errorf(
				"%w: the name %q holds %q, which is not a lowercase letter, a digit, '.', '_', or '-'",
				pluginapi.ErrSecretInvalid,
				name,
				char,
			)
		}
	}

	return nil
}

func isNameCharacter(char rune) bool {
	switch {
	case char >= 'a' && char <= 'z', char >= '0' && char <= '9':
		return true
	default:
		return char == '.' || char == '_' || char == '-'
	}
}

func validateValue(value string) error {
	length := utf8.RuneCountInString(value)
	if length == 0 || length > maxValueLength {
		return fmt.Errorf(
			"%w: the value holds %d characters, not 1 to %d",
			pluginapi.ErrSecretInvalid, length, maxValueLength,
		)
	}

	return nil
}

func validateExpiry(expiresAt time.Time, now time.Time) error {
	if !expiresAt.IsZero() && !expiresAt.After(now) {
		return fmt.Errorf("%w: the expiry is not in the future", pluginapi.ErrSecretInvalid)
	}

	return nil
}
