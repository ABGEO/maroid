package handler_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/apps/hub/internal/file"
	"github.com/abgeo/maroid/apps/hub/internal/filestore"
	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/teststore"
	"github.com/abgeo/maroid/apps/hub/internal/timezone"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	photoKind       = "photo"
	paperKind       = "paper"
	photoUpload     = "photo.upload"
	paperWrite      = "paper.write"
	paperLimit      = 5 << 20
	memberFile      = "file"
	memberUpload    = "upload"
	memberMediaType = "media_type"
	memberSize      = "size"
	jpegType        = "image/jpeg"
	pdfType         = "application/pdf"
	statePending    = "pending"
	taskCheck       = file.TaskCheck
	taskExpire      = file.TaskExpire
	taskProcess     = file.TaskProcess
)

// fileBench holds workspace H with the routes of the files mounted over the file
// kinds photo and paper of the probe, and the RustFS of the test.
type fileBench struct {
	*workspaceFixture

	// setup keeps the router of the workspaces, for a test that creates a workspace.
	setup *workspaceFixture
	store *teststore.Server
}

// fileAuthorizer answers the permissions of the hub, and the permissions of the file
// kinds of the probe, which an editor holds.
func fileAuthorizer(t *testing.T) *authz.RoleAuthorizer {
	t.Helper()

	permissions, err := authz.NewPermissionRegistry()
	require.NoError(t, err)

	probe := pluginapi.ParsePluginID(probePluginID)
	require.NoError(t, permissions.Register(
		registry.PermissionEntry{
			Name: registry.PermissionName(probe, photoUpload), Lowest: pluginapi.RoleEditor,
		},
		registry.PermissionEntry{
			Name: registry.PermissionName(probe, paperWrite), Lowest: pluginapi.RoleEditor,
		},
	))

	return authz.NewRoleAuthorizer(permissions)
}

func probeFileKinds(t *testing.T) *registry.FileKindRegistry {
	t.Helper()

	probe := pluginapi.ParsePluginID(probePluginID)
	kinds := registry.NewFileKindRegistry()
	require.NoError(t, kinds.Register(
		registry.FileKind{
			PluginID: probePluginID,
			Kind: pluginapi.FileKind{
				Key:        photoKind,
				Scope:      pluginapi.FileScopeWorkspace,
				MediaTypes: []string{jpegType, "image/heic"},
				MaxSize:    pluginapi.MaxFileSize,
				Permission: photoUpload,
				Image:      true,
			},
			Permission: registry.PermissionName(probe, photoUpload),
		},
		registry.FileKind{
			PluginID: probePluginID,
			Kind: pluginapi.FileKind{
				Key:        paperKind,
				Scope:      pluginapi.FileScopeUser,
				MediaTypes: []string{pdfType},
				MaxSize:    paperLimit,
				Permission: paperWrite,
			},
			Permission: registry.PermissionName(probe, paperWrite),
		},
	))

	return kinds
}

func filesUnderTest(t *testing.T) *fileBench {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	fixture := workspaceUnderTest(t)
	server := teststore.Start(t)
	cfg := server.Config()
	access, _ := probeAccessOf(t, fixture.database)
	access.Authorizer = fileAuthorizer(t)

	router := baseRouter()
	handler.NewFile(
		slog.New(slog.DiscardHandler),
		workspaceVerifier(t, fixture.provider),
		auth.NewResolver(fixture.database, timezone.New(slog.New(slog.DiscardHandler))),
		noIdempotency{},
		access,
		file.NewStore(fixture.database, filestore.New(cfg), probeFileKinds(t), cfg.UploadTTL),
	).Register(router)

	mounted := *fixture
	mounted.router = router

	return &fileBench{workspaceFixture: &mounted, setup: fixture, store: server}
}

func (b *fileBench) filesOf(workspaceID string) string {
	return "/workspaces/" + workspaceID + "/plugins/" + probePluginID + "/files"
}

// start sends the start of an upload in the workspace as the person.
func (b *fileBench) start(
	t *testing.T,
	who person,
	workspaceID string,
	kind, name, mediaType string,
	size int64,
) *httptest.ResponseRecorder {
	t.Helper()

	return b.call(t, who, http.MethodPost, b.filesOf(workspaceID), map[string]any{
		"kind": kind, memberName: name, memberMediaType: mediaType, memberSize: size,
	}, "")
}

// started sends a start that the hub accepts, and answers the file.
func (b *fileBench) started(
	t *testing.T,
	who person,
	workspaceID string,
	kind, name, mediaType string,
	size int64,
) map[string]any {
	t.Helper()

	response := b.start(t, who, workspaceID, kind, name, mediaType, size)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	held, ok := decode(t, response)[memberFile].(map[string]any)
	require.True(t, ok)

	return held
}

