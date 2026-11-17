package registrar_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/plugin/registrar"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	photoKind    = "photo"
	paperKind    = "paper"
	photoUpload  = thingsWrite
	mediaTypes   = "media_types"
	megabyte     = 1 << 20
	jpegType     = "image/jpeg"
	pdfType      = "application/pdf"
	fileKindRule = "file kind"
)

// filePlugin declares the file kinds that the test gives it.
type filePlugin struct {
	id    string
	kinds []pluginapi.FileKind
}

var (
	_ pluginapi.Plugin     = (*filePlugin)(nil)
	_ pluginapi.FilePlugin = (*filePlugin)(nil)
)

func (p *filePlugin) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:         pluginapi.ParsePluginID(p.id),
		Version:    pluginVersion,
		APIVersion: pluginapi.APIVersion,
	}
}

func (p *filePlugin) FileKinds() ([]pluginapi.FileKind, error) {
	return p.kinds, nil
}

func photoFileKind() pluginapi.FileKind {
	return pluginapi.FileKind{
		Key:        photoKind,
		Scope:      pluginapi.FileScopeWorkspace,
		MediaTypes: []string{jpegType, "image/png", "image/webp", "image/heic", "image/heif"},
		MaxSize:    pluginapi.MaxFileSize,
		Permission: photoUpload,
		Image:      true,
	}
}

func paperFileKind() pluginapi.FileKind {
	return pluginapi.FileKind{
		Key:        paperKind,
		Scope:      pluginapi.FileScopeWorkspace,
		MediaTypes: []string{pdfType},
		MaxSize:    5 * megabyte,
		Permission: photoUpload,
	}
}

func newFileKindRegistrar(
	t *testing.T,
) (*registrar.FileKindRegistrar, *registry.FileKindRegistry, *registry.CapabilityRegistry) {
	t.Helper()

	kinds := registry.NewFileKindRegistry()
	capabilities := registry.NewCapabilityRegistry()
	permissions := thingPermissions(t, widgetPluginID, otherPluginID)

	return registrar.NewFileKindRegistrar(kinds, capabilities, permissions), kinds, capabilities
}

// FILES-SC-001: The registrar stores each declared kind with the full name of its
// permission, and the capability report of the plugin holds files.
func TestTheFileKindRegistrarStoresEachDeclaredKind(t *testing.T) {
	t.Parallel()

	reg, kinds, capabilities := newFileKindRegistrar(t)
	plugin := &filePlugin{
		id:    widgetPluginID,
		kinds: []pluginapi.FileKind{photoFileKind(), paperFileKind()},
	}

	require.True(t, reg.Supports(loaded(plugin)))
	require.NoError(t, reg.Register(loaded(plugin)))

	photo, found := kinds.Get(widgetPluginID, photoKind)
	require.True(t, found)
	require.Equal(t, photoFileKind(), photo.Kind)
	require.Equal(t, widgetPluginID+":"+photoUpload, photo.Permission)

	_, found = kinds.Get(widgetPluginID, paperKind)
	require.True(t, found)

	report := capabilities.Of(widgetPluginID)
	require.Equal(t, []registry.FileKindItem{
		{
			PluginID:   widgetPluginID,
			Key:        photoKind,
			Scope:      string(pluginapi.FileScopeWorkspace),
			MediaTypes: photoFileKind().MediaTypes,
			MaxSize:    pluginapi.MaxFileSize,
			Image:      true,
			Permission: widgetPluginID + ":" + photoUpload,
		},
		{
			PluginID:   widgetPluginID,
			Key:        paperKind,
			Scope:      string(pluginapi.FileScopeWorkspace),
			MediaTypes: []string{pdfType},
			MaxSize:    5 * megabyte,
			Permission: widgetPluginID + ":" + photoUpload,
		},
	}, report[registry.CapFiles])
}

// FILES-SC-001: A plugin that accepts no file is no concern of the registrar.
func TestTheFileKindRegistrarSkipsAPluginWithNoFileKind(t *testing.T) {
	t.Parallel()

	reg, _, _ := newFileKindRegistrar(t)

	require.False(t, reg.Supports(loaded(&widgetPlugin{id: widgetPluginID})))
}

// FILES-SC-002: A size limit above 25 MB fails the load, and the error names the
// plugin, the kind, and max_size.
func TestTheFileKindRegistrarRefusesALimitAbove25MB(t *testing.T) {
	t.Parallel()

	reg, kinds, _ := newFileKindRegistrar(t)
	kind := photoFileKind()
	kind.MaxSize = 26 * megabyte

	err := reg.Register(loaded(&filePlugin{id: widgetPluginID, kinds: []pluginapi.FileKind{kind}}))

	require.ErrorIs(t, err, errs.ErrInvalidFileKind)
	require.Contains(t, err.Error(), widgetPluginID)
	require.Contains(t, err.Error(), photoKind)
	require.True(t, strings.HasSuffix(err.Error(), "max_size"))
	require.Empty(t, kinds.Of(widgetPluginID))
}

