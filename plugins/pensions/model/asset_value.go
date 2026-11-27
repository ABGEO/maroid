package model

import (
	"time"

	"github.com/google/uuid"
)

// AssetValue is the value of the pension account of a person on one date.
type AssetValue struct {
	ID         uuid.UUID `db:"id"`
	ValuedOn   time.Time `db:"valued_on"`
	AssetValue float64   `db:"asset_value"`
	Units      float64   `db:"units"`
	UnitPrice  float64   `db:"unit_price"`
}
