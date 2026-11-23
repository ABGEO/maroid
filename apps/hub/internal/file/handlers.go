package file

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/gen2brain/h265/heic"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/filestore"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/outbox"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// The types of the tasks of the files in the outbox.
const (
	TaskProcess       = "file.process"
	TaskCheck         = "file.check"
	TaskDiscardUpload = "file.discard_upload"
	TaskExpire        = "file.expire"
	TaskPurge         = "file.purge"
)

const fixtureEdge = 16

var errNoOwner = errors.New("file: the task names no owner")

// Objects is the part of the file store that the handlers call.
type Objects interface {
	Ping(ctx context.Context) error
	Head(ctx context.Context, key string) (filestore.Object, error)
	ReadRange(ctx context.Context, key string, limit int64) ([]byte, error)
	Read(ctx context.Context, key string) (io.ReadCloser, error)
	Write(ctx context.Context, key string, data []byte, contentType string) error
	Delete(ctx context.Context, key string) error
	DeletePrefix(ctx context.Context, prefix string) error
}

// Handlers runs the tasks of the files: it processes an upload, checks an upload that
// no completion reported, removes an upload that the variants replace, expires an
// unclaimed file, and purges the bytes of a deleted one.
type Handlers struct {
	logger  *slog.Logger
	db      *sqlx.DB
	objects Objects
	kinds   *registry.FileKindRegistry
}

// NewHandlers creates the Handlers over the database, the file store, and the file
// kinds.
func NewHandlers(
	logger *slog.Logger,
	db *sqlx.DB,
	objects Objects,
	kinds *registry.FileKindRegistry,
) *Handlers {
	return &Handlers{
		logger:  logger.With(slog.String("component", "file")),
		db:      db,
		objects: objects,
		kinds:   kinds,
	}
}

// All answers one handler of the outbox for each type of the files.
func (h *Handlers) All() []outbox.Handler {
	return []outbox.Handler{
		preparedHandler{
			handler: handler{taskType: TaskProcess, run: h.processGuarded},
			prepare: h.prepare,
		},
		handler{taskType: TaskCheck, run: h.check},
		handler{taskType: TaskDiscardUpload, run: h.discardUpload},
		handler{taskType: TaskExpire, run: h.expire},
		handler{taskType: TaskPurge, run: h.purge},
	}
}

// prepare checks that the file store answers and that each codec decodes and encodes
// an image, so a broken build fails at the start and not at the first upload.
func (h *Handlers) prepare() error {
	if err := h.objects.Ping(context.Background()); err != nil {
		return fmt.Errorf("reaching the file store: %w", err)
	}

	if err := checkCodecs(); err != nil {
		return fmt.Errorf("checking the codecs: %w", err)
	}

	return nil
}

// processGuarded processes the upload. A panic, as a decoder may raise on hostile
// bytes, fails the file as damaged.
func (h *Handlers) processGuarded(ctx context.Context, task fileTask) (err error) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}

		h.logger.ErrorContext(
			ctx,
			"the processing of a file panicked",
			slog.String("file_id", task.FileID),
			slog.Any("panic", recovered),
			slog.String("stack", string(debug.Stack())),
		)

		err = h.fail(context.WithoutCancel(ctx), task, pluginapi.FileFailureDamaged, "")
	}()

	return h.process(ctx, task)
}

// fileTask is a task of the outbox for one file, read from its payload and its owner.
type fileTask struct {
	FileID    string
	ObjectKey string
	Scope     pluginapi.FileScope
	OwnerID   string
}

// taskPayload holds the identifiers of the file, and no content of it.
type taskPayload struct {
	FileID    string `json:"file_id"`
	ObjectKey string `json:"object_key"`
}

func fileTaskOf(task outbox.Task) (fileTask, error) {
	owner, found := task.Owner()
	if !found {
		return fileTask{}, errNoOwner
	}

	var payload taskPayload
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return fileTask{}, fmt.Errorf("decoding the payload: %w", err)
	}

	return fileTask{
		FileID:    payload.FileID,
		ObjectKey: payload.ObjectKey,
		Scope:     pluginapi.FileScope(owner.Scope),
		OwnerID:   owner.ID,
	}, nil
}

// draftFor answers a task of the type for the file, due after the delay. A process
// task holds the file as its dedupe key, so one waits for a file at a time.
func draftFor(task fileTask, taskType string, delay time.Duration) model.OutboxDraft {
	draft := model.OutboxDraft{
		Type:    taskType,
		Payload: taskPayload{FileID: task.FileID, ObjectKey: task.ObjectKey},
		Owner:   &model.OutboxOwner{Scope: model.OutboxScope(task.Scope), ID: task.OwnerID},
		Delay:   delay,
	}

	if taskType == TaskProcess {
		draft.DedupeKey = TaskProcess + ":" + task.FileID
	}

	return draft
}

// handler serves one type of the files through one function of the Handlers.
type handler struct {
	taskType string
	run      func(context.Context, fileTask) error
}

func (a handler) Type() string { return a.taskType }

func (a handler) Handle(ctx context.Context, task outbox.Task) error {
	held, err := fileTaskOf(task)
	if err != nil {
		return err
	}

	return a.run(ctx, held)
}

// preparedHandler is a handler with a check before the worker starts.
type preparedHandler struct {
	handler

	prepare func() error
}

func (p preparedHandler) Prepare() error { return p.prepare() }

// checkCodecs runs a small PNG and a small HEIC through the pipeline.
func checkCodecs() error {
	fixture := image.NewNRGBA(image.Rect(0, 0, fixtureEdge, fixtureEdge))

	var pngBytes, heicBytes bytes.Buffer

	if err := png.Encode(&pngBytes, fixture); err != nil {
		return fmt.Errorf("encoding the PNG fixture: %w", err)
	}

	if err := heic.Encode(&heicBytes, fixture); err != nil {
		return fmt.Errorf("encoding the HEIC fixture: %w", err)
	}

	for mediaType, data := range map[string][]byte{pngType: pngBytes.Bytes(), heicType: heicBytes.Bytes()} {
		if _, err := variantsOf(data, mediaType); err != nil {
			return fmt.Errorf("deriving the variants of the %s fixture: %w", mediaType, err)
		}
	}

	return nil
}
