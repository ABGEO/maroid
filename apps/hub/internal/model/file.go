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

// FileReady holds the facts of a ready file that the content gave. Width and Height
// are nil for a file that is not an image.
type FileReady struct {
	MediaType string
	Width     *int
	Height    *int
}
