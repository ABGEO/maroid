package worker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// MQTTWorker manages the MQTT broker connection and dispatches
// incoming messages to registered subscribers.
type MQTTWorker struct {
	logger   *slog.Logger
	cfg      *config.Config
	registry *registry.MQTTSubscriberRegistry

	client mqtt.Client
}

var _ Worker = (*MQTTWorker)(nil)

// NewMQTTWorker creates a new MQTTWorker.
func NewMQTTWorker(
	logger *slog.Logger,
	cfg *config.Config,
	registry *registry.MQTTSubscriberRegistry,
) *MQTTWorker {
	return &MQTTWorker{
		logger: logger.With(
			slog.String("component", "worker"),
			slog.String("worker", "mqtt"),
		),
		cfg:      cfg,
		registry: registry,
	}
}

// Name returns the worker type identifier.
func (w *MQTTWorker) Name() string { return "mqtt" }

// Prepare validates that the broker is configured when subscribers are registered.
func (w *MQTTWorker) Prepare() error {
	if len(w.registry.All()) > 0 && w.cfg.MQTT.Broker == "" {
		return fmt.Errorf(
			"%w: subscribers are registered but mqtt.broker is not set",
			errs.ErrMQTTBrokerNotConfigured,
		)
	}

	return nil
}

// Start connects to the MQTT broker and subscribes all registered handlers.
// It is a no-op if no subscribers are registered.
func (w *MQTTWorker) Start(ctx context.Context) error {
	var err error

	if len(w.registry.All()) == 0 {
		w.logger.InfoContext(ctx, "no MQTT subscribers registered, skipping")

		return nil
	}

	w.client, err = w.connect()
	if err != nil {
		return err
	}

	w.logger.InfoContext(ctx, "connected to MQTT broker", slog.String("broker", w.cfg.MQTT.Broker))

	if err = w.subscribe(ctx); err != nil {
		return err
	}

	<-ctx.Done()

	return nil
}

// Stop disconnects from the MQTT broker.
func (w *MQTTWorker) Stop(ctx context.Context) error {
	if w.client == nil {
		return nil
	}

	w.logger.InfoContext(ctx, "disconnecting from MQTT broker")
	w.client.Disconnect(w.cfg.MQTT.DisconnectQuiesce)

	return nil
}

func (w *MQTTWorker) connect() (mqtt.Client, error) {
	cfg := w.cfg.MQTT

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("getting hostname: %w", err)
	}

	opts := mqtt.NewClientOptions().
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectTimeout(cfg.ConnectTimeout).
		AddBroker(cfg.Broker).
		SetClientID(fmt.Sprintf("%s@%s", cfg.ClientIDPrefix, hostname)).
		SetUsername(cfg.User).
		SetPassword(cfg.Password)
	client := mqtt.NewClient(opts)

	token := client.Connect()
	if token.WaitTimeout(cfg.ConnectTimeout) && token.Error() != nil {
		return nil, fmt.Errorf("connecting to MQTT broker %s: %w", cfg.Broker, token.Error())
	}

	return client, nil
}

func (w *MQTTWorker) subscribe(ctx context.Context) error {
	for effectiveTopic, sub := range w.registry.All() {
		// Derive the namespace prefix: effectiveTopic minus the relative topic suffix.
		// e.g. "dev/maroid/jasmine/measurement/+/+" → "dev/maroid/jasmine"
		namespace := strings.TrimSuffix(effectiveTopic, "/"+sub.Meta().Topic)
		subscribeTopic := w.buildSubscribeTopic(effectiveTopic)

		token := w.client.Subscribe(
			subscribeTopic,
			sub.Meta().QoS,
			//nolint:contextcheck // paho MessageHandler signature provides no context parameter
			w.makeHandler(namespace, sub),
		)
		if token.WaitTimeout(w.cfg.MQTT.ConnectTimeout) && token.Error() != nil {
			return fmt.Errorf("subscribing to topic %s: %w", subscribeTopic, token.Error())
		}

		w.logger.InfoContext(ctx,
			"subscribed to MQTT topic",
			slog.String("subscriber_id", sub.Meta().ID),
			slog.String("effective_topic", effectiveTopic),
			slog.String("subscribe_topic", subscribeTopic),
			slog.Int("qos", int(sub.Meta().QoS)),
		)
	}

	return nil
}

// buildSubscribeTopic wraps the effective topic in a shared subscription if configured.
func (w *MQTTWorker) buildSubscribeTopic(effectiveTopic string) string {
	if w.cfg.MQTT.SharedGroup == "" {
		return effectiveTopic
	}

	return fmt.Sprintf("$share/%s/%s", w.cfg.MQTT.SharedGroup, effectiveTopic)
}

// makeHandler returns a paho MessageHandler that strips the namespace prefix and
// dispatches to the subscriber in a goroutine to avoid blocking the MQTT receive loop.
func (w *MQTTWorker) makeHandler(
	namespace string,
	sub pluginapi.MQTTSubscriber,
) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		go func() {
			relativeTopic := strings.TrimPrefix(msg.Topic(), namespace+"/")

			logger := w.logger.With(
				slog.String("subscriber_id", sub.Meta().ID),
				slog.String("topic", msg.Topic()),
				slog.String("relative_topic", relativeTopic),
			)

			if err := sub.Handle(context.Background(), relativeTopic, msg.Payload()); err != nil {
				logger.Error("mqtt subscriber handle error", slog.Any("error", err))
			}
		}()
	}
}
