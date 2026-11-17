package file_test

import (
	"context"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/file"
	"github.com/abgeo/maroid/apps/hub/internal/filestore"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/teststore"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	probePlugin   = "dev.maroid.probe"
	secondPlugin  = "dev.maroid.second"
	photoKind     = "photo"
	heicPhotoKind = "phone-photo"
	paperKind     = "paper"
	jpegType      = "image/jpeg"
	pdfType       = "application/pdf"
	webpType      = "image/webp"
	photoSize     = 3 << 20
)

// storeBench holds a Store over a real database and the RustFS of the test, and a
// context that acts as a manager of one workspace.
type storeBench struct {
	store   *file.Store
	objects *filestore.Client
	server  *teststore.Server
	worker  *file.Worker
	db      *sqlx.DB
	user    string
	space   string
}

// in answers a context that acts as the manager in the workspace.
func (b *storeBench) in(t *testing.T) context.Context {
	t.Helper()

	return pluginapi.ContextWithActingWorkspace(
		pluginapi.ContextWithActingUser(t.Context(), b.user),
		b.space,
	)
}

func kindsUnderTest(t *testing.T) *registry.FileKindRegistry {
	t.Helper()

	kinds := registry.NewFileKindRegistry()
	declare := func(pluginID, key string, scope pluginapi.FileScope, mediaType string, image bool) {
		require.NoError(t, kinds.Register(registry.FileKind{
			PluginID: pluginID,
			Kind: pluginapi.FileKind{
				Key: key, Scope: scope, MediaTypes: []string{mediaType},
				MaxSize: pluginapi.MaxFileSize, Permission: "files.write", Image: image,
			},
			Permission: pluginID + ":files.write",
		}))
	}

	declare(probePlugin, photoKind, pluginapi.FileScopeWorkspace, jpegType, true)
	declare(probePlugin, paperKind, pluginapi.FileScopeUser, pdfType, false)
	declare(probePlugin, heicPhotoKind, pluginapi.FileScopeWorkspace, heicType, true)
	declare(secondPlugin, photoKind, pluginapi.FileScopeWorkspace, jpegType, true)

	return kinds
}

func storeUnderTest(t *testing.T) *storeBench {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	var user, workspace string

	require.NoError(t, instance.DB.Get(&user,
		`INSERT INTO public.users (first_name) VALUES ('Ana') RETURNING id;`))
	require.NoError(t, instance.DB.Get(&workspace,
		`INSERT INTO public.workspaces (name) VALUES ('A') RETURNING id;`))
	_, err = instance.DB.Exec(`INSERT INTO public.workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'manager');`, workspace, user)
	require.NoError(t, err)

	server := teststore.Start(t)
	cfg := server.Config()
	objects := filestore.New(cfg)

	kinds := kindsUnderTest(t)

	return &storeBench{
		store:   file.NewStore(instance.DB, objects, kinds, cfg.UploadTTL),
		objects: objects,
		server:  server,
		worker:  file.NewWorker(slog.New(slog.DiscardHandler), instance.DB, objects, kinds, cfg),
		db:      instance.DB,
		user:    user,
		space:   workspace,
	}
}

func (b *storeBench) start(t *testing.T, pluginID, kind, name, mediaType string) pluginapi.File {
	t.Helper()

	started, err := b.store.Start(b.in(t), pluginID, file.StartInput{
		Kind: kind, Name: name, MediaType: mediaType, Size: photoSize,
	})
	require.NoError(t, err)

	return started.File
}

// advance moves the file to processing, and to ready with the facts when ready is set.
func (b *storeBench) advance(
	t *testing.T,
	scope pluginapi.FileScope,
	id string,
	ready *model.FileReady,
) {
	t.Helper()

	ctx := b.in(t)

	require.NoError(t, database.WithScopeTx(ctx, b.db, func(tx *sqlx.Tx) error {
		files := repository.NewFile(tx)

		moved, err := files.StartProcessing(ctx, scope, id)
		require.NoError(t, err)
		require.True(t, moved)

		if ready != nil {
			moved, err = files.MarkReady(ctx, scope, id, *ready)
			require.NoError(t, err)
			require.True(t, moved)
		}

		return nil
	}))
}

func (b *storeBench) get(t *testing.T, pluginID, id string) pluginapi.File {
	t.Helper()

	listed, err := b.store.Get(b.in(t), pluginID, id)
	require.NoError(t, err)
	require.Len(t, listed, 1)

	return listed[0]
}

