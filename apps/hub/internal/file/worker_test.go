package file_test

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/gen2brain/h265/heic"
	"github.com/jmoiron/sqlx"
	"github.com/mcuadros/go-defaults"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/file"
	"github.com/abgeo/maroid/apps/hub/internal/filestore"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/outbox"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	uploadObject    = "upload"
	thumbnailObject = "thumbnail.jpg"
	displayObject   = "display.jpg"
)

// textured paints a pattern that a JPEG cannot squeeze into a few bytes.
func textured(width, height int) *image.NRGBA {
	return filled(width, height, func(x, y int) color.NRGBA {
		return color.NRGBA{R: channel(x * 7), G: channel(y * 13), B: channel(x + y), A: 255}
	})
}

func channel(value int) uint8 {
	return uint8(value % 256) //nolint:gosec // the remainder fits a byte
}

// uploaded starts an upload of the data, sends the bytes to the upload address, and
// answers the file.
func (b *storeBench) uploaded(
	t *testing.T,
	kind, name, mediaType string,
	data []byte,
) pluginapi.File {
	t.Helper()

	started, err := b.store.Start(b.in(t), probePlugin, file.StartInput{
		Kind: kind, Name: name, MediaType: mediaType, Size: int64(len(data)),
	})
	require.NoError(t, err)

	b.put(t, started.Upload, data)

	return started.File
}

func (b *storeBench) put(t *testing.T, upload filestore.Upload, data []byte) {
	t.Helper()

	request, err := http.NewRequestWithContext(
		t.Context(),
		upload.Method,
		upload.URL,
		bytes.NewReader(data),
	)
	require.NoError(t, err)

	for name, value := range upload.Headers {
		request.Header.Set(name, value)
	}

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusOK, response.StatusCode)
}

func (b *storeBench) cycle(t *testing.T) {
	t.Helper()

	require.NoError(t, b.worker.Cycle(t.Context()))
}

