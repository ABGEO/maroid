package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	hubdatabase "github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/file"
	"github.com/abgeo/maroid/apps/hub/internal/filestore"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	typeMembersRemain = "/problems/hub/members-remain"
	typeInternal      = "/problems/http/internal"
	otherMembers      = "other_members"
	lockWait          = 10 * time.Second
)

var errRefused = errors.New("the stub refuses every call")

// refusingObjects is a file store that fails and counts every call.
type refusingObjects struct {
	calls atomic.Int32
}

func (o *refusingObjects) Ping(context.Context) error {
	o.calls.Add(1)

	return errRefused
}

func (o *refusingObjects) PresignPut(
	context.Context, string, int64, string,
) (filestore.Upload, error) {
	o.calls.Add(1)

	return filestore.Upload{}, errRefused
}

func (o *refusingObjects) PresignGet(context.Context, string, string) (filestore.Address, error) {
	o.calls.Add(1)

	return filestore.Address{}, errRefused
}

// gatedPurger holds the transaction of a removal open after its lock and its count,
// until the test releases it.
type gatedPurger struct {
	locked  chan struct{}
	release chan struct{}
}

func (g gatedPurger) PurgeWorkspace(context.Context, *sqlx.Tx) (int, error) {
	close(g.locked)
	<-g.release

	return 0, nil
}

func (f *workspaceFixture) remove(
	t *testing.T,
	who person,
	workspaceID string,
) *httptest.ResponseRecorder {
	t.Helper()

	return f.call(t, who, http.MethodDelete, "/workspaces/"+workspaceID, nil, "")
}

// inScope runs fn in a transaction that acts as the user in the workspace, so the
// policy of each scoped table shows and takes their rows.
func (f *workspaceFixture) inScope(
	t *testing.T,
	workspaceID string,
	userID string,
	fn func(ctx context.Context, tx *sqlx.Tx) error,
) {
	t.Helper()

	ctx := pluginapi.ContextWithActingUser(t.Context(), userID)
	ctx = pluginapi.ContextWithActingWorkspace(ctx, workspaceID)

	require.NoError(t, hubdatabase.WithScopeTx(ctx, f.database, func(tx *sqlx.Tx) error {
		return fn(ctx, tx)
	}))
}

func (f *workspaceFixture) execIn(
	t *testing.T,
	workspaceID string,
	userID string,
	statement string,
	args ...any,
) {
	t.Helper()

	f.inScope(t, workspaceID, userID, func(ctx context.Context, tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, statement, args...)

		return err //nolint:wrapcheck // the test reads the error as it is.
	})
}

// countIn counts the rows of the table whose column names the workspace, as the user
// acting in it sees them.
func (f *workspaceFixture) countIn(
	t *testing.T,
	table string,
	column string,
	workspaceID string,
	userID string,
) int {
	t.Helper()

	var count int

	f.inScope(t, workspaceID, userID, func(ctx context.Context, tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &count,
			`SELECT count(*) FROM `+table+` WHERE `+column+` = $1;`, workspaceID)
	})

	return count
}

func (f *workspaceFixture) countTasks(t *testing.T) int {
	t.Helper()

	var count int

	require.NoError(t, f.database.GetContext(t.Context(), &count,
		`SELECT count(*) FROM public.outbox_tasks;`))

	return count
}

// addFile stores a file of the probe plugin that the person uploaded to the workspace,
// in the state, and answers its identifier.
func (f *workspaceFixture) addFile(
	t *testing.T,
	workspaceID string,
	uploader person,
	state string,
) string {
	t.Helper()

	var id string

	f.inScope(t, workspaceID, uploader.id, func(ctx context.Context, tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &id,
			`INSERT INTO public.workspace_files
				(plugin_id, kind, state, name, media_type, size, uploaded_by)
			VALUES ($1, 'photo', $2, 'leaf.webp', 'image/webp', 10, $3) RETURNING id;`,
			probePluginID, state, uploader.id)
	})

	return id
}

func (f *workspaceFixture) addSetting(t *testing.T, workspaceID string) {
	t.Helper()

	f.execIn(t, workspaceID, "",
		`INSERT INTO public.plugin_workspace_settings (plugin_id) VALUES ($1);`, probePluginID)
}

func (f *workspaceFixture) addSecret(t *testing.T, workspaceID string) {
	t.Helper()

	f.execIn(t, workspaceID, "",
		`INSERT INTO public.plugin_workspace_secrets (plugin_id, name, ciphertext)
		VALUES ($1, 'token', 'sealed');`, probePluginID)
}

