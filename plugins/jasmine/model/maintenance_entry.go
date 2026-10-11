package model

import "time"

// PlantReference names the plant of another record, with its environment.
type PlantReference struct {
	ID          string               `db:"id"`
	Number      int                  `db:"number"`
	CommonName  string               `db:"common_name"`
	Environment EnvironmentReference `db:"environment"`
}

// DisplayID answers the identifier that a person reads for the plant.
func (p *PlantReference) DisplayID() string {
	return FormatDisplayID(PrefixPlant, p.Number)
}

// MaintenanceEntry records one care action on one plant.
type MaintenanceEntry struct {
	ID          string          `db:"id"`
	Plant       PlantReference  `db:"plant"`
	Type        MaintenanceType `db:"type"`
	PerformedAt time.Time       `db:"performed_at"`
	Note        *string         `db:"note"`
	CreatedAt   time.Time       `db:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"`
}