// inScope runs fn in a transaction that acts as the person in the workspace, so the
// policies of the file tables show the rows of that scope.
func (b *fileBench) inScope(t *testing.T, who person, workspaceID string, fn func(*sqlx.Tx)) {
	t.Helper()

	tx, err := b.database.BeginTxx(t.Context(), nil)
	require.NoError(t, err)

	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(t.Context(),
		`SELECT set_config('app.user_id', $1, true), set_config('app.workspace_id', $2, true);`,
		who.id, workspaceID)
	require.NoError(t, err)

	fn(tx)
	require.NoError(t, tx.Commit())
}

// countFiles counts the files of the workspace and of the person.
func (b *fileBench) countFiles(t *testing.T, who person, workspaceID string) int {
	t.Helper()

	var count int

	b.inScope(t, who, workspaceID, func(tx *sqlx.Tx) {
		require.NoError(t, tx.Get(&count, `SELECT
			(SELECT count(*) FROM public.workspace_files) + (SELECT count(*) FROM public.user_files);`))
	})

	return count
}

func (b *fileBench) actionsOf(t *testing.T, fileID string) []string {
	t.Helper()

	var actions []string

	require.NoError(t, b.database.Select(
		&actions,
		`SELECT type FROM public.outbox_tasks WHERE payload ->> 'file_id' = $1 ORDER BY type;`,
		fileID,
	))

	return actions
}

// FILES-SC-004: An editor starts the upload of a photo, and the hub answers the
// pending file with an upload address. The row and its tasks check and expire exist.
func TestAnEditorStartsAnUpload(t *testing.T) {
	t.Parallel()

	bench := filesUnderTest(t)

	response := bench.start(t, bench.beka, bench.h, photoKind, "cat.jpg", jpegType, 3<<20)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	answer := decode(t, response)
	held, ok := answer[memberFile].(map[string]any)
	require.True(t, ok)

	id := stringOf(t, held[memberID])
	assert.Equal(t, statePending, held[memberState])
	assert.Equal(t, bench.beka.id, held["uploaded_by"])
	assert.Equal(t, false, held["claimed"])
	assert.NotContains(t, held, "addresses")
	assert.Equal(t,
		"https://hub.example.com"+bench.filesOf(bench.h)+"/"+id,
		response.Header().Get("Location"),
	)

	upload, ok := answer[memberUpload].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, http.MethodPut, upload["method"])
	assert.Contains(t, upload["url"], bench.store.Endpoint)
	assert.Equal(t, map[string]any{"Content-Type": jpegType}, upload["headers"])

	assert.Equal(t, 1, bench.countFiles(t, bench.beka, bench.h))
	assert.Equal(t, []string{taskCheck, taskExpire}, bench.actionsOf(t, id))
}

// RES-001: Every time of a file answers in UTC, as Zalando Z-169 demands.
func TestTheTimesOfAFileAnswerInUTC(t *testing.T) {
	t.Parallel()

	bench := filesUnderTest(t)

	response := bench.start(t, bench.beka, bench.h, paperKind, "lease.pdf", pdfType, 1<<20)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	answer := decode(t, response)
	held, ok := answer[memberFile].(map[string]any)
	require.True(t, ok)

	upload, ok := answer[memberUpload].(map[string]any)
	require.True(t, ok)
	assert.True(t, strings.HasSuffix(stringOf(t, held["uploaded_at"]), "Z"))
	assert.True(t, strings.HasSuffix(stringOf(t, upload["expires_at"]), "Z"))

	id := stringOf(t, held[memberID])
	bench.inScope(t, bench.beka, bench.h, func(tx *sqlx.Tx) {
		_, err := tx.ExecContext(t.Context(),
			`UPDATE public.user_files SET state = 'ready' WHERE id = $1;`, id)
		require.NoError(t, err)
	})

	response = bench.call(t, bench.beka, http.MethodGet, bench.filesOf(bench.h)+"/"+id, nil, "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	addresses, ok := decode(t, response)["addresses"].(map[string]any)
	require.True(t, ok)

	original, ok := addresses["original"].(map[string]any)
	require.True(t, ok)
	assert.True(t, strings.HasSuffix(stringOf(t, original["expires_at"]), "Z"))
}

// FILES-SC-005: A viewer does not hold the permission of the file kind, so the hub
// refuses the start and writes no row.
func TestAViewerCannotStartAnUpload(t *testing.T) {
	t.Parallel()

	bench := filesUnderTest(t)

	response := bench.start(t, bench.gio, bench.h, photoKind, "cat.jpg", jpegType, 3<<20)
	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())

	answer := decode(t, response)
	assert.Equal(t, "/problems/http/permission-denied", answer[memberType])
	assert.Equal(t, probePluginID+":"+photoUpload, answer["permission"])
	assert.Equal(t, roleEditor, answer["required_role"])
	assert.Zero(t, bench.countFiles(t, bench.beka, bench.h))
}

