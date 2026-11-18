package registrar

import (
	"fmt"
	"mime"
	"regexp"
	"slices"
	"strings"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// fileKindKeyPattern keeps a file kind key safe in an object key and in a column.
var fileKindKeyPattern = regexp.MustCompile(`^[a-z0-9-]{1,64}$`)

// FileKindRegistrar registers the file kinds that a plugin accepts.
type FileKindRegistrar struct {
	registry     *registry.FileKindRegistry
	capabilities *registry.CapabilityRegistry
	permissions  *registry.PermissionRegistry
}

var _ Registrar = (*FileKindRegistrar)(nil)

// NewFileKindRegistrar creates a new FileKindRegistrar.
func NewFileKindRegistrar(
	reg *registry.FileKindRegistry,
	capabilities *registry.CapabilityRegistry,
	permissions *registry.PermissionRegistry,
) *FileKindRegistrar {
	return &FileKindRegistrar{
		registry:     reg,
		capabilities: capabilities,
		permissions:  permissions,
	}
}

// Name returns the name of the registrar.
func (r *FileKindRegistrar) Name() string {
	return "file kind"
}

// Supports indicates whether the registrar can handle the given plugin.
func (r *FileKindRegistrar) Supports(plugin registry.LoadedPlugin) bool {
	_, ok := plugin.Instance.(pluginapi.FilePlugin)

	return ok
}

// Register checks every file kind of the plugin before it stores any, so a plugin
// with one broken kind loads none.
func (r *FileKindRegistrar) Register(plugin registry.LoadedPlugin) error {
	id := plugin.Meta.ID

	filePlugin, ok := plugin.Instance.(pluginapi.FilePlugin)
	if !ok {
		return fmt.Errorf(
			"plugin %s does not support File capability: %w",
			id,
			errs.ErrPluginCapabilityNotSupported,
		)
	}

	declared, err := filePlugin.FileKinds()
	if err != nil {
		return fmt.Errorf("retrieving the file kinds for plugin %s: %w", id, err)
	}

	kinds := make([]registry.FileKind, 0, len(declared))
	items := make([]registry.FileKindItem, 0, len(declared))

	for _, kind := range declared {
		if err = validateFileKind(id, kind); err != nil {
			return err
		}

		permission, permissionErr := permissionOf(r.permissions, id, kind.Key, kind.Permission)
		if permissionErr != nil {
			return permissionErr
		}

		kind.MediaTypes = slices.Clone(kind.MediaTypes)

		kinds = append(kinds, registry.FileKind{
			PluginID:   id.String(),
			Kind:       kind,
			Permission: permission,
		})
		items = append(items, registry.FileKindItem{
			PluginID:   id.String(),
			Key:        kind.Key,
			Scope:      string(kind.Scope),
			MediaTypes: kind.MediaTypes,
			MaxSize:    kind.MaxSize,
			Image:      kind.Image,
			Permission: permission,
		})
	}

	if err = r.registry.Register(kinds...); err != nil {
		return fmt.Errorf("registering the file kinds for plugin %s: %w", id, err)
	}

	r.capabilities.Record(id, registry.CapFiles, items)

	return nil
}

func validateFileKind(id *pluginapi.PluginID, kind pluginapi.FileKind) error {
	var rule string

	switch {
	case !fileKindKeyPattern.MatchString(kind.Key):
		rule = "key"
	case !knownFileScope(kind.Scope):
		rule = "scope"
	case len(kind.MediaTypes) == 0 || slices.ContainsFunc(kind.MediaTypes, notMediaType):
		rule = "media_types"
	case kind.MaxSize < 1 || kind.MaxSize > pluginapi.MaxFileSize:
		rule = "max_size"
	case mixesImageTypes(kind):
		rule = "image"
	default:
		return nil
	}

	return fmt.Errorf(
		"%w: plugin %s, file kind %s: %s",
		errs.ErrInvalidFileKind,
		id,
		kind.Key,
		rule,
	)
}

// notMediaType refuses all but a bare type and subtype in lowercase, with no parameter.
// ParseMediaType alone accepts a value with no subtype, such as a disposition.
func notMediaType(value string) bool {
	parsed, params, err := mime.ParseMediaType(value)

	return err != nil || len(params) > 0 || parsed != value || strings.Count(value, "/") != 1
}

func knownFileScope(scope pluginapi.FileScope) bool {
	return scope == pluginapi.FileScopeWorkspace || scope == pluginapi.FileScopeUser
}

// isImageMediaType reports a media type that the hub derives the variants from. A kind
// that is not an image keeps its upload with the metadata, so it takes none of them.
func isImageMediaType(value string) bool {
	switch value {
	case "image/jpeg", "image/png", "image/webp", "image/heic", "image/heif":
		return true
	default:
		return false
	}
}

func notImageMediaType(value string) bool {
	return !isImageMediaType(value)
}

// mixesImageTypes reports an image kind with a type that is not an image, or a kind
// that is not an image with an image type.
func mixesImageTypes(kind pluginapi.FileKind) bool {
	if kind.Image {
		return slices.ContainsFunc(kind.MediaTypes, notImageMediaType)
	}

	return slices.ContainsFunc(kind.MediaTypes, isImageMediaType)
}
