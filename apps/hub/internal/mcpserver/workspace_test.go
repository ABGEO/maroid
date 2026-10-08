package mcpserver_test

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/mcpserver"
	"github.com/abgeo/maroid/apps/hub/internal/mcpserver/tools"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
)

// scopedNotes is a table that a workspace scopes, as the table of a plugin is.
const scopedNotes = `
CREATE TABLE public.mcp_test_notes (
    id           UUID NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    body         TEXT NOT NULL
);

ALTER TABLE public.mcp_test_notes ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.mcp_test_notes FORCE ROW LEVEL SECURITY;

CREATE POLICY mcp_test_notes_isolation ON public.mcp_test_notes
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);
`

const (
	notesRead  = "notes.read"
	notesWrite = "notes.write"
)

type notesInput struct{}

type notesOutput struct {
	Bodies []string `json:"bodies"`
}

// workspaceWorld holds Ana, a manager of A and B, Gio, a viewer of A, and workspace C
// of another person. A holds two notes and B one.
type workspaceWorld struct {
	instance *testdb.Instance
	ana, gio string
	a, b, c  string
	runs     *atomic.Int32
	writes   *atomic.Int32
	notes    registry.MCPTool
	jot      registry.MCPTool
}

func newWorkspaceWorld(t *testing.T) *workspaceWorld {
	t.Helper()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	_, err = instance.DB.ExecContext(t.Context(), scopedNotes)
	require.NoError(t, err)

	insert := func(query string) string {
		var id string

		require.NoError(t, instance.DB.GetContext(t.Context(), &id, query))

		return id
	}
	scene := &workspaceWorld{instance: instance, runs: &atomic.Int32{}, writes: &atomic.Int32{}}
	scene.ana = insert(`INSERT INTO public.users (first_name) VALUES ('Ana') RETURNING id;`)
	scene.gio = insert(`INSERT INTO public.users (first_name) VALUES ('Gio') RETURNING id;`)
	other := insert(`INSERT INTO public.users (first_name) VALUES ('Nino') RETURNING id;`)

	for _, place := range []*string{&scene.a, &scene.b, &scene.c} {
		*place = insert(`INSERT INTO public.workspaces (name) VALUES ('W') RETURNING id;`)
	}

	_, err = instance.DB.ExecContext(
		t.Context(),
		`UPDATE public.workspaces SET name = CASE id WHEN $1 THEN 'A' WHEN $2 THEN 'B' ELSE 'C' END;`,
		scene.a,
		scene.b,
	)
	require.NoError(t, err)

	scene.member(t, scene.a, scene.ana, pluginapi.RoleManager)
	scene.member(t, scene.b, scene.ana, pluginapi.RoleManager)
	scene.member(t, scene.a, scene.gio, pluginapi.RoleViewer)
	scene.member(t, scene.c, other, pluginapi.RoleManager)

	scene.write(t, scene.a, "first of A", "second of A")
	scene.write(t, scene.b, "one of B")

	// A enables the probe plugin. B does not, so a call in B meets a disabled plugin.
	scene.enable(t, scene.a)

	scene.notes = scene.notesTool(t)
	scene.jot = scene.jotTool(t)

	return scene
}

// jotTool builds a tool of the probe plugin that writes, which an editor reaches,
// and counts its runs.
func (scene *workspaceWorld) jotTool(t *testing.T) registry.MCPTool {
	t.Helper()

	entry, err := mcpserver.NewPluginTool(probeID(t), pluginapi.NewTypedTool(
		pluginapi.MCPToolMeta{
			Name:        "jot",
			Title:       "Write a note",
			Description: "Write a note.",
			Permission:  notesWrite,
		},
		func(context.Context, notesInput) (notesOutput, error) {
			scene.writes.Add(1)

			return notesOutput{}, nil
		},
	))
	require.NoError(t, err)

	return entry
}

// authorizer holds the permissions of the hub, and the two of the probe plugin.
func (scene *workspaceWorld) authorizer(t *testing.T) *authz.RoleAuthorizer {
	t.Helper()

	permissions, err := authz.NewPermissionRegistry()
	require.NoError(t, err)

	probe := probeID(t)
	require.NoError(t, permissions.Register(
		registry.PermissionEntry{
			Name: registry.PermissionName(probe, notesRead), Lowest: pluginapi.RoleViewer,
		},
		registry.PermissionEntry{
			Name: registry.PermissionName(probe, notesWrite), Lowest: pluginapi.RoleEditor,
		},
	))

	return authz.NewRoleAuthorizer(permissions)
}

