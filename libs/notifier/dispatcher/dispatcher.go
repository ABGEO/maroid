// Package dispatcher provides functionality to route notification messages
// to multiple channels and transports with automatic failover support.
package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"github.com/abgeo/maroid/libs/notifier"
	"github.com/abgeo/maroid/libs/notifier/registry"
	"github.com/abgeo/maroid/libs/notifierapi"
)

var (
	// ErrChannelNotFound is returned when attempting to send a message
	// to a channel that does not exist in the configuration.
	ErrChannelNotFound = errors.New("channel not found")
	// ErrTransportsFailed is returned when all configured transports
	// for a channel fail to deliver a message.
	ErrTransportsFailed = errors.New("all transports failed")
	// ErrTransportNotFound is returned when a channel references a
	// transport that does not exist or is disabled.
	ErrTransportNotFound = errors.New("transport not found or disabled")
	// ErrNoTransports is returned when a channel or operation has
	// no transports configured for message delivery.
	ErrNoTransports = errors.New("no transports configured")
)

// ChannelDispatcher dispatches notification messages to configured channels
// with automatic transport failover support.
type ChannelDispatcher struct {
	logger *slog.Logger

	transports map[string]notifierapi.Transport
	channels   map[string]notifier.ChannelConfig
}

var _ notifierapi.Dispatcher = (*ChannelDispatcher)(nil)

// NewDispatcher creates a ChannelDispatcher from the provided configuration and registry.
// It initializes all enabled transports and validates channel references.
func NewDispatcher(
	cfg *notifier.Config,
	logger *slog.Logger,
	reg registry.Registry,
) (*ChannelDispatcher, error) {
	transports, err := buildTransports(cfg.Transports, reg, logger)
	if err != nil {
		return nil, err
	}

	if err := validateChannels(cfg.Channels, transports); err != nil {
		return nil, err
	}

	logger = logger.With(
		slog.String("component", "notifier-dispatcher"),
	)

	return &ChannelDispatcher{
		logger:     logger,
		transports: transports,
		channels:   cfg.Channels,
	}, nil
}

// Send delivers a message to the specified channel. It attempts all primary
// transports first, then falls back to fallback transports if all primaries fail.
// Returns an error only if all transports fail or if the channel doesn't exist.
func (d *ChannelDispatcher) Send(
	ctx context.Context,
	channelName string,
	msg notifierapi.Message,
) error {
	var (
		primaryErr  error
		fallbackErr error
	)

	logger := d.logger.With(
		slog.String("channel", channelName),
	)

	channel, exists := d.channels[channelName]
	if !exists {
		return fmt.Errorf("%w: %q", ErrChannelNotFound, channelName)
	}

	logger.Debug(
		"sending message",
		slog.Any("primary_transports", channel.Transports),
		slog.Any("fallback_transports", channel.Fallback),
	)

	primaryErr = d.tryTransports(ctx, channel.Transports, msg)
	if primaryErr == nil {
		return nil
	}

	logger.Error(
		"all primary transports failed",
		slog.Any("error", primaryErr),
	)

	if len(channel.Fallback) > 0 {
		fallbackErr = d.tryTransports(ctx, channel.Fallback, msg)
		if fallbackErr == nil {
			return nil
		}

		logger.Error(
			"all fallback transports failed",
			slog.Any("error", fallbackErr),
		)
	}

	d.logger.Error("all transports failed")

	return fmt.Errorf("%w for channel %q: %w",
		ErrTransportsFailed,
		channelName,
		errors.Join(primaryErr, fallbackErr),
	)
}

// Channels returns a sorted slice of all configured channel names.
func (d *ChannelDispatcher) Channels() []string {
	return slices.Sorted(maps.Keys(d.channels))
}

func buildTransports(
	configs map[string]notifier.TransportConfig,
	reg registry.Registry,
	logger *slog.Logger,
) (map[string]notifierapi.Transport, error) {
	transports := make(map[string]notifierapi.Transport, len(configs))

	for name, cfg := range configs {
		if !cfg.Enabled {
			logger.Debug("skipping disabled transport", "transport", name)

			continue
		}

		transport, err := reg.New(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to create transport %q: %w", name, err)
		}

		logger.Debug("transport created", "transport", name)
		transports[name] = transport
	}

	return transports, nil
}

func validateChannels(
	channels map[string]notifier.ChannelConfig,
	transports map[string]notifierapi.Transport,
) error {
	for channelName, channel := range channels {
		allTransports := slices.Concat(channel.Transports, channel.Fallback)

		for _, transportName := range allTransports {
			if _, exists := transports[transportName]; !exists {
				return fmt.Errorf(
					"channel %q: %w %q",
					channelName,
					ErrTransportNotFound,
					transportName,
				)
			}
		}
	}

	return nil
}

func (d *ChannelDispatcher) tryTransports(
	ctx context.Context,
	names []string,
	msg notifierapi.Message,
) error {
	if len(names) == 0 {
		return ErrNoTransports
	}

	var errs []error

	for _, name := range names {
		transport, ok := d.transports[name]
		if !ok {
			d.logger.Warn("unknown transport reference", slog.String("transport", name))

			continue
		}

		d.logger.Debug("sending via transport", slog.String("transport", name))

		if err := transport.Send(ctx, msg); err != nil {
			d.logger.Error(
				"transport failed",
				slog.String("transport", name),
				slog.Any("error", err),
			)
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
	}

	return errors.Join(errs...)
}
