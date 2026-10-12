package widget_test

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/widget"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	pluginID   = "dev.maroid.p"
	statKey    = "balance"
	listKey    = "things"
	testValue  = "4.50"
	emptyText  = "Nothing."
	setUpText  = "Connect it."
	setUpLink  = "/account"
	fernLabel  = "P001 Fern"
	shortWait  = 50 * time.Millisecond
	acceptable = 100 * time.Millisecond
)

func widgetOf(
	kind pluginapi.WidgetKind,
	load func(context.Context) (pluginapi.WidgetAnswer, error),
) registry.Widget {
	key := statKey
	if kind == pluginapi.WidgetKindList {
		key = listKey
	}

	return registry.Widget{
		PluginID: pluginID,
		Meta: pluginapi.WidgetMeta{
			Key: key, Title: "Title", Kind: kind, Size: pluginapi.WidgetSizeS,
		},
		Permission: pluginID + ":things.read",
		Load:       load,
	}
}

func answering(
	answer pluginapi.WidgetAnswer,
) func(context.Context) (pluginapi.WidgetAnswer, error) {
	return func(context.Context) (pluginapi.WidgetAnswer, error) {
		return answer, nil
	}
}

func read(t *testing.T, target registry.Widget) (pluginapi.WidgetAnswer, error) {
	t.Helper()

	//nolint:wrapcheck // the test reads the error that the reader answered.
	return widget.New(widget.Deadline).Read(t.Context(), target)
}

// WIDGET-SC-008: Each read calls Load again, so the reader keeps no answer.
func TestEachReadAsksThePluginAgain(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	target := widgetOf(
		pluginapi.WidgetKindStat,
		func(context.Context) (pluginapi.WidgetAnswer, error) {
			count := calls.Add(1)

			return pluginapi.StatAnswer(pluginapi.StatData{Value: strconv.Itoa(int(count))}), nil
		},
	)

	_, err := read(t, target)
	require.NoError(t, err)

	second, err := read(t, target)
	require.NoError(t, err)

	require.Equal(t, int32(2), calls.Load())
	require.Equal(t, "2", second.Stat.Value)
}

// WIDGET-SC-010: An answer that breaks the shape of its state or of its kind fails.
func TestTheReaderRefusesAnAnswerThatBreaksItsShape(t *testing.T) {
	t.Parallel()

	stat := pluginapi.StatData{Value: testValue}
	list := pluginapi.ListData{Items: []pluginapi.ListItem{{Label: fernLabel}}}

	cases := map[string]struct {
		kind   pluginapi.WidgetKind
		answer pluginapi.WidgetAnswer
	}{
		"an unknown state": {pluginapi.WidgetKindStat, pluginapi.WidgetAnswer{State: "broken"}},
		"ready with no stat": {
			pluginapi.WidgetKindStat, pluginapi.WidgetAnswer{State: pluginapi.WidgetStateReady},
		},
		"ready with an empty value": {
			pluginapi.WidgetKindStat, pluginapi.StatAnswer(pluginapi.StatData{}),
		},
		"a list for a stat": {pluginapi.WidgetKindStat, pluginapi.ListAnswer(list)},
		"a stat for a list": {pluginapi.WidgetKindList, pluginapi.StatAnswer(stat)},
		"ready with no list": {
			pluginapi.WidgetKindList,
			pluginapi.WidgetAnswer{State: pluginapi.WidgetStateReady},
		},
		"a stat that also carries a list": {
			pluginapi.WidgetKindStat,
			pluginapi.WidgetAnswer{State: pluginapi.WidgetStateReady, Stat: &stat, List: &list},
		},
		"a list that also carries a stat": {
			pluginapi.WidgetKindList,
			pluginapi.WidgetAnswer{State: pluginapi.WidgetStateReady, Stat: &stat, List: &list},
		},
		"an item with no label": {
			pluginapi.WidgetKindList,
			pluginapi.ListAnswer(pluginapi.ListData{Items: []pluginapi.ListItem{{Meta: "due"}}}),
		},
		"empty with no message": {pluginapi.WidgetKindStat, pluginapi.EmptyAnswer("")},
		"not set up with no message": {
			pluginapi.WidgetKindStat,
			pluginapi.NotSetUpAnswer("", setUpLink),
		},
		"not set up with no link": {
			pluginapi.WidgetKindStat,
			pluginapi.NotSetUpAnswer(setUpText, ""),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := read(t, widgetOf(tc.kind, answering(tc.answer)))

			require.ErrorIs(t, err, errs.ErrInvalidWidgetAnswer)
		})
	}
}

