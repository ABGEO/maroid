package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/jasmine/model"
	"github.com/abgeo/maroid/plugins/jasmine/repository"
)

type sensorReading struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}

// MeasurementSubscriber handles measurement messages published by flora-node devices.
type MeasurementSubscriber struct {
	logger *slog.Logger
	db     *pluginapi.PluginDB
}

var _ pluginapi.MQTTSubscriber = (*MeasurementSubscriber)(nil)

// NewMeasurementSubscriber creates a new MeasurementSubscriber.
func NewMeasurementSubscriber(logger *slog.Logger, db *pluginapi.PluginDB) *MeasurementSubscriber {
	return &MeasurementSubscriber{
		logger: logger.With(slog.String("subscriber", "measurement")),
		db:     db,
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
func (s *MeasurementSubscriber) Handle(ctx context.Context, topic string, payload []byte) error {
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

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		repo := repository.NewMeasurement(tx)

		return repo.Insert(ctx, &model.Measurement{
			Time:       reading.Time,
			PlantID:    plantID,
			MetricType: metricType,
			Value:      reading.Value,
		})
	})
	if err != nil {
		return fmt.Errorf("saving measurement: %w", err)
	}

	return nil
}