// FILES-SC-006: The file kind refuses a media type that it does not declare.
func TestAStartRefusesAMediaTypeOutsideTheKind(t *testing.T) {
	t.Parallel()

	bench := filesUnderTest(t)

	requireFieldFailure(t,
		bench.start(t, bench.beka, bench.h, photoKind, "lease.pdf", pdfType, 3<<20),
		"/media_type",
	)
	assert.Zero(t, bench.countFiles(t, bench.beka, bench.h))
}

// FILES-SC-007: The file kind refuses a size above its limit, and takes the limit.
func TestAStartRefusesASizeAboveTheKind(t *testing.T) {
	t.Parallel()

	bench := filesUnderTest(t)

	requireFieldFailure(t,
		bench.start(t, bench.beka, bench.h, paperKind, "lease.pdf", pdfType, paperLimit+1),
		"/size",
	)

	held := bench.started(t, bench.beka, bench.h, paperKind, "lease.pdf", pdfType, paperLimit)
	assert.Equal(t, statePending, held[memberState])
}

// FILES-SC-030: A file store that does not answer makes the start answer not-ready
// within 2 seconds, and the hub writes no row.
func TestAStartAnswersNotReadyWhenTheFileStoreStops(t *testing.T) {
	t.Parallel()

	bench := filesUnderTest(t)
	bench.store.Stop(t)

	started := time.Now()
	response := bench.start(t, bench.beka, bench.h, photoKind, "cat.jpg", jpegType, 3<<20)

	assert.Less(t, time.Since(started), 2*time.Second)
	require.Equal(t, http.StatusServiceUnavailable, response.Code, response.Body.String())

	answer := decode(t, response)
	assert.Equal(t, "/problems/hub/not-ready", answer[memberType])
	assert.Equal(t, []any{"file-store"}, answer["dependencies"])
	assert.Zero(t, bench.countFiles(t, bench.beka, bench.h))
}

// FILES-SC-028: A file of another workspace reads as absent, and so does an identifier
// that is not a UUID.
func TestAFileOfAnotherWorkspaceReadsAsAbsent(t *testing.T) {
	t.Parallel()

	bench := filesUnderTest(t)
	other := bench.setup.create(t, bench.ana, "B")
	bench.enable(t, other, probePluginID)

	held := bench.started(t, bench.ana, other, photoKind, "cat.jpg", jpegType, 3<<20)
	id := stringOf(t, held[memberID])

	response := bench.call(t, bench.ana, http.MethodGet, bench.filesOf(other)+"/"+id, nil, "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Equal(t, statePending, decode(t, response)[memberState])

	for _, path := range []string{bench.filesOf(bench.h) + "/" + id, bench.filesOf(bench.h) + "/not-a-uuid"} {
		response = bench.call(t, bench.ana, http.MethodGet, path, nil, "")
		require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
		assert.Equal(t, typeNotFound, decode(t, response)[memberType])
	}
}

// FILES-SC-010: The completion answers at once and queues the processing. A repeat
// on a file that is no longer pending queues nothing.
func TestTheCompletionQueuesTheProcessing(t *testing.T) {
	t.Parallel()

	bench := filesUnderTest(t)

	held := bench.started(t, bench.beka, bench.h, photoKind, "cat.jpg", jpegType, 3<<20)
	id := stringOf(t, held[memberID])
	completion := bench.filesOf(bench.h) + "/" + id + "/completions"

	response := bench.call(t, bench.beka, http.MethodPost, completion, nil, "")
	require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	assert.Equal(t, []string{taskCheck, taskExpire, taskProcess}, bench.actionsOf(t, id))

	bench.inScope(t, bench.beka, bench.h, func(tx *sqlx.Tx) {
		_, err := tx.ExecContext(t.Context(),
			`UPDATE public.workspace_files SET state = 'processing' WHERE id = $1;`, id)
		require.NoError(t, err)
	})

	response = bench.call(t, bench.beka, http.MethodPost, completion, nil, "")
	require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	assert.Equal(t, []string{taskCheck, taskExpire, taskProcess}, bench.actionsOf(t, id))

	response = bench.call(t, bench.gio, http.MethodPost, completion, nil, "")
	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())
}
