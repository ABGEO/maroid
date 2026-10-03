package pluginapi_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// PERMS-SC-002: A role holds each lowest role at or below its own place in the order
// manager, editor, viewer.
func TestARoleHoldsEveryLowestRoleBelowIt(t *testing.T) {
	t.Parallel()

	roles := []pluginapi.Role{pluginapi.RoleManager, pluginapi.RoleEditor, pluginapi.RoleViewer}
	holds := map[pluginapi.Role][]pluginapi.Role{
		pluginapi.RoleManager: {pluginapi.RoleManager, pluginapi.RoleEditor, pluginapi.RoleViewer},
		pluginapi.RoleEditor:  {pluginapi.RoleEditor, pluginapi.RoleViewer},
		pluginapi.RoleViewer:  {pluginapi.RoleViewer},
	}

	for _, role := range roles {
		for _, lowest := range roles {
			assert.Equal(t, slices.Contains(holds[role], lowest), role.Holds(lowest),
				"%s against %s", role, lowest)
		}
	}
}

// PERMS-FR-002: A role that is not one of the three holds nothing, and reaches by
// nothing.
func TestAnUnknownRoleHoldsNothing(t *testing.T) {
	t.Parallel()

	owner := pluginapi.Role("owner")

	assert.False(t, owner.Holds(pluginapi.RoleViewer))
	assert.False(t, pluginapi.RoleManager.Holds(owner))
}
