package file

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/filestore"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// errKindAbsent reports a file whose plugin declares its kind no more. The task waits
// for a load of the plugin, so the file loses nothing.
var errKindAbsent = errors.New("file: the file kind is not registered")

// scopeOf answers a context that acts as the owner of the file of the task, so the
// policy of its table shows the row.
func scopeOf(ctx context.Context, task model.FileTask) context.Context {
	if task.Scope == pluginapi.FileScopeUser {
		return pluginapi.ContextWithActingUser(ctx, task.OwnerID)
	}

	return pluginapi.ContextWithActingWorkspace(ctx, task.OwnerID)
}

// process checks the content of the upload against the kind, and makes the file
// ready or failed. A ready image gets its variants, and its upload goes away.
func (w *Worker) process(ctx context.Context, task model.FileTask) error {
	file, err := w.find(ctx, task)
	if err != nil || file == nil || isSettled(file) {
		return err
	}

	held, found := w.kinds.Get(file.PluginID, file.Kind)
	if !found {
		return fmt.Errorf("%w: %s", errKindAbsent, file.Kind)
	}

	if err = w.startProcessing(ctx, task); err != nil {
		return err
	}

	head, err := w.objects.ReadRange(ctx, task.ObjectKey+uploadObject, sniffLength)
	if errors.Is(err, filestore.ErrObjectAbsent) {
		return w.fail(ctx, task, pluginapi.FileFailureBytesAbsent, "")
	}

	if err != nil {
		return fmt.Errorf("reading the start of the upload: %w", err)
	}

	mediaType, accepted := sniff(head, held.Kind.MediaTypes)
	if !accepted {
		return w.fail(ctx, task, pluginapi.FileFailureMediaType, mediaType)
	}

	if !held.Kind.Image {
		return w.ready(ctx, file, model.FileReady{MediaType: mediaType}, false)
	}

	return w.processImage(ctx, task, file, held, mediaType)
}

func (w *Worker) startProcessing(ctx context.Context, task model.FileTask) error {
	return w.inScope(ctx, func(files *repository.File, _ *repository.FileTask) error {
		if _, err := files.StartProcessing(ctx, task.Scope, task.FileID); err != nil {
			return fmt.Errorf("moving the file to processing: %w", err)
		}

		return nil
	})
}

func (w *Worker) processImage(
	ctx context.Context,
	task model.FileTask,
	file *model.File,
	held registry.FileKind,
	mediaType string,
) error {
	key := keyOf(file)

	data, err := w.readUpload(ctx, key, held.Kind.MaxSize)
	if err != nil {
		return err
	}

	variants, err := variantsOf(data, mediaType)
	if errors.Is(err, errDamaged) {
		return w.fail(ctx, task, pluginapi.FileFailureDamaged, mediaType)
	}

	if err != nil {
		return err
	}

	for object, content := range map[string][]byte{
		thumbnailObject: variants.thumbnail,
		displayObject:   variants.display,
	} {
		if err = w.objects.Write(ctx, key+object, content, webpType); err != nil {
			return fmt.Errorf("writing a variant: %w", err)
		}
	}

	return w.ready(ctx, file, model.FileReady{
		MediaType: mediaType,
		Width:     &variants.width,
		Height:    &variants.height,
	}, true)
}