// WIDGET-SC-010: A list with no item, and each answer of the helpers with its
// members set, pass.
func TestTheReaderPassesEveryAnswerOfTheHelpers(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		kind   pluginapi.WidgetKind
		answer pluginapi.WidgetAnswer
	}{
		"a stat": {
			pluginapi.WidgetKindStat,
			pluginapi.StatAnswer(pluginapi.StatData{Value: testValue}),
		},
		"a list": {
			pluginapi.WidgetKindList,
			pluginapi.ListAnswer(
				pluginapi.ListData{Items: []pluginapi.ListItem{{Label: fernLabel}}},
			),
		},
		"a list with no item": {
			pluginapi.WidgetKindList,
			pluginapi.ListAnswer(pluginapi.ListData{}),
		},
		"empty": {pluginapi.WidgetKindList, pluginapi.EmptyAnswer(emptyText)},
		"not set up": {
			pluginapi.WidgetKindStat,
			pluginapi.NotSetUpAnswer(setUpText, setUpLink),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			answer, err := read(t, widgetOf(tc.kind, answering(tc.answer)))

			require.NoError(t, err)
			require.Equal(t, tc.answer, answer)
		})
	}
}

// pluginLinks maps each link to whether it is a path of the plugin.
func pluginLinks() map[string]bool {
	return map[string]bool{
		"/account":            true,
		"/plants/0199c5e2":    true,
		"":                    true,
		"account":             false,
		"https://example.com": false,
		"//example.com":       false,
		`/\example.com`:       false,
		"/plants/../../x":     false,
		"/plants/..":          false,
	}
}

// WIDGET-SC-012: A link is a path relative to the plugin, in every place that an
// answer carries one.
func TestTheReaderAcceptsOnlyAPathOfThePlugin(t *testing.T) {
	t.Parallel()

	links := pluginLinks()

	places := map[string]func(link string) (pluginapi.WidgetKind, pluginapi.WidgetAnswer){
		"the link of a stat": func(link string) (pluginapi.WidgetKind, pluginapi.WidgetAnswer) {
			return pluginapi.WidgetKindStat, pluginapi.StatAnswer(
				pluginapi.StatData{Value: testValue, Link: link},
			)
		},
		"the link to see all": func(link string) (pluginapi.WidgetKind, pluginapi.WidgetAnswer) {
			return pluginapi.WidgetKindList, pluginapi.ListAnswer(pluginapi.ListData{More: link})
		},
		"the link of an item": func(link string) (pluginapi.WidgetKind, pluginapi.WidgetAnswer) {
			return pluginapi.WidgetKindList, pluginapi.ListAnswer(pluginapi.ListData{
				Items: []pluginapi.ListItem{{Label: fernLabel, Link: link}},
			})
		},
	}

	for place, build := range places {
		for link, valid := range links {
			t.Run(place+" "+link, func(t *testing.T) {
				t.Parallel()

				kind, answer := build(link)
				_, err := read(t, widgetOf(kind, answering(answer)))

				if valid {
					require.NoError(t, err)

					return
				}

				require.ErrorIs(t, err, errs.ErrInvalidWidgetAnswer)
			})
		}
	}
}

