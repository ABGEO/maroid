package file

import (
	"context"
	"fmt"
	"mime"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/filestore"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	maxNameLength = 255
	// checkMargin keeps the check of an upload past the last moment that the upload
	// address still takes a body.
	checkMargin = time.Minute
	// unclaimedLife is the time from the start of an upload to the delete of an
	// unclaimed file.
	unclaimedLife = 24 * time.Hour
)

// ObjectStore is the part of the file store that the store calls.
type ObjectStore interface {
	Ping(ctx context.Context) error
	PresignPut(
		ctx context.Context,
		key string,
		size int64,
		contentType string,
	) (filestore.Upload, error)
	PresignGet(ctx context.Context, key string, attachmentName string) (filestore.Address, error)
}

// StartInput holds what a person declares at the start of an upload.
type StartInput struct {
	Kind      string
	Name      string
	MediaType string
	Size      int64
}

// Started holds a new pending file and the address that takes its bytes.
type Started struct {
	File   pluginapi.File
	Upload filestore.Upload
}

// Store starts, reads, claims, and deletes the files of every plugin in the acting scope.
type Store struct {
	db        *sqlx.DB
	objects   ObjectStore
	kinds     *registry.FileKindRegistry
	uploadTTL time.Duration
}

// NewStore creates a Store. uploadTTL is the life of an upload address.
func NewStore(
	db *sqlx.DB,
	objects ObjectStore,
	kinds *registry.FileKindRegistry,
	uploadTTL time.Duration,
) *Store {
	return &Store{db: db, objects: objects, kinds: kinds, uploadTTL: uploadTTL}
}

// Start checks the declaration against the file kind, writes a pending file with its
// check and its expiry, and answers the address that takes the bytes. It writes
// nothing when the file store does not answer.
func (s *Store) Start(ctx context.Context, pluginID string, input StartInput) (Started, error) {
	held, found := s.kinds.Get(pluginID, input.Kind)
	if !found {
		return Started{}, &FieldError{
			Pointer: "/kind",
			Detail:  "The plugin accepts no file of this kind.",
		}
	}

	mediaType, err := accepted(held.Kind, input)
	if err != nil {
		return Started{}, err
	}

	if !ownerActs(ctx, held.Kind.Scope) {
		return Started{}, pluginapi.ErrFileScopeAbsent
	}

	if err = s.objects.Ping(ctx); err != nil {
		return Started{}, fmt.Errorf("%w: %w", errs.ErrFileStoreUnavailable, err)
	}

	var created *model.File

	err = database.WithScopeTx(ctx, s.db, func(tx *sqlx.Tx) error {
		created, err = repository.NewFile(tx).Create(ctx, held.Kind.Scope, model.FileDraft{
			PluginID:  pluginID,
			Kind:      held.Kind.Key,
			Name:      input.Name,
			MediaType: mediaType,
			Size:      input.Size,
		})
		if err != nil {
			return fmt.Errorf("writing the file: %w", err)
		}

		return repository.NewFileTask(tx).Enqueue(ctx,
			taskOf(created, model.FileTaskCheck, s.uploadTTL+checkMargin),
			taskOf(created, model.FileTaskExpire, unclaimedLife),
		)
	})
	if err != nil {
		return Started{}, fmt.Errorf("starting an upload: %w", err)
	}

	upload, err := s.objects.PresignPut(ctx, keyOf(created)+uploadObject, created.Size, mediaType)
	if err != nil {
		return Started{}, fmt.Errorf("signing the upload: %w", err)
	}

	view, err := s.view(ctx, created)
	if err != nil {
		return Started{}, err
	}

	return Started{File: view, Upload: upload}, nil
}

// Complete queues the processing of a pending file of the plugin. A file in any other
// state, or one whose processing waits, takes no task, so a repeat is safe.
func (s *Store) Complete(ctx context.Context, pluginID string, id string) error {
	if !isUUID(id) {
		return pluginapi.ErrFileAbsent
	}

	err := database.WithScopeTx(ctx, s.db, func(tx *sqlx.Tx) error {
		listed, err := repository.NewFile(tx).List(ctx, pluginID, []string{id})
		if err != nil {
			return fmt.Errorf("reading the file: %w", err)
		}

		if len(listed) == 0 {
			return pluginapi.ErrFileAbsent
		}

		if listed[0].State != pluginapi.FileStatePending {
			return nil
		}

		return repository.NewFileTask(tx).
			EnqueueOnce(ctx, taskOf(&listed[0], model.FileTaskProcess, 0))
	})
	if err != nil {
		return fmt.Errorf("completing an upload: %w", err)
	}

	return nil
}

