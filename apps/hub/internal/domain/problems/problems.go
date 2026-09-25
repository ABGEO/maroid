// Package problems holds the problem types that only the hub answers. Each one
// carries a fact of a domain of the hub, so no plugin reaches for it and
// libs/rest does not hold it.
package problems

import (
	"net/http"

	"github.com/abgeo/maroid/libs/rest"
)

// The problem types that the hub owns.
const (
	TypeNetworkNotAllowed = "/problems/hub/network-not-allowed"
	TypeSettingsAbsent    = "/problems/hub/settings-absent"
	TypeIdentityLast      = "/problems/hub/identity-last"
	TypeSettingsInvalid   = "/problems/hub/settings-invalid"
)

// NewNetworkNotAllowed reports a caller that the network allowlist does not hold.
func NewNetworkNotAllowed() rest.Problem {
	return rest.Problem{
		Type:   TypeNetworkNotAllowed,
		Title:  "The caller is not on the network allowlist.",
		Status: http.StatusForbidden,
	}
}

// NewSettingsAbsent reports a plugin that declares no settings.
func NewSettingsAbsent() rest.Problem {
	return rest.Problem{
		Type:   TypeSettingsAbsent,
		Title:  "The plugin declares no settings.",
		Status: http.StatusNotFound,
	}
}

// NewIdentityLast reports a detach of the last external account of a record.
func NewIdentityLast() rest.Problem {
	return rest.Problem{
		Type:   TypeIdentityLast,
		Title:  "The last external account cannot be detached.",
		Status: http.StatusConflict,
	}
}

// NewSettingsInvalid reports settings that the schema of a plugin refuses.
func NewSettingsInvalid() rest.Problem {
	return rest.Problem{
		Type:   TypeSettingsInvalid,
		Title:  "The settings do not match the schema.",
		Status: http.StatusUnprocessableEntity,
	}
}