// fetched holds the status, the type, and the body of one read.
type fetched struct {
	status      int
	contentType string
	body        []byte
}

func read(t *testing.T, address string) fetched {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address, nil)
	require.NoError(t, err)

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)

	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	return fetched{
		status:      response.StatusCode,
		contentType: response.Header.Get("Content-Type"),
		body:        body,
	}
}

func objectsOf(id string, scope pluginapi.FileScope, owner string) string {
	return "plugins/" + probePlugin + "/" + string(scope) + "s/" + owner + "/" + id + "/"
}

// FILES-SC-041: A plugin reaches no file of another plugin, though both act in one
// workspace.
func TestAPluginReachesNoFileOfAnotherPlugin(t *testing.T) {
	t.Parallel()

	bench := storeUnderTest(t)
	held := bench.start(t, secondPlugin, photoKind, "cat.jpg", jpegType)
	probe := pluginapi.ParsePluginID(probePlugin)

	listed, err := bench.store.ReadFiles(bench.in(t), probe, held.ID)
	require.NoError(t, err)
	require.Empty(t, listed)

	require.ErrorIs(t, bench.store.ClaimFile(bench.in(t), probe, held.ID), pluginapi.ErrFileAbsent)
	require.ErrorIs(t, bench.store.DeleteFile(bench.in(t), probe, held.ID), pluginapi.ErrFileAbsent)

	require.Equal(t, held.ID, bench.get(t, secondPlugin, held.ID).ID)
}

// FILES-SC-029: A file in processing holds no address.
func TestAFileInProcessingHoldsNoAddress(t *testing.T) {
	t.Parallel()

	bench := storeUnderTest(t)
	held := bench.start(t, probePlugin, photoKind, "cat.jpg", jpegType)

	bench.advance(t, pluginapi.FileScopeWorkspace, held.ID, nil)

	got := bench.get(t, probePlugin, held.ID)
	require.Equal(t, pluginapi.FileStateProcessing, got.State)
	require.Empty(t, got.Addresses)
}

// FILES-SC-027: A ready image holds an address for thumbnail and for display, and
// each address answers its WebP.
func TestAReadyImageAnswersItsVariants(t *testing.T) {
	t.Parallel()

	bench := storeUnderTest(t)
	held := bench.start(t, probePlugin, photoKind, "cat.jpg", jpegType)
	width, height := 4032, 3024

	bench.advance(t, pluginapi.FileScopeWorkspace, held.ID,
		&model.FileReady{MediaType: jpegType, Width: &width, Height: &height})

	key := objectsOf(
		held.ID,
		pluginapi.FileScopeWorkspace,
		bench.space,
	)
	for _, object := range []string{"thumbnail.webp", "display.webp"} {
		require.NoError(t, bench.objects.Write(t.Context(), key+object, []byte(object), webpType))
	}

	got := bench.get(t, probePlugin, held.ID)
	require.Equal(t, pluginapi.FileStateReady, got.State)
	require.Equal(t, width, got.Width)
	require.Equal(t, height, got.Height)
	require.Len(t, got.Addresses, 2)

	for variant, object := range map[pluginapi.FileVariant]string{
		pluginapi.FileVariantThumbnail: "thumbnail.webp",
		pluginapi.FileVariantDisplay:   "display.webp",
	} {
		answer := read(t, got.Addresses[variant].URL)
		require.Equal(t, http.StatusOK, answer.status, variant)
		require.Equal(t, webpType, answer.contentType)
		require.Equal(t, []byte(object), answer.body)
	}
}

// FILES-SC-020: A ready file that is no image holds the variant original alone, and
// its address answers the bytes of the upload.
func TestAReadyDocumentAnswersItsOriginal(t *testing.T) {
	t.Parallel()

	bench := storeUnderTest(t)
	held := bench.start(t, probePlugin, paperKind, "lease.pdf", pdfType)
	content := []byte("%PDF-1.7")

	bench.advance(t, pluginapi.FileScopeUser, held.ID, &model.FileReady{MediaType: pdfType})

	key := objectsOf(held.ID, pluginapi.FileScopeUser, bench.user)
	require.NoError(t, bench.objects.Write(t.Context(), key+"upload", content, pdfType))

	got := bench.get(t, probePlugin, held.ID)
	require.Len(t, got.Addresses, 1)

	answer := read(t, got.Addresses[pluginapi.FileVariantOriginal].URL)
	require.Equal(t, http.StatusOK, answer.status)
	require.Equal(t, content, answer.body)
}
