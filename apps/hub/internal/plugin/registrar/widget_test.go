package registrar_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/plugin/registrar"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	widgetPluginID = "dev.maroid.p"
	otherPluginID  = "dev.maroid.q"
	thingsRead     = "things.read"
	thingsWrite    = "things.write"
	balanceTitle   = "Balance"
	keyRule        = "key"
	titleRule      = "title"
)

// widgetRuleCase changes one member of a valid meta, and names the error and the
// rule that the change breaks. A case with no error keeps every rule.
type widgetRuleCase struct {
	change  func(meta *pluginapi.WidgetMeta)
	wantErr error
	suffix  string
}

// widgetPlugin declares the widgets that the test gives it.
type widgetPlugin struct {
	id    string
	metas []pluginapi.WidgetMeta
}

var (
	_ pluginapi.Plugin       = (*widgetPlugin)(nil)
	_ pluginapi.WidgetPlugin = (*widgetPlugin)(nil)
)

func (p *widgetPlugin) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:         pluginapi.ParsePluginID(p.id),
		Version:    pluginVersion,
		APIVersion: pluginapi.APIVersion,
	}
}

func (p *widgetPlugin) Widgets() ([]pluginapi.Widget, error) {
	widgets := make([]pluginapi.Widget, 0, len(p.metas))
	for _, meta := range p.metas {
		widgets = append(widgets, pluginapi.NewWidget(
			meta,
			func(context.Context) (pluginapi.WidgetAnswer, error) {
				return pluginapi.EmptyAnswer("Nothing."), nil
			},
		))
	}

	return widgets, nil
}

func balanceMeta() pluginapi.WidgetMeta {
	return pluginapi.WidgetMeta{
		Key:        "balance",
		Title:      balanceTitle,
		Kind:       pluginapi.WidgetKindStat,
		Size:       pluginapi.WidgetSizeS,
		Permission: thingsRead,
	}
}

// thingPermissions declares things.read and things.write for each plugin it names.
func thingPermissions(t *testing.T, pluginIDs ...string) *registry.PermissionRegistry {
	t.Helper()

	permissions := registry.NewPermissionRegistry()

	for _, id := range pluginIDs {
		require.NoError(t, permissions.Register(
			registry.PermissionEntry{Name: id + ":" + thingsRead, Lowest: pluginapi.RoleViewer},
			registry.PermissionEntry{Name: id + ":" + thingsWrite, Lowest: pluginapi.RoleEditor},
		))
	}

	return permissions
}

func newWidgetRegistrar(
	t *testing.T,
) (*registrar.WidgetRegistrar, *registry.WidgetRegistry, *registry.CapabilityRegistry) {
	t.Helper()

	widgets := registry.NewWidgetRegistry()
	capabilities := registry.NewCapabilityRegistry()
	permissions := thingPermissions(t, widgetPluginID, otherPluginID)

	return registrar.NewWidgetRegistrar(widgets, capabilities, permissions), widgets, capabilities
}

func widgetRuleCases() map[string]widgetRuleCase {
	return map[string]widgetRuleCase{
		"a widget that keeps every rule": {change: func(*pluginapi.WidgetMeta) {}},
		"a title with white space": {
			change: func(m *pluginapi.WidgetMeta) { m.Title = "  " + balanceTitle + "  " },
		},
		"a key of 64 characters": {
			change: func(m *pluginapi.WidgetMeta) { m.Key = strings.Repeat("a", 64) },
		},
		"an empty key": {
			change:  func(m *pluginapi.WidgetMeta) { m.Key = "" },
			wantErr: errs.ErrInvalidWidget, suffix: keyRule,
		},
		"a key of 65 characters": {
			change:  func(m *pluginapi.WidgetMeta) { m.Key = strings.Repeat("a", 65) },
			wantErr: errs.ErrInvalidWidget, suffix: keyRule,
		},
		"a key with an uppercase letter": {
			change:  func(m *pluginapi.WidgetMeta) { m.Key = balanceTitle },
			wantErr: errs.ErrInvalidWidget, suffix: keyRule,
		},
		"a key with an underscore": {
			change:  func(m *pluginapi.WidgetMeta) { m.Key = "a_b" },
			wantErr: errs.ErrInvalidWidget, suffix: keyRule,
		},
		"an empty title": {
			change:  func(m *pluginapi.WidgetMeta) { m.Title = "" },
			wantErr: errs.ErrInvalidWidget, suffix: titleRule,
		},
		"a title of white space": {
			change:  func(m *pluginapi.WidgetMeta) { m.Title = "   " },
			wantErr: errs.ErrInvalidWidget, suffix: titleRule,
		},
		"a title of 65 characters": {
			change:  func(m *pluginapi.WidgetMeta) { m.Title = strings.Repeat("a", 65) },
			wantErr: errs.ErrInvalidWidget, suffix: titleRule,
		},
		"an unknown kind": {
			change:  func(m *pluginapi.WidgetMeta) { m.Kind = "chart" },
			wantErr: errs.ErrInvalidWidget, suffix: "kind",
		},
		"an unknown size": {
			change:  func(m *pluginapi.WidgetMeta) { m.Size = "xl" },
			wantErr: errs.ErrInvalidWidget, suffix: "size",
		},
		"no permission": {
			change:  func(m *pluginapi.WidgetMeta) { m.Permission = "" },
			wantErr: errs.ErrPermissionMissing,
		},
		"a permission the plugin does not declare": {
			change:  func(m *pluginapi.WidgetMeta) { m.Permission = "things.delete" },
			wantErr: errs.ErrPermissionUndeclared,
		},
	}
}

