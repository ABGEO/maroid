package authz_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// PERMS-FR-005: The hub declares its permissions, each at the lowest role that holds
// it, and the role of a member decides each one.
func TestTheRoleDecidesEachPermissionOfTheHub(t *testing.T) {
	t.Parallel()

	permissions, err := authz.NewPermissionRegistry()
	require.NoError(t, err)

	authorizer := authz.NewRoleAuthorizer(permissions)

	cases := []struct {
		role       pluginapi.Role
		permission string
		allowed    bool
		lowest     pluginapi.Role
	}{
		{pluginapi.RoleViewer, authz.PermissionWorkspaceRead, true, pluginapi.RoleViewer},
		{pluginapi.RoleViewer, authz.PermissionSettingsWrite, false, pluginapi.RoleEditor},
		{pluginapi.RoleEditor, authz.PermissionSettingsWrite, true, pluginapi.RoleEditor},
		{pluginapi.RoleEditor, authz.PermissionMembersWrite, false, pluginapi.RoleManager},
		{pluginapi.RoleViewer, authz.PermissionMembershipLeave, true, pluginapi.RoleViewer},
		{pluginapi.RoleManager, authz.PermissionWorkspaceWrite, true, pluginapi.RoleManager},
	}

	for _, tc := range cases {
		allowed, lowest, err := authorizer.Allowed(tc.role, tc.permission)

		require.NoError(t, err)
		assert.Equal(t, tc.allowed, allowed, "%s against %s", tc.role, tc.permission)
		assert.Equal(t, tc.lowest, lowest)
	}
}

// PERMS-FR-013: A permission that no one declares is a failure, not a refusal.
func TestAnUndeclaredPermissionFails(t *testing.T) {
	t.Parallel()

	permissions, err := authz.NewPermissionRegistry()
	require.NoError(t, err)

	allowed, _, err := authz.NewRoleAuthorizer(permissions).
		Allowed(pluginapi.RoleManager, "dev.maroid.probe:notes.delete")

	require.ErrorIs(t, err, errs.ErrPermissionUndeclared)
	assert.False(t, allowed)
}

// PERMS-FR-017: The permissions that a role holds are the ones at or below it, in one
// order.
func TestARoleHoldsThePermissionsAtOrBelowIt(t *testing.T) {
	t.Parallel()

	permissions, err := authz.NewPermissionRegistry()
	require.NoError(t, err)

	authorizer := authz.NewRoleAuthorizer(permissions)

	assert.Equal(t, []string{
		authz.PermissionMembershipLeave,
		authz.PermissionSettingsRead,
		authz.PermissionWorkspaceRead,
	}, authorizer.Held(pluginapi.RoleViewer))
	assert.Len(t, authorizer.Held(pluginapi.RoleManager), 6)
}