func (w *Worker) readUpload(ctx context.Context, key string, maxSize int64) ([]byte, error) {
	body, err := w.objects.Read(ctx, key+uploadObject)
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
func (w *Worker) check(ctx context.Context, task model.FileTask) error {
	file, err := w.find(ctx, task)
	if err != nil || file == nil {
		return err
	}

	switch file.State {
	case pluginapi.FileStatePending:
		return w.checkPending(ctx, task)
	case pluginapi.FileStateReady:
		return w.checkReady(ctx, task, file)
	case pluginapi.FileStateProcessing, pluginapi.FileStateFailed:
		return nil
	default:
		return nil
	}
}

func (w *Worker) checkPending(ctx context.Context, task model.FileTask) error {
	present, err := w.uploadPresent(ctx, task)
	if err != nil {
		return err
	}

	if !present {
		return w.fail(ctx, task, pluginapi.FileFailureBytesAbsent, "")
	}

	return w.enqueue(ctx, draftOf(task, model.FileTaskProcess))
}

func (w *Worker) checkReady(ctx context.Context, task model.FileTask, file *model.File) error {
	held, found := w.kinds.Get(file.PluginID, file.Kind)
	if !found || !held.Kind.Image {
		return nil
	}

	present, err := w.uploadPresent(ctx, task)
	if err != nil || !present {
		return err
	}

	return w.enqueue(ctx, draftOf(task, model.FileTaskDiscardUpload))
}

func (w *Worker) uploadPresent(ctx context.Context, task model.FileTask) (bool, error) {
	_, err := w.objects.Head(ctx, task.ObjectKey+uploadObject)

	switch {
	case errors.Is(err, filestore.ErrObjectAbsent):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("reading the facts of the upload: %w", err)
	default:
		return true, nil
	}
}

func (w *Worker) enqueue(ctx context.Context, draft model.FileTaskDraft) error {
	return w.inScope(ctx, func(_ *repository.File, tasks *repository.FileTask) error {
		if err := tasks.Enqueue(ctx, draft); err != nil {
			return fmt.Errorf("queuing the task %s: %w", draft.Action, err)
		}

		return nil
	})
}

func (w *Worker) discardUpload(ctx context.Context, task model.FileTask) error {
	if err := w.objects.Delete(ctx, task.ObjectKey+uploadObject); err != nil {
		return fmt.Errorf("removing the upload: %w", err)
	}

	return nil
}

// expire removes an unclaimed file and queues the purge of its bytes in one
// transaction. A referenced file takes a claim and stays.
func (w *Worker) expire(ctx context.Context, task model.FileTask) error {
	return w.inScope(ctx, func(files *repository.File, tasks *repository.FileTask) error {
		removed, err := files.Expire(ctx, task.Scope, task.FileID)
		if err != nil {
			return fmt.Errorf("expiring the file: %w", err)
		}

		if !removed {
			return nil
		}

		if err = tasks.Enqueue(ctx, draftOf(task, model.FileTaskPurge)); err != nil {
			return fmt.Errorf("queuing the purge: %w", err)
		}

		return nil
	})
}

func (w *Worker) purge(ctx context.Context, task model.FileTask) error {
	if err := w.objects.DeletePrefix(ctx, task.ObjectKey); err != nil {
		return fmt.Errorf("purging the objects of a file: %w", err)
	}

	return nil
}

func (w *Worker) find(ctx context.Context, task model.FileTask) (*model.File, error) {
	var file *model.File

	err := w.inScope(ctx, func(files *repository.File, _ *repository.FileTask) error {
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

func (w *Worker) fail(
	ctx context.Context,
	task model.FileTask,
	failure pluginapi.FileFailure,
	mediaType string,
) error {
	return w.inScope(ctx, func(files *repository.File, _ *repository.FileTask) error {
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
func (w *Worker) ready(
	ctx context.Context,
	file *model.File,
	facts model.FileReady,
	discard bool,
) error {
	return w.inScope(ctx, func(files *repository.File, tasks *repository.FileTask) error {
		moved, err := files.MarkReady(ctx, file.Scope, file.ID, facts)
		if err != nil {
			return fmt.Errorf("marking the file ready: %w", err)
		}

		if !moved || !discard {
			return nil
		}

		if err = tasks.Enqueue(ctx, taskOf(file, model.FileTaskDiscardUpload, 0)); err != nil {
			return fmt.Errorf("queuing the removal of the upload: %w", err)
		}

		return nil
	})
}

func (w *Worker) inScope(
	ctx context.Context,
	fn func(files *repository.File, tasks *repository.FileTask) error,
) error {
	err := database.WithScopeTx(ctx, w.db, func(tx *sqlx.Tx) error {
		return fn(repository.NewFile(tx), repository.NewFileTask(tx))
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

// draftOf answers a due task of the action for the file of the task.
func draftOf(task model.FileTask, action model.FileTaskAction) model.FileTaskDraft {
	return model.FileTaskDraft{
		FileID:    task.FileID,
		Scope:     task.Scope,
		OwnerID:   task.OwnerID,
		ObjectKey: task.ObjectKey,
		Action:    action,
	}
}
