package registrar_test

import (
	"context"
	"log/slog"
	"net/http"
	"testing"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/apps/hub/internal/plugin/registrar"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	notesRead      = "notes.read"
	notesWrite     = "notes.write"
	probeNotesRead = "dev.maroid.probe:notes.read"
)

// probe is a plugin that declares the permissions, routes, commands, and tools that
// the test gives it.
type probe struct {
	id          string
	permissions []pluginapi.Permission
	routes      []pluginapi.Route
	commands    []pluginapi.TelegramCommand
	tools       []pluginapi.MCPTool
}

var (
	_ pluginapi.PermissionPlugin      = (*probe)(nil)
	_ pluginapi.RoutePlugin           = (*probe)(nil)
	_ pluginapi.TelegramCommandPlugin = (*probe)(nil)
	_ pluginapi.MCPToolPlugin         = (*probe)(nil)
)

func (p *probe) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:         pluginapi.ParsePluginID(p.id),
		Version:    pluginVersion,
		APIVersion: pluginapi.APIVersion,
	}
}

func (p *probe) Permissions() ([]pluginapi.Permission, error) { return p.permissions, nil }

func (p *probe) Routes() ([]pluginapi.Route, error) { return p.routes, nil }

func (p *probe) TelegramCommands() ([]pluginapi.TelegramCommand, error) { return p.commands, nil }

func (p *probe) MCPTools() ([]pluginapi.MCPTool, error) { return p.tools, nil }

// noteCommand is a command of the bot that names the permission the test gives it.
type noteCommand struct {
	permission string
}

func (c noteCommand) Meta() pluginapi.TelegramCommandMeta {
	return pluginapi.TelegramCommandMeta{
		Command: "notes", Description: "List the notes", Permission: c.permission,
	}
}

func (noteCommand) Validate(telego.Update) error { return nil }

func (noteCommand) Handle(*th.Context, telego.Update) error { return nil }

func notePermissions() []pluginapi.Permission {
	return []pluginapi.Permission{
		{Name: notesRead, Description: "Read the notes", Lowest: pluginapi.RoleViewer},
		{Name: notesWrite, Description: "Write the notes", Lowest: pluginapi.RoleEditor},
	}
}

func noteRoute(method string, permission string) pluginapi.Route {
	return pluginapi.Route{
		Method:     method,
		Pattern:    "/notes",
		Handler:    func(http.ResponseWriter, *http.Request) {},
		Permission: permission,
	}
}

func noteTool(permission string) *pluginapi.TypedTool[codeInput, codeOutput] {
	return pluginapi.NewTypedTool(
		pluginapi.MCPToolMeta{
			Name:        "notes",
			Description: "List the notes.",
			Permission:  permission,
		},
		func(context.Context, codeInput) (codeOutput, error) { return codeOutput{}, nil },
	)
}

// registrars holds every registry that the registrars of a permission and of an
// entry write into, in the order of the loader.
type registrars struct {
	permissions  *registry.PermissionRegistry
	capabilities *registry.CapabilityRegistry
	handlers     *handler.Registry
	commands     *registry.TelegramCommandRegistry
	tools        *registry.MCPToolRegistry
	list         []registrar.Registrar
}

func newRegistrars() *registrars {
	set := &registrars{
		permissions:  registry.NewPermissionRegistry(),
		capabilities: registry.NewCapabilityRegistry(),
		handlers:     handler.NewRegistry(),
		commands:     registry.NewTelegramCommandRegistry(),
		tools:        registry.NewMCPToolRegistry(),
	}

	set.list = []registrar.Registrar{
		registrar.NewPermissionRegistrar(set.permissions, set.capabilities),
		registrar.NewHandlerRegistrar(
			slog.New(
				slog.DiscardHandler,
			),
			nil,
			nil,
			set.handlers,
			set.capabilities,
			set.permissions,
			nil,
			handler.WorkspaceAccess{Authorizer: authz.NewRoleAuthorizer(set.permissions)},
		),
		registrar.NewTelegramCommandRegistrar(
			set.commands, set.capabilities, nil, set.permissions,
			authz.NewRoleAuthorizer(set.permissions),
		),
		registrar.NewMCPToolRegistrar(set.tools, set.capabilities, set.permissions),
	}

	return set
}

// load runs every registrar that supports the plugin, as the loader does, and stops
// at the first failure.
func (set *registrars) load(plugin pluginapi.Plugin) error {
	for _, one := range set.list {
		if !one.Supports(plugin) {
			continue
		}

		if err := one.Register(plugin); err != nil {
			return err //nolint:wrapcheck // the test reads the error as it is.
		}
	}

	return nil
}

