package database_test

import (
	"context"
	"io/fs"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
)

// fixture holds one table that a workspace scopes and one that a user scopes, in
// the schema of the hub.
const fixture = `
CREATE TABLE public.scope_notes_of_workspace (
    id           UUID NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    body         TEXT NOT NULL
);

ALTER TABLE public.scope_notes_of_workspace ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.scope_notes_of_workspace FORCE ROW LEVEL SECURITY;

CREATE POLICY scope_notes_of_workspace_isolation ON public.scope_notes_of_workspace
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

CREATE TABLE public.scope_notes_of_user (
    id      UUID NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    body    TEXT NOT NULL
);

ALTER TABLE public.scope_notes_of_user ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.scope_notes_of_user FORCE ROW LEVEL SECURITY;

CREATE POLICY scope_notes_of_user_isolation ON public.scope_notes_of_user
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);
`

func startWithScopedTables(t *testing.T) *testdb.Instance {
	t.Helper()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	_, err = instance.DB.ExecContext(t.Context(), fixture)
	require.NoError(t, err)

	return instance
}

func insertReturningID(t *testing.T, instance *testdb.Instance, query string) string {
	t.Helper()

	var id string

	require.NoError(t, instance.DB.GetContext(t.Context(), &id, query))

	return id
}

func count(ctx context.Context, t *testing.T, instance *testdb.Instance, table string) int {
	t.Helper()

	var found int

	require.NoError(t, database.WithScopeTx(ctx, instance.DB, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &found, `SELECT count(*) FROM `+table+`;`)
	}))

	return found
}

func write(ctx context.Context, t *testing.T, instance *testdb.Instance, table string) {
	t.Helper()

	require.NoError(t, database.WithScopeTx(ctx, instance.DB, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO `+table+` (body) VALUES ('note');`)

		return err //nolint:wrapcheck // the test reads the error as it is.
	}))
}

// IDENT-SC-001, IDENT-SC-005: The transaction of the hub carries the acting
// workspace and the acting user, each into the policy of its own scope, and a
// context that carries neither reaches no scoped row.
func TestWithScopeTxCarriesBothScopes(t *testing.T) {
	t.Parallel()

	instance := startWithScopedTables(t)

	workspaceA := insertReturningID(
		t,
		instance,
		`INSERT INTO public.workspaces (name) VALUES ('A') RETURNING id;`,
	)
	workspaceB := insertReturningID(
		t,
		instance,
		`INSERT INTO public.workspaces (name) VALUES ('B') RETURNING id;`,
	)
	userA := insertReturningID(t, instance, `INSERT INTO public.users DEFAULT VALUES RETURNING id;`)

	inA := pluginapi.ContextWithActingUser(
		pluginapi.ContextWithActingWorkspace(t.Context(), workspaceA),
		userA,
	)
	inB := pluginapi.ContextWithActingWorkspace(t.Context(), workspaceB)

	write(inA, t, instance, "public.scope_notes_of_workspace")
	write(inA, t, instance, "public.scope_notes_of_user")

	require.Equal(t, 1, count(inA, t, instance, "public.scope_notes_of_workspace"))
	require.Equal(t, 1, count(inA, t, instance, "public.scope_notes_of_user"))
	require.Equal(t, 0, count(inB, t, instance, "public.scope_notes_of_workspace"))
	require.Equal(t, 0, count(inB, t, instance, "public.scope_notes_of_user"))
	require.Equal(t, 0, count(t.Context(), t, instance, "public.scope_notes_of_workspace"))
	require.Equal(t, 0, count(t.Context(), t, instance, "public.scope_notes_of_user"))
}