// notesTool builds a tool of the probe plugin that lists the notes of the acting
// workspace, and counts its runs.
func (scene *workspaceWorld) notesTool(t *testing.T) registry.MCPTool {
	t.Helper()

	entry, err := mcpserver.NewPluginTool(probeID(t), pluginapi.NewTypedTool(
		pluginapi.MCPToolMeta{
			Name:        "notes",
			Title:       "List the notes",
			Description: "List the notes.",
			Permission:  notesRead,
		},
		func(ctx context.Context, _ notesInput) (notesOutput, error) {
			scene.runs.Add(1)

			bodies := []string{}

			readErr := database.WithScopeTx(ctx, scene.instance.DB, func(tx *sqlx.Tx) error {
				return tx.SelectContext(ctx, &bodies,
					`SELECT body FROM public.mcp_test_notes ORDER BY body;`)
			})

			return notesOutput{
				Bodies: bodies,
			}, readErr //nolint:wrapcheck // the test reads it as it is.
		},
	))
	require.NoError(t, err)

	return entry
}

func (scene *workspaceWorld) member(
	t *testing.T,
	workspaceID string,
	user string,
	role pluginapi.Role,
) {
	t.Helper()

	_, err := scene.instance.DB.ExecContext(t.Context(),
		`INSERT INTO public.workspace_members (workspace_id, user_id, role) VALUES ($1, $2, $3);`,
		workspaceID, user, role)
	require.NoError(t, err)
}

func (scene *workspaceWorld) enable(t *testing.T, workspaceID string) {
	t.Helper()

	_, err := scene.instance.DB.ExecContext(t.Context(),
		`INSERT INTO public.workspace_plugins (workspace_id, plugin_id) VALUES ($1, $2);`,
		workspaceID, probePluginID)
	require.NoError(t, err)
}