// due makes the task of the file with the action due now, as a later clock would.
func (b *storeBench) due(t *testing.T, fileID, taskType string) {
	t.Helper()

	result, err := b.db.ExecContext(t.Context(),
		`UPDATE public.outbox_tasks SET run_after = now()
		WHERE payload ->> 'file_id' = $1 AND type = $2;`,
		fileID, taskType)
	require.NoError(t, err)

	count, err := result.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func (b *storeBench) actionsOf(t *testing.T, fileID string) []string {
	t.Helper()

	var actions []string

	require.NoError(t, b.db.Select(
		&actions,
		`SELECT type FROM public.outbox_tasks
		WHERE payload ->> 'file_id' = $1 AND run_after <= now() ORDER BY type;`,
		fileID,
	))

	return actions
}

func (b *storeBench) workspaceKey(id string) string {
	return objectsOf(id, pluginapi.FileScopeWorkspace, b.space)
}

// holds reports whether the file store holds the object.
func (b *storeBench) holds(t *testing.T, key string) bool {
	t.Helper()

	_, err := b.objects.Head(t.Context(), key)
	if err != nil {
		require.ErrorIs(t, err, filestore.ErrObjectAbsent)

		return false
	}

	return true
}

func (b *storeBench) failure(
	t *testing.T,
	id string,
) (pluginapi.FileState, pluginapi.FileFailure, string) {
	t.Helper()

	got := b.get(t, probePlugin, id)

	return got.State, got.Failure, got.MediaType
}

// FILES-SC-010: One cycle after the completion makes a photo ready with its variants,
// and the upload goes away in the next cycle.
func TestACompletedPhotoBecomesReady(t *testing.T) {
	t.Parallel()

	bench := storeUnderTest(t)
	held := bench.uploaded(t, photoKind, "cat.jpg", jpegType, jpegWith(t, textured(1200, 900), 1))

	require.NoError(t, bench.store.Complete(bench.in(t), probePlugin, held.ID))
	bench.cycle(t)

	got := bench.get(t, probePlugin, held.ID)
	require.Equal(t, pluginapi.FileStateReady, got.State)
	require.Equal(t, [2]int{1200, 900}, [2]int{got.Width, got.Height})

	key := bench.workspaceKey(held.ID)
	require.True(t, bench.holds(t, key+thumbnailObject))
	require.True(t, bench.holds(t, key+displayObject))
	require.Equal(t, []string{file.TaskDiscardUpload}, bench.actionsOf(t, held.ID))

	bench.cycle(t)
	require.False(t, bench.holds(t, key+uploadObject))
}

// FILES-SC-011: A check that finds the upload of a file with no completion queues its
// processing, and the next cycle makes it ready.
func TestTheCheckProcessesAnUploadWithNoCompletion(t *testing.T) {
	t.Parallel()

	bench := storeUnderTest(t)
	held := bench.uploaded(t, photoKind, "cat.jpg", jpegType, jpegWith(t, textured(640, 480), 1))

	bench.due(t, held.ID, file.TaskCheck)
	bench.cycle(t)
	require.Equal(t, []string{file.TaskProcess}, bench.actionsOf(t, held.ID))

	bench.cycle(t)
	require.Equal(t, pluginapi.FileStateReady, bench.get(t, probePlugin, held.ID).State)
}

// FILES-SC-012: The bytes of a PDF under the type of a photo fail with media_type, and
// the file store holds no variant.
func TestTheContentDecidesTheMediaType(t *testing.T) {
	t.Parallel()

	bench := storeUnderTest(t)
	held := bench.uploaded(
		t,
		photoKind,
		"cat.jpg",
		jpegType,
		[]byte("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n1 0 obj\n"),
	)

	require.NoError(t, bench.store.Complete(bench.in(t), probePlugin, held.ID))
	bench.cycle(t)

	state, failure, mediaType := bench.failure(t, held.ID)
	require.Equal(t, pluginapi.FileStateFailed, state)
	require.Equal(t, pluginapi.FileFailureMediaType, failure)
	require.Equal(t, pdfType, mediaType)

	key := bench.workspaceKey(held.ID)
	require.False(t, bench.holds(t, key+thumbnailObject))
	require.False(t, bench.holds(t, key+displayObject))
}

// FILES-SC-013: A cut JPEG fails as damaged, and a file that never received bytes
// fails as bytes_absent at its check.
func TestBrokenAndAbsentBytesFail(t *testing.T) {
	t.Parallel()

	bench := storeUnderTest(t)
	whole := jpegWith(t, textured(1200, 900), 1)
	require.Greater(t, len(whole), 4096)

	cut := bench.uploaded(t, photoKind, "portrait.jpg", jpegType, whole[:4096])
	require.NoError(t, bench.store.Complete(bench.in(t), probePlugin, cut.ID))

	never := bench.start(t, probePlugin, photoKind, "cat.jpg", jpegType)
	bench.due(t, never.ID, file.TaskCheck)

	bench.cycle(t)

	state, failure, _ := bench.failure(t, cut.ID)
	require.Equal(t, pluginapi.FileStateFailed, state)
	require.Equal(t, pluginapi.FileFailureDamaged, failure)

	state, failure, _ = bench.failure(t, never.ID)
	require.Equal(t, pluginapi.FileStateFailed, state)
	require.Equal(t, pluginapi.FileFailureBytesAbsent, failure)
}

// FILES-SC-017: A HEIC photo becomes ready with its type and its size.
func TestAHEICPhotoBecomesReady(t *testing.T) {
	t.Parallel()

	bench := storeUnderTest(t)

	var photo bytes.Buffer

	require.NoError(t, heic.Encode(&photo, textured(1024, 768)))

	held := bench.uploaded(t, heicPhotoKind, "phone.heic", heicType, photo.Bytes())
	require.NoError(t, bench.store.Complete(bench.in(t), probePlugin, held.ID))
	bench.cycle(t)

	got := bench.get(t, probePlugin, held.ID)
	require.Equal(t, pluginapi.FileStateReady, got.State)
	require.Equal(t, heicType, got.MediaType)
	require.Equal(t, [2]int{1024, 768}, [2]int{got.Width, got.Height})
}

// FILES-SC-019: An upload that a client sends again after the photo became ready goes
// away at the check, and the variants stay.
func TestARepeatedUploadGoesAway(t *testing.T) {
	t.Parallel()

	bench := storeUnderTest(t)
	data := jpegWith(t, textured(640, 480), 1)

	started, err := bench.store.Start(bench.in(t), probePlugin, file.StartInput{
		Kind: photoKind, Name: "cat.jpg", MediaType: jpegType, Size: int64(len(data)),
	})
	require.NoError(t, err)

	bench.put(t, started.Upload, data)
	require.NoError(t, bench.store.Complete(bench.in(t), probePlugin, started.File.ID))
	bench.cycle(t)
	bench.cycle(t)

	key := bench.workspaceKey(started.File.ID)
	require.False(t, bench.holds(t, key+uploadObject))

	bench.put(t, started.Upload, data)
	bench.due(t, started.File.ID, file.TaskCheck)
	bench.cycle(t)
	bench.cycle(t)

	require.False(t, bench.holds(t, key+uploadObject))
	require.True(t, bench.holds(t, key+thumbnailObject))
	require.True(t, bench.holds(t, key+displayObject))
}

// FILES-SC-020: A document becomes ready as uploaded, and answers its original alone.
func TestADocumentKeepsItsUpload(t *testing.T) {
	t.Parallel()

	bench := storeUnderTest(t)
	content := []byte("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n1 0 obj\n")
	held := bench.uploaded(t, paperKind, "paper.pdf", pdfType, content)

	require.NoError(t, bench.store.Complete(bench.in(t), probePlugin, held.ID))
	bench.cycle(t)

	got := bench.get(t, probePlugin, held.ID)
	require.Equal(t, pluginapi.FileStateReady, got.State)
	require.Len(t, got.Addresses, 1)
	require.Equal(t, content, read(t, got.Addresses[pluginapi.FileVariantOriginal].URL).body)
	require.Empty(t, bench.actionsOf(t, held.ID))
}

// FILES-SC-022: An unclaimed file goes away after 24 hours, and so do its bytes.
func TestAnUnclaimedFileExpires(t *testing.T) {
	t.Parallel()

	bench := storeUnderTest(t)
	held := bench.uploaded(t, photoKind, "cat.jpg", jpegType, jpegWith(t, textured(640, 480), 1))

	require.NoError(t, bench.store.Complete(bench.in(t), probePlugin, held.ID))
	bench.cycle(t)
	bench.cycle(t)

	bench.due(t, held.ID, file.TaskExpire)
	bench.cycle(t)
	bench.cycle(t)

	listed, err := bench.store.Get(bench.in(t), probePlugin, held.ID)
	require.NoError(t, err)
	require.Empty(t, listed)

	key := bench.workspaceKey(held.ID)
	for _, object := range []string{uploadObject, thumbnailObject, displayObject} {
		require.False(t, bench.holds(t, key+object), object)
	}
}

// FILES-SC-043: No variant of a photo carries the EXIF of the upload.
func TestTheVariantsOfAPhotoCarryNoEXIF(t *testing.T) {
	t.Parallel()

	bench := storeUnderTest(t)
	upload := jpegWith(t, textured(640, 480), 1)
	require.Contains(t, string(upload), cameraMaker)

	held := bench.uploaded(t, photoKind, "gps.jpg", jpegType, upload)
	require.NoError(t, bench.store.Complete(bench.in(t), probePlugin, held.ID))
	bench.cycle(t)

	for variant, address := range bench.get(t, probePlugin, held.ID).Addresses {
		body := read(t, address.URL).body
		require.NotContains(t, string(body), cameraMaker, variant)
		require.NotContains(t, string(body), "Exif", variant)
	}
}

// FILES-DD-005: A task that the file store fails stays, with the error and a later
// time, and the error names no key.
func TestAFailedTaskWaitsForARetry(t *testing.T) {
	t.Parallel()

	bench := storeUnderTest(t)
	held := bench.uploaded(t, photoKind, "cat.jpg", jpegType, jpegWith(t, textured(640, 480), 1))

	require.NoError(t, bench.store.Complete(bench.in(t), probePlugin, held.ID))
	bench.server.Stop(t)
	bench.cycle(t)

	task := bench.taskOf(t, held.ID, file.TaskProcess)
	require.Equal(t, 1, task.Attempts)
	require.Nil(t, task.LockedUntil)
	require.NotNil(t, task.LastError)
	require.NotContains(t, *task.LastError, held.ID)
	require.True(t, task.RunAfter.After(time.Now()))
}

// FILES-DD-007: The worker starts only when the file store answers and the codecs run.
func TestPrepareChecksTheFileStoreAndTheCodecs(t *testing.T) {
	t.Parallel()

	bench := storeUnderTest(t)
	require.NoError(t, bench.worker.Prepare())

	bench.server.Stop(t)
	require.Error(t, bench.worker.Prepare())
}

// hostileObjects is the file store of the test, whose full read and purge panic as a
// decoder may on hostile bytes.
type hostileObjects struct{ *filestore.Client }

func (hostileObjects) Read(context.Context, string) (io.ReadCloser, error) {
	panic("a hostile upload")
}

func (hostileObjects) DeletePrefix(context.Context, string) error {
	panic("a hostile purge")
}

// stallingObjects is the file store of the test, whose full read never answers.
type stallingObjects struct{ *filestore.Client }

func (stallingObjects) Read(ctx context.Context, _ string) (io.ReadCloser, error) {
	<-ctx.Done()

	return nil, fmt.Errorf("stalling: %w", ctx.Err())
}

func (b *storeBench) workerOver(
	objects file.Objects,
	kinds *registry.FileKindRegistry,
	cfg config.Outbox,
	options ...outbox.Option,
) *outbox.Worker {
	return workerOf(b.db, objects, kinds, cfg, options...)
}

// workerOf answers the worker of the outbox with the handlers of the files alone.
func workerOf(
	db *sqlx.DB,
	objects file.Objects,
	kinds *registry.FileKindRegistry,
	cfg config.Outbox,
	options ...outbox.Option,
) *outbox.Worker {
	logger := slog.New(slog.DiscardHandler)
	handlers := outbox.NewRegistry()

	if err := handlers.Register(file.NewHandlers(logger, db, objects, kinds).All()...); err != nil {
		panic(err)
	}

	return outbox.NewWorker(logger, db, handlers, cfg, options...)
}

// outboxConfig answers the configuration of the outbox with its defaults.
func outboxConfig() config.Outbox {
	var cfg config.Outbox

	defaults.SetDefaults(&cfg)

	return cfg
}

func (b *storeBench) taskOf(
	t *testing.T,
	fileID string,
	taskType string,
) model.OutboxTask {
	t.Helper()

	var task model.OutboxTask

	require.NoError(t, b.db.Get(&task, `SELECT id, type, payload, owner_scope, owner_id,
		dedupe_key, run_after, locked_until, attempts, last_error, created_at
		FROM public.outbox_tasks WHERE payload ->> 'file_id' = $1 AND type = $2;`, fileID, taskType))

	return task
}

// FILES-DD-005: A panic of a process task fails the file as damaged, and a panic of
// any other task leaves it for a retry. The worker goes on.
func TestAPanicOfATaskStaysInTheTask(t *testing.T) {
	t.Parallel()

	bench := storeUnderTest(t)
	worker := bench.workerOver(
		hostileObjects{bench.objects},
		kindsUnderTest(t),
		outboxConfig(),
	)

	held := bench.uploaded(t, photoKind, "cat.jpg", jpegType, jpegWith(t, textured(640, 480), 1))
	require.NoError(t, bench.store.Complete(bench.in(t), probePlugin, held.ID))
	require.NoError(t, worker.Cycle(t.Context()))

	state, failure, _ := bench.failure(t, held.ID)
	require.Equal(t, pluginapi.FileStateFailed, state)
	require.Equal(t, pluginapi.FileFailureDamaged, failure)

	require.NoError(t, bench.deleteFiles(t, probePlugin, held.ID))
	require.NoError(t, worker.Cycle(t.Context()))

	purge := bench.taskOf(t, held.ID, file.TaskPurge)
	require.Equal(t, 1, purge.Attempts)
	require.Nil(t, purge.LockedUntil)
	require.NotNil(t, purge.LastError)
	require.Contains(t, *purge.LastError, "panicked")
}

// FILES-DD-005: A task that passes its deadline stops, and waits for a retry before
// its lease lapses.
func TestATaskStopsAtItsDeadline(t *testing.T) {
	t.Parallel()

	bench := storeUnderTest(t)
	worker := bench.workerOver(
		stallingObjects{bench.objects},
		kindsUnderTest(t),
		outboxConfig(),
		outbox.WithTaskTimeout(200*time.Millisecond),
	)

	held := bench.uploaded(t, photoKind, "cat.jpg", jpegType, jpegWith(t, textured(640, 480), 1))
	require.NoError(t, bench.store.Complete(bench.in(t), probePlugin, held.ID))

	began := time.Now()

	require.NoError(t, worker.Cycle(t.Context()))
	require.Less(t, time.Since(began), 5*time.Second)

	task := bench.taskOf(t, held.ID, file.TaskProcess)
	require.Equal(t, 1, task.Attempts)
	require.Nil(t, task.LockedUntil)
	require.NotNil(t, task.LastError)
	require.Contains(t, *task.LastError, "deadline")
}

// FILES-DD-005: A lease that fills every slot leases again at once, so a queue longer
// than the concurrency never waits for the next poll.
func TestAFullLeaseLeasesAgainAtOnce(t *testing.T) {
	t.Parallel()

	bench := storeUnderTest(t)
	cfg := outboxConfig()
	cfg.PollInterval = time.Hour
	cfg.Concurrency = 1

	content := []byte("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n1 0 obj\n")
	ids := make([]string, 0, 3)

	for range 3 {
		held := bench.uploaded(t, paperKind, "paper.pdf", pdfType, content)
		require.NoError(t, bench.store.Complete(bench.in(t), probePlugin, held.ID))
		ids = append(ids, held.ID)
	}

	ctx, cancel := context.WithCancel(t.Context())
	worker := bench.workerOver(bench.objects, kindsUnderTest(t), cfg)
	stopped := make(chan error, 1)

	go func() { stopped <- worker.Start(ctx) }()

	require.Eventually(t, func() bool {
		for _, id := range ids {
			if bench.get(t, probePlugin, id).State != pluginapi.FileStateReady {
				return false
			}
		}

		return true
	}, 10*time.Second, 100*time.Millisecond)

	cancel()
	require.NoError(t, <-stopped)
}

// FILES-DD-006: A file whose kind the plugin declares no more waits for a load of the
// plugin, and fails with media_type once it is 24 hours old.
func TestAFileOfAnAbsentKindFailsAtTheExpiry(t *testing.T) {
	t.Parallel()

	bench := storeUnderTest(t)
	worker := bench.workerOver(bench.objects, registry.NewFileKindRegistry(), outboxConfig())

	held := bench.uploaded(t, photoKind, "cat.jpg", jpegType, jpegWith(t, textured(640, 480), 1))
	require.NoError(t, bench.store.Complete(bench.in(t), probePlugin, held.ID))
	require.NoError(t, worker.Cycle(t.Context()))

	require.Equal(t, pluginapi.FileStatePending, bench.get(t, probePlugin, held.ID).State)
	require.Equal(t, 1, bench.taskOf(t, held.ID, file.TaskProcess).Attempts)

	require.NoError(t, database.WithScopeTx(bench.in(t), bench.db, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(t.Context(), `UPDATE public.workspace_files
			SET created_at = now() - interval '25 hours' WHERE id = $1;`, held.ID)
		require.NoError(t, err)

		return nil
	}))

	bench.due(t, held.ID, file.TaskProcess)
	require.NoError(t, worker.Cycle(t.Context()))

	state, failure, _ := bench.failure(t, held.ID)
	require.Equal(t, pluginapi.FileStateFailed, state)
	require.Equal(t, pluginapi.FileFailureMediaType, failure)
}

