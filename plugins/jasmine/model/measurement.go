package model

import "time"

// Measurement represents a single sensor reading.
type Measurement struct {
	Time       time.Time  `db:"time"`
	SourceType SourceType `db:"source_type"`
	SourceID   string     `db:"source_id"`
	MetricType string     `db:"metric_type"`
	Value      float64    `db:"value"`
}