// WIDGET-SC-002: The registrar stores a widget that keeps every rule, with its
// title trimmed, and refuses one that breaks a rule, naming the plugin, the key,
// and the rule.
func TestTheWidgetRegistrarChecksEachRuleOfAWidget(t *testing.T) {
	t.Parallel()

	for name, tc := range widgetRuleCases() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			reg, widgets, _ := newWidgetRegistrar(t)
			meta := balanceMeta()
			tc.change(&meta)

			err := reg.Register(
				&widgetPlugin{id: widgetPluginID, metas: []pluginapi.WidgetMeta{meta}},
			)

			if tc.wantErr == nil {
				require.NoError(t, err)

				stored, found := widgets.Get(widgetPluginID, meta.Key)
				require.True(t, found)
				require.Equal(t, strings.TrimSpace(meta.Title), stored.Meta.Title)

				return
			}

			require.ErrorIs(t, err, tc.wantErr)
			require.Contains(t, err.Error(), widgetPluginID)
			require.Contains(t, err.Error(), meta.Key)
			require.True(t, strings.HasSuffix(err.Error(), tc.suffix))
			require.Empty(t, widgets.Of(widgetPluginID))
		})
	}
}

// WIDGET-SC-003: Two widgets of one plugin under one key fail the load.
func TestTheWidgetRegistrarRefusesTwoWidgetsOfOnePluginUnderOneKey(t *testing.T) {
	t.Parallel()

	reg, widgets, _ := newWidgetRegistrar(t)

	err := reg.Register(&widgetPlugin{
		id:    widgetPluginID,
		metas: []pluginapi.WidgetMeta{balanceMeta(), balanceMeta()},
	})

	require.ErrorIs(t, err, errs.ErrWidgetAlreadyRegistered)
	require.Empty(t, widgets.Of(widgetPluginID))
}

// WIDGET-SC-003: Two plugins each declare a widget under one key, and both load.
func TestTheWidgetRegistrarKeepsOneKeyInTwoPlugins(t *testing.T) {
	t.Parallel()

	reg, widgets, _ := newWidgetRegistrar(t)
	summary := balanceMeta()
	summary.Key = "summary"

	require.NoError(
		t,
		reg.Register(&widgetPlugin{id: widgetPluginID, metas: []pluginapi.WidgetMeta{summary}}),
	)
	require.NoError(
		t,
		reg.Register(&widgetPlugin{id: otherPluginID, metas: []pluginapi.WidgetMeta{summary}}),
	)

	_, inP := widgets.Get(widgetPluginID, "summary")
	_, inQ := widgets.Get(otherPluginID, "summary")

	require.True(t, inP)
	require.True(t, inQ)
}

// WIDGET-SC-003: A list at the smallest size loads. The registrar checks no pair
// of a kind and a size.
func TestTheWidgetRegistrarAcceptsAListAtTheSmallestSize(t *testing.T) {
	t.Parallel()

	reg, _, _ := newWidgetRegistrar(t)
	meta := balanceMeta()
	meta.Kind = pluginapi.WidgetKindList
	meta.Size = pluginapi.WidgetSizeXS

	require.NoError(
		t,
		reg.Register(&widgetPlugin{id: widgetPluginID, metas: []pluginapi.WidgetMeta{meta}}),
	)
}

// WIDGET-SC-004: The capability of a plugin names each widget in the order of the
// declaration, with the full name of its permission.
func TestTheWidgetRegistrarRecordsTheWidgetsInTheOrderOfTheDeclaration(t *testing.T) {
	t.Parallel()

	reg, widgets, capabilities := newWidgetRegistrar(t)

	second := balanceMeta()
	second.Key = "b"
	first := balanceMeta()
	first.Key = "a"
	first.Title = "Things"
	first.Kind = pluginapi.WidgetKindList
	first.Size = pluginapi.WidgetSizeM

	require.NoError(t, reg.Register(&widgetPlugin{
		id:    widgetPluginID,
		metas: []pluginapi.WidgetMeta{second, first},
	}))

	permission := widgetPluginID + ":" + thingsRead

	require.Equal(t, []registry.WidgetItem{
		{
			PluginID:   widgetPluginID,
			Key:        "b",
			Title:      balanceTitle,
			Kind:       "stat",
			Size:       "s",
			Permission: permission,
		},
		{
			PluginID:   widgetPluginID,
			Key:        "a",
			Title:      "Things",
			Kind:       "list",
			Size:       "m",
			Permission: permission,
		},
	}, capabilities.Of(widgetPluginID)[registry.CapWidgets])

	keys := make([]string, 0, 2)
	for _, widget := range widgets.Of(widgetPluginID) {
		keys = append(keys, widget.Meta.Key)
	}

	require.Equal(t, []string{"b", "a"}, keys)
}

// WIDGET-FR-001: A plugin that declares no widget is not a widget plugin, and the
// registrar skips it.
func TestTheWidgetRegistrarSkipsAPluginThatDeclaresNoWidget(t *testing.T) {
	t.Parallel()

	reg, _, _ := newWidgetRegistrar(t)

	require.False(t, reg.Supports(&plainPlugin{}))
	require.True(t, reg.Supports(&widgetPlugin{id: widgetPluginID}))
}