// FILES-SC-035: Twenty phone photos, one after the other, each reach ready within 30
// seconds of the completion at the 95th percentile, with concurrency 2 and a poll of
// 2 seconds.
func TestTwentyPhonePhotosReachReadyInTime(t *testing.T) {
	t.Parallel()

	const (
		uploads    = 20
		percentile = 95
		limit      = 30 * time.Second
	)

	bench := storeUnderTest(t)
	cfg := outboxConfig()
	require.Equal(t, 2, cfg.Concurrency)
	require.Equal(t, 2*time.Second, cfg.PollInterval)

	var photo bytes.Buffer

	require.NoError(t, heic.Encode(&photo, textured(4032, 3024)))

	ctx, cancel := context.WithCancel(t.Context())
	worker := bench.workerOver(bench.objects, kindsUnderTest(t), cfg)
	stopped := make(chan error, 1)

	go func() { stopped <- worker.Start(ctx) }()

	waits := make([]time.Duration, 0, uploads)

	for range uploads {
		held := bench.uploaded(t, heicPhotoKind, "phone.heic", heicType, photo.Bytes())
		require.NoError(t, bench.store.Complete(bench.in(t), probePlugin, held.ID))

		completed := time.Now()

		require.Eventually(t, func() bool {
			return bench.get(t, probePlugin, held.ID).State == pluginapi.FileStateReady
		}, 2*limit, 50*time.Millisecond)

		waits = append(waits, time.Since(completed))
	}

	cancel()
	require.NoError(t, <-stopped)

	slices.Sort(waits)
	require.Less(t, waits[uploads*percentile/100-1], limit, "the waits: %v", waits)
}