func (f *workspaceFixture) addProviderWithRoute(t *testing.T, workspaceID string) {
	t.Helper()

	f.inScope(t, workspaceID, "", func(ctx context.Context, tx *sqlx.Tx) error {
		var provider string

		if err := tx.GetContext(ctx, &provider,
			`INSERT INTO public.notification_providers (kind, name)
			VALUES ('telegram', 'Home') RETURNING id;`); err != nil {
			return err //nolint:wrapcheck // the test reads the error as it is.
		}

		_, err := tx.ExecContext(ctx,
			`INSERT INTO public.notification_routes (provider_id, plugin_id, topic_id)
			VALUES ($1, $2, 'reminder');`, provider, probePluginID)

		return err //nolint:wrapcheck // the test reads the error as it is.
	})
}

func (f *workspaceFixture) selectInChat(
	t *testing.T,
	who person,
	chatID int64,
	workspaceID string,
) {
	t.Helper()

	f.execIn(t, "", who.id,
		`INSERT INTO public.telegram_chats (chat_id, selected_workspace_id) VALUES ($1, $2);`,
		chatID, workspaceID)
}

// waitForALockWait returns once a statement of another session waits for a lock.
func (f *workspaceFixture) waitForALockWait(t *testing.T) {
	t.Helper()

	require.Eventually(t, func() bool {
		var waiting int

		err := f.database.GetContext(t.Context(), &waiting,
			`SELECT count(*) FROM pg_locks WHERE NOT granted;`)

		return err == nil && waiting > 0
	}, lockWait, 10*time.Millisecond)
}

func names(t *testing.T, response *httptest.ResponseRecorder) []string {
	t.Helper()

	listed := items(t, response)

	found := make([]string, 0, len(listed))
	for _, item := range listed {
		found = append(found, stringOf(t, item[memberName]))
	}

	return found
}

// WSREMOVE-SC-001: The only member removes a workspace. It answers not-found, and the
// list of the person names the other workspace alone.
func TestTheOnlyMemberRemovesAWorkspace(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	spaceS := fixture.create(t, fixture.nino, "S")
	fixture.create(t, fixture.nino, "Second")

	response := fixture.remove(t, fixture.nino, spaceS)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())

	read := fixture.call(t, fixture.nino, http.MethodGet, "/workspaces/"+spaceS, nil, "")
	assert.Equal(t, http.StatusNotFound, read.Code)

	listed := fixture.call(t, fixture.nino, http.MethodGet, "/workspaces", nil, "")
	require.Equal(t, http.StatusOK, listed.Code)
	assert.Equal(t, []string{"Second"}, names(t, listed))
}

// WSREMOVE-SC-002: The removal of the last workspace of a person leaves an empty list.
func TestTheRemovalOfTheLastWorkspaceLeavesAnEmptyList(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	spaceS := fixture.create(t, fixture.nino, "S")

	response := fixture.remove(t, fixture.nino, spaceS)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())

	listed := fixture.call(t, fixture.nino, http.MethodGet, "/workspaces", nil, "")
	require.Equal(t, http.StatusOK, listed.Code)
	assert.Empty(t, items(t, listed))
}

// WSREMOVE-SC-003: A viewer and an editor cannot remove a workspace. The refusal names
// the permission and the lowest role that holds it.
func TestAViewerAndAnEditorCannotRemoveAWorkspace(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)

	for _, who := range []person{fixture.gio, fixture.beka} {
		refused := fixture.remove(t, who, fixture.h)
		require.Equal(t, http.StatusForbidden, refused.Code, refused.Body.String())

		answered := decode(t, refused)
		assert.Equal(t, "/problems/http/permission-denied", answered[problemType])
		assert.Equal(t, "workspace.remove", answered["permission"])
		assert.Equal(t, roleManager, answered["required_role"])
	}

	assert.Len(t, fixture.memberIDs(t, fixture.ana, fixture.h), 3)
}

