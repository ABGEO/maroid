// Package problems holds the problem types that only the hub answers. Each one
// carries a fact of a domain of the hub, so no plugin reaches for it and
// libs/rest does not hold it.
package problems

import (
	"net/http"

	"github.com/abgeo/maroid/libs/rest/problem"
)

// The problem types that the hub owns.
const (
	TypeNetworkNotAllowed = "/problems/hub/network-not-allowed"
	TypeSettingsAbsent    = "/problems/hub/settings-absent"
	TypeIdentityLast      = "/problems/hub/identity-last"
	TypeSettingsInvalid   = "/problems/hub/settings-invalid"
	TypeNotReady          = "/problems/hub/not-ready"
	TypeMemberExists      = "/problems/hub/member-exists"
	TypeManagerLast       = "/problems/hub/manager-last"
	TypeAdministratorLast = "/problems/hub/administrator-last"
)

// NotReadyProblem reports a hub that cannot serve a request, with the name of each
// dependency that failed.
type NotReadyProblem struct {
	problem.Problem

	Dependencies []string `json:"dependencies,omitempty"`
}

// NewNetworkNotAllowed reports a caller that the network allowlist does not hold.
func NewNetworkNotAllowed() *problem.Problem {
	return &problem.Problem{
		Type:   TypeNetworkNotAllowed,
		Title:  "The caller is not on the network allowlist.",
		Status: http.StatusForbidden,
	}
}

// NewSettingsAbsent reports a plugin that declares no settings.
func NewSettingsAbsent() *problem.Problem {
	return &problem.Problem{
		Type:   TypeSettingsAbsent,
		Title:  "The plugin declares no settings.",
		Status: http.StatusNotFound,
	}
}

// NewIdentityLast reports a detach of the last external account of a record.
func NewIdentityLast() *problem.Problem {
	return &problem.Problem{
		Type:   TypeIdentityLast,
		Title:  "The last external account cannot be detached.",
		Status: http.StatusConflict,
	}
}

// NewMemberExists reports an addition of a user record that is already a member
// of the workspace.
func NewMemberExists() *problem.Problem {
	return &problem.Problem{
		Type:   TypeMemberExists,
		Title:  "The user record is already a member of the workspace.",
		Status: http.StatusConflict,
	}
}

// NewManagerLast reports a change that leaves a workspace with no manager.
func NewManagerLast() *problem.Problem {
	return &problem.Problem{
		Type:   TypeManagerLast,
		Title:  "The change leaves the workspace with no manager.",
		Status: http.StatusConflict,
	}
}

// NewAdministratorLast reports a change that leaves the instance with no active
// administrator.
func NewAdministratorLast() *problem.Problem {
	return &problem.Problem{
		Type:   TypeAdministratorLast,
		Title:  "The change leaves the instance with no active administrator.",
		Status: http.StatusConflict,
	}
}

// NewSettingsInvalid reports settings that the schema of a plugin refuses, with
// one item for each field that failed.
func NewSettingsInvalid(failures ...problem.FieldFailure) *problem.ValidationProblem {
	return &problem.ValidationProblem{
		Problem: problem.Problem{
			Type:   TypeSettingsInvalid,
			Title:  "The settings do not match the schema.",
			Status: http.StatusUnprocessableEntity,
		},
		Errors: failures,
	}
}

// NewNotReady reports a hub that cannot serve a request. A shutdown names no
// dependency, because none failed.
func NewNotReady(dependencies ...string) *NotReadyProblem {
	return &NotReadyProblem{
		Problem: problem.Problem{
			Type:   TypeNotReady,
			Title:  "The hub cannot serve a request.",
			Status: http.StatusServiceUnavailable,
		},
		Dependencies: dependencies,
	}
}