// fileKindCase changes one member of a valid kind, and names the error and the rule
// that the change breaks. A case with no error keeps every rule.
type fileKindCase struct {
	change  func(kind *pluginapi.FileKind)
	wantErr error
	suffix  string
}

func fileKindCases() map[string]fileKindCase {
	return map[string]fileKindCase{
		"a kind that keeps every rule": {change: func(*pluginapi.FileKind) {}},
		"a limit of exactly 25 MB": {
			change: func(k *pluginapi.FileKind) { k.MaxSize = pluginapi.MaxFileSize },
		},
		"a kind of the user scope": {
			change: func(k *pluginapi.FileKind) { k.Scope = pluginapi.FileScopeUser },
		},
		"no media type": {
			change:  func(k *pluginapi.FileKind) { k.MediaTypes = nil },
			wantErr: errs.ErrInvalidFileKind, suffix: mediaTypes,
		},
		"a media type with no subtype": {
			change:  func(k *pluginapi.FileKind) { k.MediaTypes = []string{"attachment"} },
			wantErr: errs.ErrInvalidFileKind, suffix: mediaTypes,
		},
		"a media type with a parameter": {
			change:  func(k *pluginapi.FileKind) { k.MediaTypes = []string{"text/plain; charset=utf-8"} },
			wantErr: errs.ErrInvalidFileKind,
			suffix:  mediaTypes,
		},
		"a media type in uppercase": {
			change:  func(k *pluginapi.FileKind) { k.MediaTypes = []string{"Image/JPEG"} },
			wantErr: errs.ErrInvalidFileKind, suffix: mediaTypes,
		},
		"an image kind that accepts a document": {
			change: func(k *pluginapi.FileKind) {
				k.Image = true
				k.MediaTypes = []string{jpegType, pdfType}
			},
			wantErr: errs.ErrInvalidFileKind, suffix: "image",
		},
		"a limit of zero": {
			change:  func(k *pluginapi.FileKind) { k.MaxSize = 0 },
			wantErr: errs.ErrInvalidFileKind, suffix: "max_size",
		},
		"an unknown scope": {
			change:  func(k *pluginapi.FileKind) { k.Scope = "instance" },
			wantErr: errs.ErrInvalidFileKind, suffix: "scope",
		},
		"a key with an uppercase letter": {
			change:  func(k *pluginapi.FileKind) { k.Key = "Paper" },
			wantErr: errs.ErrInvalidFileKind, suffix: "key",
		},
		"no permission": {
			change:  func(k *pluginapi.FileKind) { k.Permission = "" },
			wantErr: errs.ErrPermissionMissing,
		},
		"a permission the plugin does not declare": {
			change:  func(k *pluginapi.FileKind) { k.Permission = "things.delete" },
			wantErr: errs.ErrPermissionUndeclared,
		},
	}
}

// FILES-SC-003: The registrar refuses a kind with no media type, with a permission
// that the plugin does not declare, or with an image flag over a document type, and
// stores no kind of that plugin.
func TestTheFileKindRegistrarChecksEachRuleOfAKind(t *testing.T) {
	t.Parallel()

	for name, tc := range fileKindCases() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			reg, kinds, _ := newFileKindRegistrar(t)
			kind := paperFileKind()
			tc.change(&kind)

			err := reg.Register(loaded(&filePlugin{
				id:    widgetPluginID,
				kinds: []pluginapi.FileKind{photoFileKind(), kind},
			}))

			if tc.wantErr == nil {
				require.NoError(t, err)
				require.Len(t, kinds.Of(widgetPluginID), 2)

				return
			}

			require.ErrorIs(t, err, tc.wantErr)
			require.Contains(t, err.Error(), widgetPluginID)
			require.True(t, strings.HasSuffix(err.Error(), tc.suffix))
			require.Empty(t, kinds.Of(widgetPluginID))
		})
	}
}

// FILES-SC-003: Two kinds of one plugin under one key fail the load, and two plugins
// keep the same key apart.
func TestTheFileKindRegistrarRefusesTwoKindsOfOnePluginUnderOneKey(t *testing.T) {
	t.Parallel()

	reg, kinds, _ := newFileKindRegistrar(t)

	err := reg.Register(loaded(&filePlugin{
		id:    widgetPluginID,
		kinds: []pluginapi.FileKind{paperFileKind(), paperFileKind()},
	}))
	require.ErrorIs(t, err, errs.ErrFileKindAlreadyRegistered)
	require.Contains(t, err.Error(), fileKindRule+" "+paperKind)
	require.Empty(t, kinds.Of(widgetPluginID))

	for _, id := range []string{widgetPluginID, otherPluginID} {
		require.NoError(t, reg.Register(loaded(&filePlugin{
			id:    id,
			kinds: []pluginapi.FileKind{paperFileKind()},
		})))
	}
}
