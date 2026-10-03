package model_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/abgeo/maroid/apps/hub/internal/model"
)

// EXTID-SC-025: The first workspace of a record takes its first name. WSPACE-DD-004
// cuts a long name so the whole name keeps to the limit, and names a record with no
// first name Workspace.
func TestTheFirstWorkspaceTakesTheFirstName(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"Nino":                  "Nino's Workspace",
		"  Nino  ":              "Nino's Workspace",
		"":                      "Workspace",
		"   ":                   "Workspace",
		strings.Repeat("ა", 70): strings.Repeat("ა", 52) + "'s Workspace",
	}

	for firstName, want := range cases {
		name := model.FirstWorkspaceName(firstName)

		assert.Equal(t, want, name, "first name %q", firstName)
		assert.LessOrEqual(t, len([]rune(name)), model.WorkspaceNameMaxLength)
	}
}
