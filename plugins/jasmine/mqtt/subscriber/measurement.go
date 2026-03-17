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
		ID: "measurement",
		// {source_type}/{source_id}/measurement/{metric_type}
		Topic: "+/+/measurement/+",
		QoS:   1,
	}
}

// Handle processes an incoming measurement message.
func (s *MeasurementSubscriber) Handle(ctx context.Context, topic string, payload []byte) error {
	parts := strings.Split(topic, "/")
	source := parts[0]
	sourceID := parts[1]
	metricType := parts[3]

	sourceType, err := model.ParseSourceType(source)
	if err != nil {
		return fmt.Errorf("parsing source type: %w", err)
	}

	var reading sensorReading
	if err := json.Unmarshal(payload, &reading); err != nil {
		return fmt.Errorf("parsing payload: %w", err)
	}

	s.logger.Info(
		"measurement received",
		slog.String("source_type", string(sourceType)),
		slog.String("source_id", sourceID),
		slog.String("metric_type", metricType),
		slog.Float64("value", reading.Value),
		slog.Time("time", reading.Time),
	)

	err = s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		repo := repository.NewMeasurement(tx)

		return repo.Insert(ctx, &model.Measurement{
			Time:       reading.Time,
			SourceType: sourceType,
			SourceID:   sourceID,
			MetricType: metricType,
			Value:      reading.Value,
		})
	})
	if err != nil {
		return fmt.Errorf("saving measurement: %w", err)
	}

	return nil
}