// WIDGET-SC-012: The link of not set up is a path of the plugin too.
func TestTheReaderChecksTheLinkOfNotSetUp(t *testing.T) {
	t.Parallel()

	for link, valid := range pluginLinks() {
		if link == "" {
			continue
		}

		t.Run("the link of not set up "+link, func(t *testing.T) {
			t.Parallel()

			_, err := read(
				t,
				widgetOf(
					pluginapi.WidgetKindStat,
					answering(pluginapi.NotSetUpAnswer(setUpText, link)),
				),
			)

			if valid {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, errs.ErrInvalidWidgetAnswer)
		})
	}
}

// WIDGET-NFR-001, WIDGET-SC-013: The hub waits 5 seconds, and a Load that ignores
// its context still ends the read at the deadline.
func TestTheReaderStopsWaitingAtTheDeadline(t *testing.T) {
	t.Parallel()

	require.Equal(t, 5*time.Second, widget.Deadline)

	release := make(chan struct{})

	t.Cleanup(func() { close(release) })

	target := widgetOf(
		pluginapi.WidgetKindStat,
		func(context.Context) (pluginapi.WidgetAnswer, error) {
			select {
			case <-release:
			case <-time.After(10 * time.Second):
			}

			return pluginapi.StatAnswer(pluginapi.StatData{Value: testValue}), nil
		},
	)

	started := time.Now()
	_, err := widget.New(shortWait).Read(t.Context(), target)

	require.ErrorIs(t, err, errs.ErrWidgetTimeout)
	require.Less(t, time.Since(started), acceptable)
}

// WIDGET-NFR-001, WIDGET-SC-013: A Load that honours its context and returns at the
// deadline answers the timeout too, not a failure of the plugin.
func TestTheReaderTimesOutALoadThatHonoursItsContext(t *testing.T) {
	t.Parallel()

	target := widgetOf(
		pluginapi.WidgetKindStat,
		func(ctx context.Context) (pluginapi.WidgetAnswer, error) {
			<-ctx.Done()

			return pluginapi.WidgetAnswer{}, ctx.Err()
		},
	)

	for range 20 {
		_, err := widget.New(shortWait).Read(t.Context(), target)

		require.ErrorIs(t, err, errs.ErrWidgetTimeout)
	}
}

// WIDGET-SC-013: A request that the client gives up is no timeout of the plugin.
func TestTheReaderTellsACancelledRequestFromATimeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	target := widgetOf(
		pluginapi.WidgetKindStat,
		func(ctx context.Context) (pluginapi.WidgetAnswer, error) {
			<-ctx.Done()

			return pluginapi.WidgetAnswer{}, ctx.Err()
		},
	)

	_, err := widget.New(widget.Deadline).Read(ctx, target)

	require.ErrorIs(t, err, context.Canceled)
	require.NotErrorIs(t, err, errs.ErrWidgetTimeout)
}

// WIDGET-SC-011: A Load that panics answers an error, and the process keeps running.
func TestTheReaderTurnsAPanicIntoAnError(t *testing.T) {
	t.Parallel()

	target := widgetOf(
		pluginapi.WidgetKindStat,
		func(context.Context) (pluginapi.WidgetAnswer, error) {
			panic("the plugin broke")
		},
	)

	_, err := read(t, target)

	require.ErrorContains(t, err, "the plugin broke")
}

// WIDGET-SC-009: Load reads the acting user and the acting workspace of the request.
func TestTheReaderKeepsTheActorsOfTheRequest(t *testing.T) {
	t.Parallel()

	target := widgetOf(
		pluginapi.WidgetKindStat,
		func(ctx context.Context) (pluginapi.WidgetAnswer, error) {
			return pluginapi.StatAnswer(pluginapi.StatData{
				Value:   pluginapi.ActingUserFromContext(ctx),
				Caption: pluginapi.ActingWorkspaceFromContext(ctx),
			}), nil
		},
	)

	ctx := pluginapi.ContextWithActingWorkspace(
		pluginapi.ContextWithActingUser(t.Context(), "user-a"),
		"workspace-w",
	)

	answer, err := widget.New(widget.Deadline).Read(ctx, target)

	require.NoError(t, err)
	require.Equal(t, "user-a", answer.Stat.Value)
	require.Equal(t, "workspace-w", answer.Stat.Caption)
}
