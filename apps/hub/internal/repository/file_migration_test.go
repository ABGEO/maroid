package repository_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	checkViolation = "23514"
	probePlugin    = "dev.maroid.probe"
)

// fileOwners holds two users and two workspaces. Each user manages the workspace of
// the same letter.
type fileOwners struct {
	instance   *testdb.Instance
	userA      string
	userB      string
	workspaceA string
	workspaceB string
}

func newFileOwners(t *testing.T) *fileOwners {
	t.Helper()

	instance := startWithCoreMigrations(t)
	owners := &fileOwners{
		instance: instance,
		userA:    insertUser(t, instance, nameOfA),
		userB:    insertUser(t, instance, nameOfB),
	}
	owners.workspaceA = managedWorkspace(t, instance, owners.userA)
	owners.workspaceB = managedWorkspace(t, instance, owners.userB)

	return owners
}

func managedWorkspace(t *testing.T, instance *testdb.Instance, manager string) string {
	t.Helper()

	var workspace string

	require.NoError(t, instance.DB.Get(&workspace,
		`INSERT INTO public.workspaces (name) VALUES ('W') RETURNING id;`))

	_, err := instance.DB.Exec(
		`INSERT INTO public.workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'manager');`,
		workspace,
		manager,
	)
	require.NoError(t, err)

	return workspace
}

// insertFile writes a pending file of the probe in the scope of the context, and
// answers its identifier.
func insertFile(ctx context.Context, t *testing.T, db *sqlx.DB, table string) string {
	t.Helper()

	var id string

	require.NoError(t, database.WithScopeTx(ctx, db, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &id, `INSERT INTO `+table+
			` (plugin_id, kind, name, media_type, size)
			VALUES ($1, 'photo', 'cat.jpg', 'image/jpeg', 3145728) RETURNING id;`,
			probePlugin,
		)
	}))

	return id
}

// updateFile runs one update of the file in the scope of the context.
func updateFile(ctx context.Context, db *sqlx.DB, table, set, id string, value any) error {
	return execInScope(ctx, db, `UPDATE `+table+` SET `+set+` = $1 WHERE id = $2;`, value, id)
}

func execInScope(ctx context.Context, db *sqlx.DB, query string, args ...any) error {
	err := database.WithScopeTx(ctx, db, func(tx *sqlx.Tx) error {
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("running the statement: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("updating the file: %w", err)
	}

	return nil
}

func requireCheckViolation(t *testing.T, err error) {
	t.Helper()

	var pgErr *pgconn.PgError

	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, checkViolation, pgErr.Code)
}

// FILES-SC-044: The trigger refuses an update of the plugin, the kind, or the
// workspace of a workspace file, and lets every other column change.
func TestAWorkspaceFileKeepsItsPluginItsKindAndItsWorkspace(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	owners := newFileOwners(t)
	ctx := actingIn(context.Background(), owners.userA, owners.workspaceA)
	file := insertFile(ctx, t, owners.instance.DB, "public.workspace_files")

	for column, value := range map[string]any{
		"plugin_id":    "dev.maroid.other",
		"kind":         "paper",
		"workspace_id": owners.workspaceB,
	} {
		requireCheckViolation(
			t,
			updateFile(ctx, owners.instance.DB, "public.workspace_files", column, file, value),
		)
	}

	require.NoError(t, updateFile(ctx, owners.instance.DB, "public.workspace_files", "state", file,
		string(pluginapi.FileStateProcessing)))
}

// FILES-SC-044: The trigger refuses an update of the user of a user file.
func TestAUserFileKeepsItsUser(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	owners := newFileOwners(t)
	ctx := actingIn(context.Background(), owners.userA, owners.workspaceA)
	file := insertFile(ctx, t, owners.instance.DB, "public.user_files")

	requireCheckViolation(
		t,
		updateFile(ctx, owners.instance.DB, "public.user_files", "user_id", file, owners.userB),
	)
}

// FILES-SC-042: A row of a user file reaches no other user, in a workspace that both
// of them could act in.
func TestAUserFileReachesNoOtherUser(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	owners := newFileOwners(t)
	file := insertFile(
		actingIn(context.Background(), owners.userA, owners.workspaceA),
		t,
		owners.instance.DB,
		"public.user_files",
	)

	ctx := actingIn(context.Background(), owners.userB, owners.workspaceA)

	var count int

	require.NoError(t, database.WithScopeTx(ctx, owners.instance.DB, func(tx *sqlx.Tx) error {
		return tx.GetContext(
			ctx,
			&count,
			`SELECT count(*) FROM public.user_files WHERE id = $1;`,
			file,
		)
	}))
	require.Zero(t, count)
}

// FILES-SC-012: A failed file names its reason, and a file in any other state names none.
func TestAFailedFileNamesItsReason(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	owners := newFileOwners(t)
	ctx := actingIn(context.Background(), owners.userA, owners.workspaceA)
	file := insertFile(ctx, t, owners.instance.DB, "public.workspace_files")

	requireCheckViolation(
		t,
		updateFile(ctx, owners.instance.DB, "public.workspace_files", "state", file,
			string(pluginapi.FileStateFailed)),
	)
	requireCheckViolation(
		t,
		updateFile(ctx, owners.instance.DB, "public.workspace_files", "failure", file,
			string(pluginapi.FileFailureDamaged)),
	)

	require.NoError(t, execInScope(ctx, owners.instance.DB,
		`UPDATE public.workspace_files SET state = 'failed', failure = 'media_type' WHERE id = $1;`,
		file))
}
