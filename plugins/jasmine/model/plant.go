package model

import "time"

// Plant represents one plant that a workspace keeps.
type Plant struct {
	ID             string               `db:"id"`
	Number         int                  `db:"number"`
	CommonName     string               `db:"common_name"`
	ScientificName *string              `db:"scientific_name"`
	Environment    EnvironmentReference `db:"environment"`
	AcquiredOn     *time.Time           `db:"acquired_on"`
	CareNotes      *string              `db:"care_notes"`
	Status         PlantStatus          `db:"status"`
	CreatedAt      time.Time            `db:"created_at"`
	UpdatedAt      time.Time            `db:"updated_at"`
}

// DisplayID answers the identifier that a person reads for the plant.
func (p *Plant) DisplayID() string {
	return FormatDisplayID(PrefixPlant, p.Number)
}
