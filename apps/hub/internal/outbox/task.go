package outbox

import (
	"context"

	"github.com/abgeo/maroid/apps/hub/internal/model"
)

// Task is one leased task of the outbox.
type Task = model.OutboxTask

// Draft is a task to enqueue.
type Draft = model.OutboxDraft

// Owner is the workspace or the user that a task acts for.
type Owner = model.OutboxOwner

// Handler runs the tasks of one type. A task runs at least once, so Handle is
// idempotent. The context acts as the owner of the task, when it names one.
type Handler interface {
	Type() string
	Handle(ctx context.Context, task Task) error
}

// Preparer is a Handler that checks its dependencies before the worker starts.
type Preparer interface {
	Prepare() error
}