// Get answers the files of the plugin under the identifiers, with a read address for
// each variant of a ready file. It omits an identifier that the plugin and the acting
// scope do not hold, and answers pluginapi.ErrTooManyFiles past
// pluginapi.MaxFilesPerGet identifiers.
func (s *Store) Get(ctx context.Context, pluginID string, ids ...string) ([]pluginapi.File, error) {
	if len(ids) > pluginapi.MaxFilesPerGet {
		return nil, pluginapi.ErrTooManyFiles
	}

	valid := slices.DeleteFunc(slices.Clone(ids), func(id string) bool { return !isUUID(id) })

	var listed []model.File

	err := database.WithScopeTx(ctx, s.db, func(tx *sqlx.Tx) error {
		var err error

		listed, err = repository.NewFile(tx).List(ctx, pluginID, valid)
		if err != nil {
			return fmt.Errorf("reading the files: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading the files: %w", err)
	}

	files := make([]pluginapi.File, 0, len(listed))

	for index := range listed {
		view, viewErr := s.view(ctx, &listed[index])
		if viewErr != nil {
			return nil, viewErr
		}

		files = append(files, view)
	}

	return files, nil
}

// Kind answers one file kind of the plugin, so a route checks its permission before
// it starts or answers anything.
func (s *Store) Kind(pluginID string, key string) (registry.FileKind, bool) {
	return s.kinds.Get(pluginID, key)
}

// Claim marks the file of the plugin as used, so the expiry keeps it.
func (s *Store) Claim(ctx context.Context, pluginID string, id string) error {
	if !isUUID(id) {
		return pluginapi.ErrFileAbsent
	}

	err := database.WithScopeTx(ctx, s.db, func(tx *sqlx.Tx) error {
		claimed, err := repository.NewFile(tx).Claim(ctx, pluginID, id)
		if err != nil {
			return fmt.Errorf("claiming the file: %w", err)
		}

		if !claimed {
			return pluginapi.ErrFileAbsent
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("claiming a file: %w", err)
	}

	return nil
}

// Delete removes the files of the plugin and queues the purge of their bytes through
// the transaction of the plugin, so both commit with the rows that the plugin removed
// first. It answers pluginapi.ErrFileReferenced while a row references a file, and
// pluginapi.ErrFileAbsent for an identifier that the plugin and the acting scope do
// not hold.
func (s *Store) Delete(ctx context.Context, tx *sqlx.Tx, pluginID string, ids ...string) error {
	files := repository.NewFile(tx)
	tasks := repository.NewFileTask(tx)

	for _, id := range ids {
		if !isUUID(id) {
			return fmt.Errorf("deleting the file %q: %w", id, pluginapi.ErrFileAbsent)
		}

		deleted, err := files.Delete(ctx, pluginID, id)
		if err != nil {
			return fmt.Errorf("deleting the file %q: %w", id, err)
		}

		if deleted == nil {
			return fmt.Errorf("deleting the file %q: %w", id, pluginapi.ErrFileAbsent)
		}

		if err = tasks.Enqueue(ctx, taskOf(deleted, model.FileTaskPurge, 0)); err != nil {
			return fmt.Errorf("queuing the purge of the file %q: %w", id, err)
		}
	}

	return nil
}

// view answers the facts of the file, with a read address for each variant when it is
// ready. A ready image holds a width, and every other ready file holds its original.
func (s *Store) view(ctx context.Context, file *model.File) (pluginapi.File, error) {
	view := pluginapi.File{
		ID:         file.ID,
		Kind:       file.Kind,
		State:      file.State,
		Name:       file.Name,
		MediaType:  file.MediaType,
		Size:       file.Size,
		UploadedBy: file.UploadedBy,
		UploadedAt: file.CreatedAt,
		Claimed:    file.ClaimedAt != nil,
	}

	if file.Failure != nil {
		view.Failure = *file.Failure
	}

	if file.Width != nil && file.Height != nil {
		view.Width, view.Height = *file.Width, *file.Height
	}

	if file.State != pluginapi.FileStateReady {
		return view, nil
	}

	variants := map[pluginapi.FileVariant]string{pluginapi.FileVariantOriginal: uploadObject}
	attachment := file.Name

	if file.Width != nil {
		variants = map[pluginapi.FileVariant]string{
			pluginapi.FileVariantThumbnail: thumbnailObject,
			pluginapi.FileVariantDisplay:   displayObject,
		}
		attachment = ""
	}

	view.Addresses = make(map[pluginapi.FileVariant]pluginapi.ReadAddress, len(variants))

	for variant, object := range variants {
		address, err := s.objects.PresignGet(ctx, keyOf(file)+object, attachment)
		if err != nil {
			return pluginapi.File{}, fmt.Errorf("signing the read of a file: %w", err)
		}

		view.Addresses[variant] = pluginapi.ReadAddress{
			URL:       address.URL,
			ExpiresAt: address.ExpiresAt,
		}
	}

	return view, nil
}

// accepted checks the declaration against the kind, and answers the media type in its
// plain form.
func accepted(kind pluginapi.FileKind, input StartInput) (string, error) {
	if length := utf8.RuneCountInString(input.Name); length == 0 || length > maxNameLength {
		return "", &FieldError{
			Pointer: "/name",
			Detail:  "The name of the file holds 1 to 255 characters.",
		}
	}

	mediaType, _, err := mime.ParseMediaType(strings.ToLower(input.MediaType))
	if err != nil || !slices.Contains(kind.MediaTypes, mediaType) {
		return "", &FieldError{
			Pointer: "/media_type",
			Detail:  "The file kind does not accept this media type.",
		}
	}

	switch {
	case input.Size < 1:
		return "", &FieldError{Pointer: "/size", Detail: "The file holds no byte."}
	case input.Size > kind.MaxSize:
		return "", &FieldError{
			Pointer: "/size",
			Detail:  "The file is larger than the file kind accepts.",
		}
	}

	return mediaType, nil
}

func ownerActs(ctx context.Context, scope pluginapi.FileScope) bool {
	if scope == pluginapi.FileScopeUser {
		return pluginapi.ActingUserFromContext(ctx) != ""
	}

	return pluginapi.ActingWorkspaceFromContext(ctx) != ""
}

func keyOf(file *model.File) string {
	return objectKey(file.PluginID, file.Scope, file.OwnerID, file.ID)
}

func taskOf(
	file *model.File,
	action model.FileTaskAction,
	delay time.Duration,
) model.FileTaskDraft {
	return model.FileTaskDraft{
		FileID:    file.ID,
		Scope:     file.Scope,
		OwnerID:   file.OwnerID,
		ObjectKey: keyOf(file),
		Action:    action,
		Delay:     delay,
	}
}

func isUUID(value string) bool {
	return uuid.Validate(value) == nil
}
