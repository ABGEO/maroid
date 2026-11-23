package file

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/filestore"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// errKindAbsent reports a file whose plugin declares its kind no more. The task waits
// for a load of the plugin until the file is as old as an unclaimed file can be.
var errKindAbsent = errors.New("file: the file kind is not registered")

// process checks the content of the upload against the kind, and makes the file
// ready or failed. A ready image gets its variants, and its upload goes away.
func (h *Handlers) process(ctx context.Context, task fileTask) error {
	file, err := h.find(ctx, task)
	if err != nil || file == nil || isSettled(file) {
		return err
	}

	held, found := h.kinds.Get(file.PluginID, file.Kind)
	if !found {
		return h.kindAbsent(ctx, task, file)
	}

	if err = h.startProcessing(ctx, task); err != nil {
		return err
	}

	head, err := h.objects.ReadRange(ctx, task.ObjectKey+uploadObject, sniffLength)
	if errors.Is(err, filestore.ErrObjectAbsent) {
		return h.fail(ctx, task, pluginapi.FileFailureBytesAbsent, "")
	}

	if err != nil {
		return fmt.Errorf("reading the start of the upload: %w", err)
	}

	mediaType, accepted := sniff(head, held.Kind.MediaTypes)
	if !accepted {
		return h.fail(ctx, task, pluginapi.FileFailureMediaType, mediaType)
	}

	if !held.Kind.Image {
		return h.ready(ctx, file, model.FileReady{MediaType: mediaType}, false)
	}

	return h.processImage(ctx, task, file, held, mediaType)
}

// kindAbsent leaves the task for a retry while a load of the plugin may still declare
// the kind, and fails the file once it is as old as an unclaimed file can be.
func (h *Handlers) kindAbsent(ctx context.Context, task fileTask, file *model.File) error {
	if time.Since(file.CreatedAt) < unclaimedLife {
		return fmt.Errorf("%w: %s", errKindAbsent, file.Kind)
	}

	return h.fail(ctx, task, pluginapi.FileFailureMediaType, "")
}

func (h *Handlers) startProcessing(ctx context.Context, task fileTask) error {
	return h.inScope(ctx, func(files *repository.File, _ *repository.OutboxTask) error {
		if _, err := files.StartProcessing(ctx, task.Scope, task.FileID); err != nil {
			return fmt.Errorf("moving the file to processing: %w", err)
		}

		return nil
	})
}

func (h *Handlers) processImage(
	ctx context.Context,
	task fileTask,
	file *model.File,
	held registry.FileKind,
	mediaType string,
) error {
	key := keyOf(file)

	data, err := h.readUpload(ctx, key, held.Kind.MaxSize)
	if err != nil {
		return err
	}

	variants, err := variantsOf(data, mediaType)
	if errors.Is(err, errDamaged) {
		return h.fail(ctx, task, pluginapi.FileFailureDamaged, mediaType)
	}

	if err != nil {
		return err
	}

	for object, content := range map[string][]byte{
		thumbnailObject: variants.thumbnail,
		displayObject:   variants.display,
	} {
		if err = h.objects.Write(ctx, key+object, content, jpegType); err != nil {
			return fmt.Errorf("writing a variant: %w", err)
		}
	}

	return h.ready(ctx, file, model.FileReady{
		MediaType: mediaType,
		Width:     &variants.width,
		Height:    &variants.height,
	}, true)
}

func (h *Handlers) readUpload(ctx context.Context, key string, maxSize int64) ([]byte, error) {
	body, err := h.objects.Read(ctx, key+uploadObject)
	if err != nil {
		return nil, fmt.Errorf("reading the upload: %w", err)
	}

	defer func() { _ = body.Close() }()

	data, err := io.ReadAll(io.LimitReader(body, maxSize+1))
	if err != nil {
		return nil, fmt.Errorf("reading the upload: %w", err)
	}

	return data, nil
}

// check processes a pending file whose upload exists, and fails one whose upload never
// came. It removes an upload that a client sent again after an image became ready.
func (h *Handlers) check(ctx context.Context, task fileTask) error {
	file, err := h.find(ctx, task)
	if err != nil || file == nil {
		return err
	}

	switch file.State {
	case pluginapi.FileStatePending:
		return h.checkPending(ctx, task)
	case pluginapi.FileStateReady:
		return h.checkReady(ctx, task, file)
	case pluginapi.FileStateProcessing, pluginapi.FileStateFailed:
		return nil
	default:
		return nil
	}
}

func (h *Handlers) checkPending(ctx context.Context, task fileTask) error {
	present, err := h.uploadPresent(ctx, task)
	if err != nil {
		return err
	}

	if !present {
		return h.fail(ctx, task, pluginapi.FileFailureBytesAbsent, "")
	}

	return h.inScope(ctx, func(_ *repository.File, tasks *repository.OutboxTask) error {
		if err := tasks.Enqueue(ctx, draftOf(task, TaskProcess)); err != nil {
			return fmt.Errorf("queuing the processing: %w", err)
		}

		return nil
	})
}

