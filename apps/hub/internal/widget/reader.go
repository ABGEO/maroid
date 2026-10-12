// Package widget reads the widgets of the plugins for the home page.
package widget

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// Deadline is how long the hub waits for a plugin to answer one widget.
const Deadline = 5 * time.Second

// Reader calls the Load of a widget and checks its answer. It keeps no answer.
type Reader struct {
	deadline time.Duration
}

// NewReader creates a Reader that waits for each plugin up to the deadline.
func NewReader(deadline time.Duration) *Reader {
	return &Reader{deadline: deadline}
}

type outcome struct {
	answer pluginapi.WidgetAnswer
	err    error
}

// Read answers the data of the widget for the acting user of the context. A plugin
// that ignores its context still ends the read at the deadline, and its goroutine
// ends alone when Load returns.
func (r *Reader) Read(ctx context.Context, target registry.Widget) (pluginapi.WidgetAnswer, error) {
	ctx, cancel := context.WithTimeout(ctx, r.deadline)
	defer cancel()

	done := make(chan outcome, 1)

	go func() {
		// A panic in a goroutine stops the process, and no recoverer of a route
		// reaches it. Load sent nothing, so the buffer of one takes this outcome.
		defer func() {
			if recovered := recover(); recovered != nil {
				done <- outcome{err: fmt.Errorf(
					"%w: plugin %s, widget %s: %v",
					errs.ErrWidgetPanicked, target.PluginID, target.Meta.Key, recovered,
				)}
			}
		}()

		done <- load(ctx, target)
	}()

	select {
	case <-ctx.Done():
		return pluginapi.WidgetAnswer{}, fmt.Errorf(
			"%w: plugin %s, widget %s", errs.ErrWidgetTimeout, target.PluginID, target.Meta.Key,
		)
	case result := <-done:
		if result.err != nil {
			return pluginapi.WidgetAnswer{}, result.err
		}

		if err := check(target.Meta.Kind, result.answer); err != nil {
			return pluginapi.WidgetAnswer{}, fmt.Errorf(
				"%w: plugin %s, widget %s: %s",
				errs.ErrInvalidWidgetAnswer, target.PluginID, target.Meta.Key, err.reason,
			)
		}

		return result.answer, nil
	}
}

func load(ctx context.Context, target registry.Widget) outcome {
	answer, err := target.Load(ctx)
	if err != nil {
		return outcome{err: fmt.Errorf(
			"loading the widget %s of plugin %s: %w", target.Meta.Key, target.PluginID, err,
		)}
	}

	return outcome{answer: answer}
}

type violation struct {
	reason string
}

func check(kind pluginapi.WidgetKind, answer pluginapi.WidgetAnswer) *violation {
	switch answer.State {
	case pluginapi.WidgetStateReady:
		return checkReady(kind, answer)
	case pluginapi.WidgetStateEmpty:
		if answer.Message == "" {
			return &violation{"empty with no message"}
		}

		return nil
	case pluginapi.WidgetStateNotSetUp:
		if answer.Message == "" || answer.Link == "" {
			return &violation{"not set up with no message or no link"}
		}

		return checkLinks(answer.Link)
	default:
		return &violation{"the state " + string(answer.State)}
	}
}

func checkReady(kind pluginapi.WidgetKind, answer pluginapi.WidgetAnswer) *violation {
	switch kind {
	case pluginapi.WidgetKindStat:
		return checkStat(answer)
	case pluginapi.WidgetKindList:
		return checkList(answer)
	default:
		return &violation{"the kind " + string(kind)}
	}
}

func checkStat(answer pluginapi.WidgetAnswer) *violation {
	if answer.Stat == nil || answer.List != nil || answer.Stat.Value == "" {
		return &violation{"a stat with no value"}
	}

	return checkLinks(answer.Stat.Link)
}

func checkList(answer pluginapi.WidgetAnswer) *violation {
	if answer.List == nil || answer.Stat != nil {
		return &violation{"a list with no items"}
	}

	links := []string{answer.List.More}

	for _, item := range answer.List.Items {
		if item.Label == "" {
			return &violation{"an item with no label"}
		}

		links = append(links, item.Link)
	}

	return checkLinks(links...)
}

// checkLinks keeps every link a path of the plugin, so a widget never sends the
// person to another site. An empty link is no link.
func checkLinks(links ...string) *violation {
	for _, link := range links {
		if link == "" {
			continue
		}

		if !strings.HasPrefix(link, "/") ||
			strings.HasPrefix(link, "//") ||
			strings.HasPrefix(link, `/\`) ||
			slices.Contains(strings.Split(link, "/"), "..") {
			return &violation{"the link " + link}
		}
	}

	return nil
}