// PERMS-SC-010: The registry holds each permission of a plugin under the prefix of
// the plugin, at the lowest role that the plugin declares.
func TestTheRegistryHoldsThePermissionsOfAPlugin(t *testing.T) {
	t.Parallel()

	set := newRegistrars()

	require.NoError(t, set.load(&probe{id: toolPluginID, permissions: notePermissions()}))

	read, found := set.permissions.Get(probeNotesRead)
	require.True(t, found)
	assert.Equal(t, pluginapi.RoleViewer, read.Lowest)

	write, found := set.permissions.Get("dev.maroid.probe:notes.write")
	require.True(t, found)
	assert.Equal(t, pluginapi.RoleEditor, write.Lowest)

	assert.Equal(t, []registry.PermissionItem{
		{Name: probeNotesRead, Role: pluginapi.RoleViewer},
		{Name: "dev.maroid.probe:notes.write", Role: pluginapi.RoleEditor},
	}, set.capabilities.Of(toolPluginID)[registry.CapPermissions])
}

// PERMS-SC-011: A route that names no permission, a command that names one the
// plugin does not declare, and a plugin that declares one name twice each fail the
// load, name the plugin and the entry, and leave nothing of that kind registered.
func TestAnEntryWithNoDeclaredPermissionFailsTheLoad(t *testing.T) {
	t.Parallel()

	duplicated := append(notePermissions(), pluginapi.Permission{
		Name: notesRead, Lowest: pluginapi.RoleManager,
	})

	cases := []struct {
		name   string
		plugin *probe
		target error
		entry  string
	}{
		{
			name: "a route with no permission",
			plugin: &probe{
				id: toolPluginID, permissions: notePermissions(),
				routes: []pluginapi.Route{noteRoute(http.MethodGet, "")},
			},
			target: errs.ErrPermissionMissing,
			entry:  "GET /notes",
		},
		{
			name: "a command with an undeclared permission",
			plugin: &probe{
				id: toolPluginID, permissions: notePermissions(),
				commands: []pluginapi.TelegramCommand{noteCommand{permission: "notes.delete"}},
			},
			target: errs.ErrPermissionUndeclared,
			entry:  "notes.delete",
		},
		{
			name: "a permission declared twice",
			plugin: &probe{
				id: toolPluginID, permissions: duplicated,
				tools: []pluginapi.MCPTool{noteTool(notesRead)},
			},
			target: errs.ErrPermissionAlreadyRegistered,
			entry:  notesRead,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			set := newRegistrars()
			err := set.load(tc.plugin)

			require.ErrorIs(t, err, tc.target)
			assert.Contains(t, err.Error(), toolPluginID)
			assert.Contains(t, err.Error(), tc.entry)
			assert.Empty(t, set.capabilities.Of(toolPluginID)[registry.CapAPI], "no route")
			assert.Empty(t, set.commands.All())
			assert.Empty(t, set.tools.All())
		})
	}
}

// PERMS-SC-012: Two plugins that declare one local name hold two permissions, each at
// its own lowest role. The refusal of the route waits for the check of the routes.
func TestOneLocalNameOfTwoPluginsNamesTwoPermissions(t *testing.T) {
	t.Parallel()

	set := newRegistrars()

	require.NoError(t, set.load(&probe{id: "dev.maroid.p", permissions: []pluginapi.Permission{
		{Name: notesWrite, Lowest: pluginapi.RoleEditor},
	}}))
	require.NoError(t, set.load(&probe{id: "dev.maroid.q", permissions: []pluginapi.Permission{
		{Name: notesWrite, Lowest: pluginapi.RoleManager},
	}}))

	ofP, found := set.permissions.Get("dev.maroid.p:notes.write")
	require.True(t, found)
	assert.Equal(t, pluginapi.RoleEditor, ofP.Lowest)

	ofQ, found := set.permissions.Get("dev.maroid.q:notes.write")
	require.True(t, found)
	assert.Equal(t, pluginapi.RoleManager, ofQ.Lowest)
}

// PCAP-SC-004: The report of a plugin carries the permission of each route under the
// prefix of the workspace, and the permission of each command.
func TestTheReportCarriesThePermissionOfEachEntry(t *testing.T) {
	t.Parallel()

	set := newRegistrars()

	require.NoError(t, set.load(&probe{
		id:          toolPluginID,
		permissions: notePermissions(),
		routes: []pluginapi.Route{
			noteRoute(http.MethodGet, notesRead),
			noteRoute(http.MethodPost, notesWrite),
		},
		commands: []pluginapi.TelegramCommand{noteCommand{permission: notesRead}},
	}))

	report := set.capabilities.Of(toolPluginID)
	prefix := "/workspaces/{workspaceId}/plugins/dev.maroid.probe/api"

	assert.Equal(t, []registry.APIRoute{
		{
			Method:     http.MethodGet,
			Path:       prefix + "/notes",
			Permission: probeNotesRead,
		},
		{
			Method:     http.MethodPost,
			Path:       prefix + "/notes",
			Permission: "dev.maroid.probe:notes.write",
		},
	}, report[registry.CapAPI])

	assert.Equal(t, []registry.TelegramCommand{{
		Command:     "dev_maroid_probe_notes",
		Description: "List the notes (plugin dev.maroid.probe)",
		Permission:  probeNotesRead,
	}}, report[registry.CapTelegramCommands])
}
