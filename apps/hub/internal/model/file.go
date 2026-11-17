package model

import (
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// File is one file of one plugin. OwnerID holds the workspace or the user that the
// scope names. UploadedBy is the owner for a file of the user scope.
type File struct {
	ID         string                 `db:"id"`
	Scope      pluginapi.FileScope    `db:"scope"`
	OwnerID    string                 `db:"owner_id"`
	PluginID   string                 `db:"plugin_id"`
	Kind       string                 `db:"kind"`
	State      pluginapi.FileState    `db:"state"`
	Failure    *pluginapi.FileFailure `db:"failure"`
	Name       string                 `db:"name"`
	MediaType  string                 `db:"media_type"`
	Size       int64                  `db:"size"`
	Width      *int                   `db:"width"`
	Height     *int                   `db:"height"`
	UploadedBy string                 `db:"uploaded_by"`
	ClaimedAt  *time.Time             `db:"claimed_at"`
	CreatedAt  time.Time              `db:"created_at"`
	UpdatedAt  time.Time              `db:"updated_at"`
}

// FileDraft holds what the start of an upload declares. The session gives the owner
// and the uploader.
type FileDraft struct {
	PluginID  string
	Kind      string
	Name      string
	MediaType string
	Size      int64
}

// FileTaskAction names the work that one task of the outbox does.
type FileTaskAction string

// The actions of a task.
const (
	FileTaskProcess       FileTaskAction = "process"
	FileTaskCheck         FileTaskAction = "check"
	FileTaskDiscardUpload FileTaskAction = "discard_upload"
	FileTaskExpire        FileTaskAction = "expire"
	FileTaskPurge         FileTaskAction = "purge"
)

// FileTask is one task of the outbox. It names the owner of its file, so the worker
// runs it in the scope of that owner.
type FileTask struct {
	ID          string              `db:"id"`
	FileID      string              `db:"file_id"`
	Scope       pluginapi.FileScope `db:"scope"`
	OwnerID     string              `db:"owner_id"`
	ObjectKey   string              `db:"object_key"`
	Action      FileTaskAction      `db:"action"`
	RunAfter    time.Time           `db:"run_after"`
	LockedUntil *time.Time          `db:"locked_until"`
	Attempts    int                 `db:"attempts"`
	LastError   *string             `db:"last_error"`
	CreatedAt   time.Time           `db:"created_at"`
}

// FileTaskDraft holds a task to enqueue. Delay holds the time from the commit to the
// first run.
type FileTaskDraft struct {
	FileID    string
	Scope     pluginapi.FileScope
	OwnerID   string
	ObjectKey string
	Action    FileTaskAction
	Delay     time.Duration
}

// FileReady holds the facts of a ready file that the content gave. Width and Height
// are nil for a file that is not an image.
type FileReady struct {
	MediaType string
	Width     *int
	Height    *int
}
