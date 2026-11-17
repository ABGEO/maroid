package pluginapi

import (
	"context"
	"errors"
	"time"
)

// FileScope names the owner of every file of a kind.
type FileScope string

const (
	// FileScopeWorkspace gives each file to the acting workspace, for every member.
	FileScopeWorkspace FileScope = "workspace"
	// FileScopeUser gives each file to the acting user alone.
	FileScopeUser FileScope = "user"
)

// MaxFileSize is the largest size limit that a file kind declares, in bytes.
const MaxFileSize int64 = 25 << 20

// FileKind declares what a plugin accepts for one purpose. Key is unique within the
// plugin. Permission is a local name that the plugin declares, as for a route.
type FileKind struct {
	Key        string
	Scope      FileScope
	MediaTypes []string
	MaxSize    int64
	Permission string
	// Image makes the hub derive the variants. Every media type is then one of
	// image/jpeg, image/png, image/webp, image/heic, or image/heif.
	Image bool
}

// FilePlugin is a plugin that accepts files.
type FilePlugin interface {
	FileKinds() ([]FileKind, error)
}

// FileState names where a file stands between its upload and its use.
type FileState string

// The states of a file.
const (
	FileStatePending    FileState = "pending"
	FileStateProcessing FileState = "processing"
	FileStateReady      FileState = "ready"
	FileStateFailed     FileState = "failed"
)

// FileFailure names the reason of a failed file.
type FileFailure string

// The reasons of a failed file.
const (
	FileFailureMediaType   FileFailure = "media_type"
	FileFailureDamaged     FileFailure = "damaged"
	FileFailureBytesAbsent FileFailure = "bytes_absent"
)

// FileVariant names one form of a ready file that a read address serves.
type FileVariant string

// The variants of a file. An image holds the thumbnail and the display image.
// Any other file holds the original.
const (
	FileVariantOriginal  FileVariant = "original"
	FileVariantThumbnail FileVariant = "thumbnail"
	FileVariantDisplay   FileVariant = "display"
)

// ReadAddress is a signed address of one variant, and the time it stops working.
type ReadAddress struct {
	URL       string
	ExpiresAt time.Time
}

// File holds the facts of one file. Failure is empty unless State is failed.
// Width and Height are zero for a file that is not an image. Addresses holds a
// read address for each variant of a ready file, and nothing for any other state.
type File struct {
	ID         string
	Kind       string
	State      FileState
	Failure    FileFailure
	Name       string
	MediaType  string
	Size       int64
	Width      int
	Height     int
	UploadedBy string
	UploadedAt time.Time
	Claimed    bool
	Addresses  map[FileVariant]ReadAddress
}

var (
	// ErrFileAbsent reports that the plugin and the acting scope hold no file under
	// the identifier.
	ErrFileAbsent = errors.New("file: absent")
	// ErrFileReferenced reports that a row still references the file.
	ErrFileReferenced = errors.New("file: a row references it")
	// ErrFileScopeAbsent reports that the unit of work carries no owner of the scope:
	// no acting user for the user scope, or no acting workspace for the workspace scope.
	ErrFileScopeAbsent = errors.New("file: no owner of the scope")
)

// PluginFiles reads, claims, and deletes the files of one plugin in the acting scope.
type PluginFiles interface {
	// Get answers the files under the identifiers. It omits an identifier that the
	// plugin and the acting scope do not hold.
	Get(ctx context.Context, ids ...string) ([]File, error)
	// Claim marks the file as used by a row of the plugin, so the hub keeps it past
	// the expiry of an unclaimed file. A file in any state takes a claim.
	Claim(ctx context.Context, id string) error
	// Delete removes the file and its bytes. It answers ErrFileReferenced while a row
	// references the file.
	Delete(ctx context.Context, id string) error
}
