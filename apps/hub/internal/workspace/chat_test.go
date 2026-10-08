package workspace_test

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
)

const chatID = int64(-1001234567890)

// chatWorld holds workspaces H, G, and K. Ana is a member of H and G, Gio of H alone,
// and Beka of H, G, and K.
type chatWorld struct {
	instance  *testdb.Instance
	selection *workspace.ChatSelection
	h, g, k   string
	ana, gio  string
	beka      string
}

func newChatWorld(t *testing.T) *chatWorld {
	t.Helper()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	scene := &chatWorld{
		instance:  instance,
		selection: workspace.NewChatSelection(instance.DB),
	}

	person := func(name string) string {
		return insertID(t, instance,
			`INSERT INTO public.users (first_name) VALUES ('`+name+`') RETURNING id;`)
	}
	place := func(name string, people ...string) string {
		id := insertID(t, instance,
			`INSERT INTO public.workspaces (name) VALUES ('`+name+`') RETURNING id;`)

		for _, member := range people {
			_, err := instance.DB.ExecContext(
				t.Context(),
				`INSERT INTO public.workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'manager');`,
				id,
				member,
			)
			require.NoError(t, err)
		}

		return id
	}

	scene.ana, scene.gio, scene.beka = person("Ana"), person("Gio"), person("Beka")
	scene.h = place("H", scene.ana, scene.gio, scene.beka)
	scene.g = place("G", scene.ana, scene.beka)
	scene.k = place("K", scene.beka)

	return scene
}

func as(user string) context.Context {
	return pluginapi.ContextWithActingUser(context.Background(), user)
}

// stored reads the selection that the row of the chat holds for the person, or nil.
func (scene *chatWorld) stored(t *testing.T, user string) *string {
	t.Helper()

	ctx := as(user)

	var selected sql.NullString

	require.NoError(t, database.WithScopeTx(ctx, scene.instance.DB, func(tx *sqlx.Tx) error {
		err := tx.GetContext(ctx, &selected,
			`SELECT selected_workspace_id FROM public.telegram_chats WHERE chat_id = $1;`, chatID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}

		return err //nolint:wrapcheck // the test reads the error as it is.
	}))

	if !selected.Valid {
		return nil
	}

	return &selected.String
}

func names(t *testing.T, scene *chatWorld, user string) []string {
	t.Helper()

	choices, err := scene.selection.Choices(as(user))
	require.NoError(t, err)

	found := make([]string, 0, len(choices))
	for _, choice := range choices {
		found = append(found, choice.Name)
	}

	return found
}

// WSPACE-SC-013: A person of two workspaces picks one, and every later update of the
// chat acts in it.
func TestAPickedWorkspaceServesEveryLaterUpdate(t *testing.T) {
	t.Parallel()

	scene := newChatWorld(t)

	acting, _, err := scene.selection.Acting(as(scene.ana), chatID)
	require.NoError(t, err)
	assert.Empty(t, acting, "two memberships and no selection leave the choice to Ana")
	assert.ElementsMatch(t, []string{"H", "G"}, names(t, scene, scene.ana))

	picked, err := scene.selection.Select(as(scene.ana), chatID, scene.g)
	require.NoError(t, err)
	assert.Equal(t, "G", picked.Name)

	for range 2 {
		acting, _, err = scene.selection.Acting(as(scene.ana), chatID)
		require.NoError(t, err)
		assert.Equal(t, scene.g, acting)
	}

	require.NotNil(t, scene.stored(t, scene.ana))
	assert.Equal(t, scene.g, *scene.stored(t, scene.ana))
}

// WSPACE-SC-014: A person of one workspace acts in it with no pick, and the chat
// stores it.
func TestTheOnlyWorkspaceNeedsNoPick(t *testing.T) {
	t.Parallel()

	scene := newChatWorld(t)

	acting, role, err := scene.selection.Acting(as(scene.gio), chatID)
	require.NoError(t, err)
	assert.Equal(t, scene.h, acting)
	assert.Equal(t, pluginapi.RoleManager, role, "the chat carries the role of the membership")

	require.NotNil(t, scene.stored(t, scene.gio))
	assert.Equal(t, scene.h, *scene.stored(t, scene.gio))
}

// WSPACE-SC-016: A removal clears the selection at the next update of the removed
// person, who then picks between the workspaces that remain.
func TestARemovedMemberLosesTheSelection(t *testing.T) {
	t.Parallel()

	scene := newChatWorld(t)

	_, err := scene.selection.Select(as(scene.beka), chatID, scene.h)
	require.NoError(t, err)

	require.NoError(t, database.WithTx(t.Context(), scene.instance.DB, func(tx *sqlx.Tx) error {
		return repository.NewWorkspaceMember(tx).Remove(t.Context(), scene.h, scene.beka)
	}))

	acting, _, err := scene.selection.Acting(as(scene.beka), chatID)
	require.NoError(t, err)
	assert.Empty(t, acting)
	assert.Nil(t, scene.stored(t, scene.beka), "the chat stores no selection")
	assert.ElementsMatch(t, []string{"G", "K"}, names(t, scene, scene.beka))
}

// WSPACE-SC-016: A tap on a workspace of no membership writes nothing.
func TestAPickOfAWorkspaceOfNoMembershipWritesNothing(t *testing.T) {
	t.Parallel()

	scene := newChatWorld(t)

	_, err := scene.selection.Select(as(scene.gio), chatID, scene.k)
	require.ErrorIs(t, err, errs.ErrMemberNotFound)
	assert.Nil(t, scene.stored(t, scene.gio))
}

// WSPACE-FR-016: Two people in one chat hold a selection each.
func TestEachPersonOfAChatHoldsTheirOwnSelection(t *testing.T) {
	t.Parallel()

	scene := newChatWorld(t)

	_, err := scene.selection.Select(as(scene.ana), chatID, scene.g)
	require.NoError(t, err)

	_, err = scene.selection.Select(as(scene.beka), chatID, scene.k)
	require.NoError(t, err)

	assert.Equal(t, scene.g, *scene.stored(t, scene.ana))
	assert.Equal(t, scene.k, *scene.stored(t, scene.beka))
}
