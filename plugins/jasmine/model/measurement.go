package model

import "time"

// Measurement represents a single sensor reading.
type Measurement struct {
	Time       time.Time `db:"time"`
	PlantID    string    `db:"plant_id"`
	MetricType string    `db:"metric_type"`
	Value      float64   `db:"value"`
}