// WSREMOVE-SC-004: A manager who shares the workspace removes nothing, and the answer
// counts the other members.
func TestAManagerWhoSharesTheWorkspaceRemovesNothing(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	fixture.addProviderWithRoute(t, fixture.h)
	fixture.addSetting(t, fixture.h)

	tasks := fixture.countTasks(t)

	refused := fixture.remove(t, fixture.ana, fixture.h)
	require.Equal(t, http.StatusConflict, refused.Code, refused.Body.String())
	assert.Equal(t, "application/problem+json", refused.Header().Get("Content-Type"))

	answered := decode(t, refused)
	assert.Equal(t, typeMembersRemain, answered[problemType])
	assert.InDelta(t, 2, answered[otherMembers], 0)
	assert.Equal(t, "The workspace holds 2 other members.", answered["detail"])

	assert.Len(t, fixture.memberIDs(t, fixture.ana, fixture.h), 3)
	assert.Equal(
		t,
		1,
		fixture.countIn(t, "public.notification_providers", "workspace_id", fixture.h, ""),
	)
	assert.Equal(
		t,
		1,
		fixture.countIn(t, "public.plugin_workspace_settings", "workspace_id", fixture.h, ""),
	)
	assert.Equal(t, tasks, fixture.countTasks(t))
}

// WSREMOVE-SC-005: A second manager blocks the removal as any other member does.
func TestASecondManagerBlocksTheRemoval(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	spaceS := fixture.create(t, fixture.nino, "S")
	fixture.add(t, fixture.nino, spaceS, fixture.ana, roleManager)

	refused := fixture.remove(t, fixture.nino, spaceS)
	require.Equal(t, http.StatusConflict, refused.Code, refused.Body.String())

	answered := decode(t, refused)
	assert.Equal(t, typeMembersRemain, answered[problemType])
	assert.InDelta(t, 1, answered[otherMembers], 0)
	assert.Equal(t, "The workspace holds 1 other member.", answered["detail"])
}

// WSREMOVE-SC-006: An addition that holds its transaction open blocks the removal that
// starts after it, and the removal then counts the new member.
func TestAnOpenAdditionBlocksTheRemoval(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	spaceS := fixture.create(t, fixture.nino, "S")

	tx, err := fixture.database.BeginTxx(t.Context(), nil)
	require.NoError(t, err)

	defer func() { _ = tx.Rollback() }()

	_, err = repository.NewWorkspaceMember(tx).
		Add(t.Context(), spaceS, fixture.beka.id, pluginapi.RoleEditor)
	require.NoError(t, err)

	request := fixture.request(t, fixture.nino, http.MethodDelete, "/workspaces/"+spaceS, nil, "")
	removal := make(chan *httptest.ResponseRecorder, 1)

	go func() { removal <- fixture.serve(request) }()

	fixture.waitForALockWait(t)
	require.NoError(t, tx.Commit())

	refused := <-removal
	require.Equal(t, http.StatusConflict, refused.Code, refused.Body.String())
	assert.InDelta(t, 1, decode(t, refused)[otherMembers], 0)
	assert.Contains(t, fixture.memberIDs(t, fixture.nino, spaceS), fixture.beka.id)
}

// WSREMOVE-SC-006: A removal that holds its lock turns the addition that starts after
// it into not-found.
func TestALockedRemovalTurnsTheAdditionIntoNotFound(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	admin := fixture.administrator(t)
	spaceS := fixture.create(t, fixture.nino, "S")

	gate := gatedPurger{locked: make(chan struct{}), release: make(chan struct{})}
	remover := workspace.NewRemover(slog.New(slog.DiscardHandler), fixture.database, gate)

	ctx := pluginapi.ContextWithActingUser(t.Context(), fixture.nino.id)
	ctx = pluginapi.ContextWithActingWorkspace(ctx, spaceS)
	ctx = auth.ContextWithAdministrator(ctx, false)

	removal := make(chan error, 1)

	go func() { removal <- remover.Remove(ctx) }()

	<-gate.locked

	request := fixture.request(t, admin, http.MethodPost, "/workspaces/"+spaceS+"/members",
		map[string]any{memberUserID: fixture.beka.id, memberRole: roleEditor}, "")
	addition := make(chan *httptest.ResponseRecorder, 1)

	go func() { addition <- fixture.serve(request) }()

	fixture.waitForALockWait(t)
	close(gate.release)

	require.NoError(t, <-removal)

	refused := <-addition
	require.Equal(t, http.StatusNotFound, refused.Code, refused.Body.String())
	assert.Equal(t, typeNotFound, decode(t, refused)[problemType])
}

