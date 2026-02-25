package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
)

type sensorReading struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}

// MeasurementSubscriber handles measurement messages published by flora-node devices.
type MeasurementSubscriber struct {
	logger *slog.Logger
}

var _ pluginapi.MQTTSubscriber = (*MeasurementSubscriber)(nil)

// NewMeasurementSubscriber creates a new MeasurementSubscriber.
func NewMeasurementSubscriber(logger *slog.Logger) *MeasurementSubscriber {
	return &MeasurementSubscriber{
		logger: logger.With(slog.String("subscriber", "measurement")),
	}
}

// Meta returns the subscriber metadata.
func (s *MeasurementSubscriber) Meta() pluginapi.MQTTSubscriberMeta {
	return pluginapi.MQTTSubscriberMeta{
		ID:    "measurement",
		Topic: "plant/+/measurement/+",
		QoS:   1,
	}
}

// Handle processes an incoming measurement message.
func (s *MeasurementSubscriber) Handle(_ context.Context, topic string, payload []byte) error {
	parts := strings.Split(topic, "/")
	metricType := parts[3]
	plantID := parts[1]

	var reading sensorReading
	if err := json.Unmarshal(payload, &reading); err != nil {
		return fmt.Errorf("parsing payload: %w", err)
	}

	s.logger.Info(
		"measurement received",
		slog.String("plant_id", plantID),
		slog.String("metric_type", metricType),
		slog.Float64("value", reading.Value),
		slog.Time("time", reading.Time),
	)

	return nil
}