func (scene *workspaceWorld) write(t *testing.T, workspace string, bodies ...string) {
	t.Helper()

	ctx := pluginapi.ContextWithActingWorkspace(t.Context(), workspace)

	require.NoError(t, database.WithScopeTx(ctx, scene.instance.DB, func(tx *sqlx.Tx) error {
		for _, body := range bodies {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO public.mcp_test_notes (body) VALUES ($1);`, body); err != nil {
				return err //nolint:wrapcheck // the test reads it as it is.
			}
		}

		return nil
	}))
}

// sessionAs connects a client to the server of the hub, as NewServer builds it,
// with the acting user that a verified token would give.
func (scene *workspaceWorld) sessionAs(
	t *testing.T,
	user string,
	entries ...registry.MCPTool,
) *mcp.ClientSession {
	t.Helper()

	toolRegistry := registry.NewMCPToolRegistry()
	require.NoError(t, toolRegistry.Register(entries...))

	server := mcpserver.NewServer(
		slog.New(slog.DiscardHandler),
		toolRegistry,
		scene.instance.DB,
		workspace.NewEnablements(scene.instance.DB, registry.NewPluginRegistry()),
		scene.authorizer(t),
	)
	server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			return next(pluginapi.ContextWithActingUser(ctx, user), method, req)
		}
	})

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: testVersion}, nil)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(t.Context(), serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	clientSession, err := client.Connect(t.Context(), clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = clientSession.Close() })

	return clientSession
}

func structured(t *testing.T, result *mcp.CallToolResult, target any) {
	t.Helper()

	require.False(t, result.IsError, "the call failed: %v", result.Content)

	encoded, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(encoded, target))
}

// MCPHUB-SC-026: list_workspaces names each workspace of the acting user with the
// role, and no workspace of another person.
func TestListWorkspacesNamesTheMembershipsOfTheActingUser(t *testing.T) {
	t.Parallel()

	scene := newWorkspaceWorld(t)
	entry := tools.NewListWorkspaces(scene.instance.DB)

	result, err := scene.sessionAs(t, scene.ana, entry).CallTool(t.Context(), &mcp.CallToolParams{
		Name: entry.Name, Arguments: json.RawMessage(`{}`),
	})
	require.NoError(t, err)

	var answered tools.ListWorkspacesOutput

	structured(t, result, &answered)

	names := map[string]string{}
	for _, one := range answered.Workspaces {
		names[one.ID] = one.Name + " " + string(one.Role)
	}

	assert.Equal(t, map[string]string{scene.a: "A manager", scene.b: "B manager"}, names)
}

// MCPHUB-SC-027: A tool of a plugin reads the rows of the workspace of its call, and
// a call that names no workspace fails on the member workspace.
// MCPHUB-INV-003: One token reaches each workspace of its user one call at a time.
func TestAToolActsInTheWorkspaceOfItsCall(t *testing.T) {
	t.Parallel()

	scene := newWorkspaceWorld(t)
	session := scene.sessionAs(t, scene.ana, scene.notes)

	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name: scene.notes.Name, Arguments: json.RawMessage(`{"workspace":"` + scene.a + `"}`),
	})
	require.NoError(t, err)

	var answered notesOutput

	structured(t, result, &answered)
	assert.Equal(t, []string{"first of A", "second of A"}, answered.Bodies)

	missing, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name: scene.notes.Name, Arguments: json.RawMessage(`{}`),
	})
	require.NoError(t, err)
	require.True(t, missing.IsError)

	text, ok := missing.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, "workspace")
}

// MCPHUB-SC-028: A workspace of no membership and a workspace that does not enable the
// plugin answer the error of a tool that does not exist, and the tool runs zero times.
func TestAWorkspaceOfNoMembershipAnswersAsAnUnknownTool(t *testing.T) {
	t.Parallel()

	scene := newWorkspaceWorld(t)
	session := scene.sessionAs(t, scene.ana, scene.notes)

	_, refused := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name: scene.notes.Name, Arguments: json.RawMessage(`{"workspace":"` + scene.c + `"}`),
	})
	require.Error(t, refused)

	_, unknown := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name: "dev_maroid_probe_absent", Arguments: json.RawMessage(`{}`),
	})
	require.Error(t, unknown)

	assert.Equal(t,
		replaceName(unknown.Error(), "dev_maroid_probe_absent"),
		replaceName(refused.Error(), scene.notes.Name),
		"both answers carry one error")

	_, disabled := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name: scene.notes.Name, Arguments: json.RawMessage(`{"workspace":"` + scene.b + `"}`),
	})
	require.Error(t, disabled)
	assert.Equal(t,
		replaceName(unknown.Error(), "dev_maroid_probe_absent"),
		replaceName(disabled.Error(), scene.notes.Name),
		"a workspace that does not enable the plugin answers the same error")
	assert.Zero(t, scene.runs.Load())
}

// MCPHUB-DD-022: A model of a plugin that declares the member workspace fails the load.
func TestAModelThatDeclaresTheWorkspaceFailsTheLoad(t *testing.T) {
	t.Parallel()

	type claimingInput struct {
		Workspace string `json:"workspace"`
	}

	_, err := mcpserver.NewPluginTool(probeID(t), pluginapi.NewTypedTool(
		pluginapi.MCPToolMeta{Name: "claim", Title: "Claim", Description: "Claim."},
		func(context.Context, claimingInput) (notesOutput, error) { return notesOutput{}, nil },
	))

	require.ErrorContains(t, err, "workspace")
}

// replaceName removes the name of the tool, which the error of the SDK quotes.
func replaceName(text string, name string) string {
	return strings.ReplaceAll(text, name, "<tool>")
}

// MCPHUB-SC-029: A viewer calls a tool whose permission needs an editor. The result
// carries isError and names the permission and the role, and the tool runs zero times.
// PERMS-SC-013: The tool of the probe refuses the viewer the same way as the route.
func TestAToolOfARoleBelowItsPermissionRunsNothing(t *testing.T) {
	t.Parallel()

	scene := newWorkspaceWorld(t)
	session := scene.sessionAs(t, scene.gio, scene.notes, scene.jot)

	refused, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name: scene.jot.Name, Arguments: json.RawMessage(`{"workspace":"` + scene.a + `"}`),
	})
	require.NoError(t, err)
	require.True(t, refused.IsError)

	text, ok := refused.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, "dev.maroid.probe:notes.write")
	assert.Contains(t, text.Text, "editor")
	assert.Zero(t, scene.writes.Load())

	read, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name: scene.notes.Name, Arguments: json.RawMessage(`{"workspace":"` + scene.a + `"}`),
	})
	require.NoError(t, err)
	assert.False(t, read.IsError, "a viewer reaches the read tool")
}