func (h *Handlers) checkReady(ctx context.Context, task fileTask, file *model.File) error {
	held, found := h.kinds.Get(file.PluginID, file.Kind)
	if !found || !held.Kind.Image {
		return nil
	}

	present, err := h.uploadPresent(ctx, task)
	if err != nil || !present {
		return err
	}

	return h.enqueue(ctx, draftOf(task, TaskDiscardUpload))
}

func (h *Handlers) uploadPresent(ctx context.Context, task fileTask) (bool, error) {
	_, err := h.objects.Head(ctx, task.ObjectKey+uploadObject)

	switch {
	case errors.Is(err, filestore.ErrObjectAbsent):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("reading the facts of the upload: %w", err)
	default:
		return true, nil
	}
}

func (h *Handlers) enqueue(ctx context.Context, draft model.OutboxDraft) error {
	return h.inScope(ctx, func(_ *repository.File, tasks *repository.OutboxTask) error {
		if err := tasks.Enqueue(ctx, draft); err != nil {
			return fmt.Errorf("queuing the task %s: %w", draft.Type, err)
		}

		return nil
	})
}

func (h *Handlers) discardUpload(ctx context.Context, task fileTask) error {
	if err := h.objects.Delete(ctx, task.ObjectKey+uploadObject); err != nil {
		return fmt.Errorf("removing the upload: %w", err)
	}

	return nil
}

// expire removes an unclaimed file and queues the purge of its bytes in one
// transaction. A referenced file takes a claim and stays.
func (h *Handlers) expire(ctx context.Context, task fileTask) error {
	return h.inScope(ctx, func(files *repository.File, tasks *repository.OutboxTask) error {
		removed, err := files.Expire(ctx, task.Scope, task.FileID)
		if err != nil {
			return fmt.Errorf("expiring the file: %w", err)
		}

		if !removed {
			return nil
		}

		if err = tasks.Enqueue(ctx, draftOf(task, TaskPurge)); err != nil {
			return fmt.Errorf("queuing the purge: %w", err)
		}

		return nil
	})
}

func (h *Handlers) purge(ctx context.Context, task fileTask) error {
	if err := h.objects.DeletePrefix(ctx, task.ObjectKey); err != nil {
		return fmt.Errorf("purging the objects of a file: %w", err)
	}

	return nil
}

func (h *Handlers) find(ctx context.Context, task fileTask) (*model.File, error) {
	var file *model.File

	err := h.inScope(ctx, func(files *repository.File, _ *repository.OutboxTask) error {
		var err error

		if file, err = files.Find(ctx, task.Scope, task.FileID); err != nil {
			return fmt.Errorf("finding the file: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading the file: %w", err)
	}

	return file, nil
}

func (h *Handlers) fail(
	ctx context.Context,
	task fileTask,
	failure pluginapi.FileFailure,
	mediaType string,
) error {
	return h.inScope(ctx, func(files *repository.File, _ *repository.OutboxTask) error {
		if _, err := files.MarkFailed(
			ctx,
			task.Scope,
			task.FileID,
			failure,
			mediaType,
		); err != nil {
			return fmt.Errorf("marking the file failed: %w", err)
		}

		return nil
	})
}

// ready marks the file ready, and queues the removal of its upload in the same
// transaction when the variants replace it.
func (h *Handlers) ready(
	ctx context.Context,
	file *model.File,
	facts model.FileReady,
	discard bool,
) error {
	return h.inScope(ctx, func(files *repository.File, tasks *repository.OutboxTask) error {
		moved, err := files.MarkReady(ctx, file.Scope, file.ID, facts)
		if err != nil {
			return fmt.Errorf("marking the file ready: %w", err)
		}

		if !moved || !discard {
			return nil
		}

		if err = tasks.Enqueue(ctx, taskOf(file, TaskDiscardUpload, 0)); err != nil {
			return fmt.Errorf("queuing the removal of the upload: %w", err)
		}

		return nil
	})
}

func (h *Handlers) inScope(
	ctx context.Context,
	fn func(files *repository.File, tasks *repository.OutboxTask) error,
) error {
	err := database.WithScopeTx(ctx, h.db, func(tx *sqlx.Tx) error {
		return fn(repository.NewFile(tx), repository.NewOutboxTask(tx))
	})
	if err != nil {
		return fmt.Errorf("acting as the owner of the file: %w", err)
	}

	return nil
}

// isSettled reports a file that no later task changes.
func isSettled(file *model.File) bool {
	return file.State == pluginapi.FileStateReady || file.State == pluginapi.FileStateFailed
}

// draftOf answers a due task of the type for the file of the task.
func draftOf(task fileTask, taskType string) model.OutboxDraft {
	return draftFor(task, taskType, 0)
}