// WSREMOVE-SC-007: An administrator who is no member removes a shared workspace, and
// no member reads it again.
func TestAnAdministratorRemovesASharedWorkspace(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	admin := fixture.administrator(t)

	response := fixture.remove(t, admin, fixture.h)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())

	for _, who := range []person{fixture.ana, fixture.beka, fixture.gio} {
		read := fixture.call(t, who, http.MethodGet, "/workspaces/"+fixture.h, nil, "")
		assert.Equal(t, http.StatusNotFound, read.Code)
	}
}

// WSREMOVE-SC-008: An administrator removes a workspace of which they are the only
// member.
func TestAnAdministratorRemovesTheirOwnWorkspace(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	admin := fixture.administrator(t)
	own := fixture.create(t, admin, "Own")

	response := fixture.remove(t, admin, own)
	assert.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
}

// WSREMOVE-SC-009: The removal deletes the row of the workspace in each table of the
// hub, clears the selection of a chat, and leaves the rows of another workspace.
func TestTheRemovalDeletesEveryRowOfTheWorkspaceAlone(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	spaceS := fixture.create(t, fixture.nino, "S")

	fixture.enable(t, spaceS, probePluginID)

	for space, uploader := range map[string]person{spaceS: fixture.nino, fixture.h: fixture.ana} {
		fixture.addSetting(t, space)
		fixture.addSecret(t, space)
		fixture.addProviderWithRoute(t, space)
		fixture.addFile(t, space, uploader, "ready")
	}

	fixture.selectInChat(t, fixture.nino, 4001, spaceS)
	fixture.selectInChat(t, fixture.ana, 4002, fixture.h)

	response := fixture.remove(t, fixture.nino, spaceS)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())

	tables := []string{
		"public.workspace_members",
		"public.workspace_plugins",
		"public.plugin_workspace_settings",
		"public.plugin_workspace_secrets",
		"public.notification_providers",
		"public.notification_routes",
		"public.workspace_files",
	}

	for _, table := range tables {
		assert.Zero(t, fixture.countIn(t, table, "workspace_id", spaceS, ""), table)
		assert.NotZero(t, fixture.countIn(t, table, "workspace_id", fixture.h, ""), table)
	}

	assert.Zero(
		t,
		fixture.countIn(
			t,
			"public.telegram_chats",
			"selected_workspace_id",
			spaceS,
			fixture.nino.id,
		),
	)
	assert.Equal(
		t,
		1,
		fixture.countIn(
			t,
			"public.telegram_chats",
			"selected_workspace_id",
			fixture.h,
			fixture.ana.id,
		),
	)
}

// WSREMOVE-SC-010: The removal deletes the rows of a schema that no plugin loads, past
// a reference with no action and a file that a photo restricts. A delete from a
// transaction with no scope leaves no row behind either.
func TestTheRemovalReachesATableOfNoLoadedPlugin(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	fixture.instance.Migrate(t, "probe", os.DirFS("testdata/probe"))

	spaceS := fixture.create(t, fixture.nino, "S")
	other := fixture.create(t, fixture.nino, "Other")

	for _, space := range []string{spaceS, other} {
		fileID := fixture.addFile(t, space, fixture.nino, "ready")

		fixture.inScope(t, space, "", func(ctx context.Context, tx *sqlx.Tx) error {
			var place string

			if err := tx.GetContext(ctx, &place,
				`INSERT INTO probe.places (name) VALUES ('Balcony') RETURNING id;`); err != nil {
				return err //nolint:wrapcheck // the test reads the error as it is.
			}

			_, err := tx.ExecContext(ctx,
				`INSERT INTO probe.photos (place_id, file_id) VALUES ($1, $2);`, place, fileID)

			return err //nolint:wrapcheck // the test reads the error as it is.
		})
	}

	response := fixture.remove(t, fixture.nino, spaceS)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())

	_, err := fixture.database.ExecContext(t.Context(),
		`DELETE FROM public.workspaces WHERE id = $1;`, other)
	require.NoError(t, err)

	for _, space := range []string{spaceS, other} {
		for _, table := range []string{"probe.places", "probe.photos", "public.workspace_files"} {
			assert.Zero(t, fixture.countIn(t, table, "workspace_id", space, ""), table)
		}
	}
}

