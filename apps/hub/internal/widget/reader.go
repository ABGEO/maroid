// Package widget reads the widgets of the plugins for the home page.
package widget

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
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

// New creates a Reader that waits for each plugin up to the deadline.
func New(deadline time.Duration) *Reader {
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
		// reaches it. Load sent nothing, so the buffer of one takes this outcome. The
		// stack exists only in this goroutine, so the error carries it.
		defer func() {
			if recovered := recover(); recovered != nil {
				done <- outcome{err: fmt.Errorf(
					"%w: plugin %s, widget %s: %v\n%s",
					errs.ErrWidgetPanicked, target.PluginID, target.Meta.Key, recovered, debug.Stack(),
				)}
			}
		}()

		done <- load(ctx, target)
	}()

	select {
	case <-ctx.Done():
		return pluginapi.WidgetAnswer{}, ended(ctx, target)
	case result := <-done:
		// A plugin that honours its context returns at the same moment that the
		// context ends, and the select picks either case.
		if result.err != nil && ctx.Err() != nil {
			return pluginapi.WidgetAnswer{}, ended(ctx, target)
		}

		if result.err != nil {
			return pluginapi.WidgetAnswer{}, result.err
		}

		if err := check(target.Meta.Kind, result.answer); err != nil {
			return pluginapi.WidgetAnswer{}, fmt.Errorf(
				"%w: plugin %s, widget %s: %w",
				errs.ErrInvalidWidgetAnswer, target.PluginID, target.Meta.Key, err,
			)
		}

		return result.answer, nil
	}
}

// ended tells a deadline from a request that the client gave up.
func ended(ctx context.Context, target registry.Widget) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf(
			"%w: plugin %s, widget %s", errs.ErrWidgetTimeout, target.PluginID, target.Meta.Key,
		)
	}

	return fmt.Errorf(
		"reading the widget %s of plugin %s: %w", target.Meta.Key, target.PluginID, ctx.Err(),
	)
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

var (
	errNoMessage = errors.New("no message")
	errNoLink    = errors.New("not set up with no link")
	errState     = errors.New("unknown state")
	errKind      = errors.New("unknown kind")
	errStat      = errors.New("ready with no stat, an empty value, or a list")
	errList      = errors.New("ready with no list, or with a stat")
	errLabel     = errors.New("an item with no label")
	errLink      = errors.New("a link outside the plugin")
)

func check(kind pluginapi.WidgetKind, answer pluginapi.WidgetAnswer) error {
	switch answer.State {
	case pluginapi.WidgetStateReady:
		return checkReady(kind, answer)
	case pluginapi.WidgetStateEmpty:
		if answer.Message == "" {
			return errNoMessage
		}

		return nil
	case pluginapi.WidgetStateNotSetUp:
		switch {
		case answer.Message == "":
			return errNoMessage
		case answer.Link == "":
			return errNoLink
		}

		return checkLinks(answer.Link)
	default:
		return fmt.Errorf("%w: %s", errState, answer.State)
	}
}

func checkReady(kind pluginapi.WidgetKind, answer pluginapi.WidgetAnswer) error {
	switch kind {
	case pluginapi.WidgetKindStat:
		return checkStat(answer)
	case pluginapi.WidgetKindList:
		return checkList(answer)
	default:
		return fmt.Errorf("%w: %s", errKind, kind)
	}
}

func checkStat(answer pluginapi.WidgetAnswer) error {
	if answer.Stat == nil || answer.List != nil || answer.Stat.Value == "" {
		return errStat
	}

	return checkLinks(answer.Stat.Link)
}

func checkList(answer pluginapi.WidgetAnswer) error {
	if answer.List == nil || answer.Stat != nil {
		return errList
	}

	links := []string{answer.List.More}

	for _, item := range answer.List.Items {
		if item.Label == "" {
			return errLabel
		}

		links = append(links, item.Link)
	}

	return checkLinks(links...)
}

// checkLinks keeps every link a path of the plugin, so a widget never sends the
// person to another site. An empty link is no link.
func checkLinks(links ...string) error {
	for _, link := range links {
		if link == "" {
			continue
		}

		if !strings.HasPrefix(link, "/") ||
			strings.HasPrefix(link, "//") ||
			strings.HasPrefix(link, `/\`) ||
			slices.Contains(strings.Split(link, "/"), "..") {
			return fmt.Errorf("%w: %s", errLink, link)
		}
	}

	return nil
}
