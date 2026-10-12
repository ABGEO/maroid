package pluginapi_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// WIDGET-SC-001: NewWidget answers its meta and the answer of its load function.
func TestNewWidgetAnswersItsMetaAndItsLoad(t *testing.T) {
	t.Parallel()

	meta := pluginapi.WidgetMeta{
		Key:        "balance",
		Title:      "Balance",
		Kind:       pluginapi.WidgetKindStat,
		Size:       pluginapi.WidgetSizeS,
		Permission: "things.read",
	}
	answer := pluginapi.StatAnswer(pluginapi.StatData{Value: "4.50"})

	widget := pluginapi.NewWidget(meta, func(context.Context) (pluginapi.WidgetAnswer, error) {
		return answer, nil
	})

	loaded, err := widget.Load(t.Context())

	require.NoError(t, err)
	require.Equal(t, meta, widget.Meta())
	require.Equal(t, answer, loaded)
}

// WIDGET-SC-001: Each answer helper sets its state and the members of that state alone.
func TestEachAnswerHelperSetsItsStateAlone(t *testing.T) {
	t.Parallel()

	stat := pluginapi.StatData{Value: "4.50", Unit: "GEL"}
	list := pluginapi.ListData{
		Items: []pluginapi.ListItem{{Label: "P001 Fern"}},
		More:  "/attention",
	}

	cases := map[string]struct {
		got  pluginapi.WidgetAnswer
		want pluginapi.WidgetAnswer
	}{
		"stat": {
			got:  pluginapi.StatAnswer(stat),
			want: pluginapi.WidgetAnswer{State: pluginapi.WidgetStateReady, Stat: &stat},
		},
		"list": {
			got:  pluginapi.ListAnswer(list),
			want: pluginapi.WidgetAnswer{State: pluginapi.WidgetStateReady, List: &list},
		},
		"empty": {
			got: pluginapi.EmptyAnswer("No plant needs care."),
			want: pluginapi.WidgetAnswer{
				State:   pluginapi.WidgetStateEmpty,
				Message: "No plant needs care.",
			},
		},
		"not set up": {
			got: pluginapi.NotSetUpAnswer("Connect your parking account.", "/account"),
			want: pluginapi.WidgetAnswer{
				State:   pluginapi.WidgetStateNotSetUp,
				Message: "Connect your parking account.",
				Link:    "/account",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, tc.got)
		})
	}
}