// WSREMOVE-SC-011: A table that names the workspace with no action fails the removal,
// and the removal deletes nothing and queues nothing.
func TestAReferenceWithNoActionFailsTheWholeRemoval(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	fixture.instance.Migrate(t, "stale", os.DirFS("testdata/stale"))

	spaceS := fixture.create(t, fixture.nino, "S")
	fixture.addSetting(t, spaceS)
	fixture.addFile(t, spaceS, fixture.nino, "ready")

	_, err := fixture.database.ExecContext(t.Context(),
		`INSERT INTO stale.leftovers (workspace_id) VALUES ($1);`, spaceS)
	require.NoError(t, err)

	tasks := fixture.countTasks(t)

	failed := fixture.remove(t, fixture.nino, spaceS)
	require.Equal(t, http.StatusInternalServerError, failed.Code, failed.Body.String())
	assert.Equal(t, typeInternal, decode(t, failed)[problemType])

	read := fixture.call(t, fixture.nino, http.MethodGet, "/workspaces/"+spaceS, nil, "")
	assert.Equal(t, http.StatusOK, read.Code)
	assert.Equal(
		t,
		1,
		fixture.countIn(t, "public.plugin_workspace_settings", "workspace_id", spaceS, ""),
	)
	assert.Equal(t, 1, fixture.countIn(t, "public.workspace_files", "workspace_id", spaceS, ""))
	assert.Equal(t, 1, fixture.countIn(t, "stale.leftovers", "workspace_id", spaceS, ""))
	assert.Equal(t, tasks, fixture.countTasks(t))
}

// WSREMOVE-SC-012: The removal leaves every record of the user.
func TestTheRemovalLeavesTheRecordsOfTheUser(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	spaceS := fixture.create(t, fixture.nino, "S")

	fixture.execIn(t, "", fixture.nino.id,
		`INSERT INTO public.plugin_user_settings (plugin_id) VALUES ($1);`, probePluginID)
	fixture.execIn(t, "", fixture.nino.id,
		`INSERT INTO public.plugin_user_secrets (plugin_id, name, ciphertext)
		VALUES ($1, 'token', 'sealed');`, probePluginID)
	fixture.execIn(t, "", fixture.nino.id,
		`INSERT INTO public.user_files (plugin_id, kind, name, media_type, size)
		VALUES ($1, 'avatar', 'me.webp', 'image/webp', 10);`, probePluginID)
	fixture.selectInChat(t, fixture.nino, 4001, spaceS)

	response := fixture.remove(t, fixture.nino, spaceS)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())

	userTables := []string{
		"public.plugin_user_settings",
		"public.plugin_user_secrets",
		"public.user_files",
		"public.telegram_chats",
	}

	for _, table := range userTables {
		var count int

		fixture.inScope(t, "", fixture.nino.id, func(ctx context.Context, tx *sqlx.Tx) error {
			return tx.GetContext(ctx, &count, `SELECT count(*) FROM `+table+`;`)
		})

		assert.Equal(t, 1, count, table)
	}
}

// queueFileTasks queues a task of each type for the file, owned by the workspace.
func (f *workspaceFixture) queueFileTasks(
	t *testing.T,
	workspaceID string,
	fileID string,
	types ...string,
) {
	t.Helper()

	tx, err := f.database.BeginTxx(t.Context(), nil)
	require.NoError(t, err)

	defer func() { _ = tx.Rollback() }()

	for _, taskType := range types {
		require.NoError(t, repository.NewOutboxTask(tx).Enqueue(t.Context(), model.OutboxDraft{
			Type:    taskType,
			Payload: map[string]string{"file_id": fileID},
			Owner:   &model.OutboxOwner{Scope: model.OutboxScopeWorkspace, ID: workspaceID},
			Delay:   time.Hour,
		}))
	}

	require.NoError(t, tx.Commit())
}

// queuedTask is a task of the outbox as SC-013 reads it.
type queuedTask struct {
	Type       string          `db:"type"`
	Payload    json.RawMessage `db:"payload"`
	OwnerScope *string         `db:"owner_scope"`
	RunAfter   time.Time       `db:"run_after"`
}

// tasksOwnedBy reads every task of the outbox that the workspace owns.
func (f *workspaceFixture) tasksOwnedBy(t *testing.T, workspaceID string) []queuedTask {
	t.Helper()

	var tasks []queuedTask

	require.NoError(t, f.database.SelectContext(t.Context(), &tasks,
		`SELECT type, payload, owner_scope, run_after FROM public.outbox_tasks
		WHERE owner_id = $1 ORDER BY type;`, workspaceID))

	return tasks
}

// WSREMOVE-SC-013: The removal queues one purge of each file, owned by the workspace,
// after the life of an upload address and the margin of a check. It drops every other
// task of a file of the workspace.
func TestTheRemovalQueuesThePurgeOfEachFile(t *testing.T) {
	t.Parallel()

	const wait = 16 * time.Minute

	fixture := workspaceUnderTest(t)
	spaceS := fixture.create(t, fixture.nino, "S")
	ready := fixture.addFile(t, spaceS, fixture.nino, "ready")
	pending := fixture.addFile(t, spaceS, fixture.nino, "pending")

	fixture.queueFileTasks(t, spaceS, pending, file.TaskCheck, file.TaskExpire)
	fixture.queueFileTasks(t, fixture.h, pending, file.TaskCheck)

	before := time.Now()

	response := fixture.remove(t, fixture.nino, spaceS)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())

	after := time.Now()
	purged := map[string]string{}

	for _, task := range fixture.tasksOwnedBy(t, spaceS) {
		require.Equal(t, file.TaskPurge, task.Type)
		require.NotNil(t, task.OwnerScope)
		assert.Equal(t, string(model.OutboxScopeWorkspace), *task.OwnerScope)
		assert.WithinRange(t, task.RunAfter, before.Add(wait).Add(-time.Second), after.Add(wait))

		var payload struct {
			FileID    string `json:"file_id"`
			ObjectKey string `json:"object_key"`
		}

		require.NoError(t, json.Unmarshal(task.Payload, &payload))

		purged[payload.FileID] = payload.ObjectKey
	}

	keyOf := func(fileID string) string {
		return "plugins/" + probePluginID + "/workspaces/" + spaceS + "/" + fileID + "/"
	}

	assert.Equal(t, map[string]string{ready: keyOf(ready), pending: keyOf(pending)}, purged)
	assert.Len(t, fixture.tasksOwnedBy(t, fixture.h), 1)
}

// WSREMOVE-SC-017: The removal answers before any call to the file store or the
// secret store, so a store that fails every call does not fail it.
func TestTheRemovalCallsNoStore(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	spaceS := fixture.create(t, fixture.nino, "S")
	fixture.addFile(t, spaceS, fixture.nino, "ready")
	fixture.addSecret(t, spaceS)

	fixture.notifications.cipher.mu.Lock()
	ciphered := len(fixture.notifications.cipher.keys)
	fixture.notifications.cipher.mu.Unlock()

	response := fixture.remove(t, fixture.nino, spaceS)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())

	assert.Zero(t, fixture.objects.calls.Load())

	fixture.notifications.cipher.mu.Lock()
	defer fixture.notifications.cipher.mu.Unlock()

	assert.Len(t, fixture.notifications.cipher.keys, ciphered)
}

// WSREMOVE-SC-022: The removal clears the selection of a chat, and the next command
// acts in the other workspace of the person.
func TestTheChatOfARemovedWorkspaceActsInTheOtherOne(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	admin := fixture.administrator(t)
	spaceW := fixture.create(t, fixture.gio, "W")

	const chatID = 4003

	fixture.selectInChat(t, fixture.gio, chatID, fixture.h)

	response := fixture.remove(t, admin, fixture.h)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())

	var selected sql.NullString

	fixture.inScope(t, "", fixture.gio.id, func(ctx context.Context, tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &selected,
			`SELECT selected_workspace_id FROM public.telegram_chats WHERE chat_id = $1;`, chatID)
	})
	assert.False(t, selected.Valid)

	ctx := pluginapi.ContextWithActingUser(t.Context(), fixture.gio.id)

	acting, role, err := workspace.NewChatSelection(fixture.database).Acting(ctx, chatID)
	require.NoError(t, err)
	assert.Equal(t, spaceW, acting)
	assert.Equal(t, pluginapi.RoleManager, role)
}

// WSREMOVE-SC-023: A read of a plugin route of the workspace answers not-found right
// after the removal.
func TestAReadAfterTheRemovalAnswersNotFound(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	admin := fixture.administrator(t)

	read := fixture.call(t, fixture.beka, http.MethodGet, fixture.notesOfProbe(), nil, "")
	require.Equal(t, http.StatusOK, read.Code, read.Body.String())

	response := fixture.remove(t, admin, fixture.h)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())

	again := fixture.call(t, fixture.beka, http.MethodGet, fixture.notesOfProbe(), nil, "")
	require.Equal(t, http.StatusNotFound, again.Code, again.Body.String())
	assert.Equal(t, typeNotFound, decode(t, again)[problemType])
}
