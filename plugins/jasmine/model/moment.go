package model

import "time"

// PhotoKind is the key of the file kind of a photo of a plant.
const PhotoKind = "plant-photo"

// Moment represents one entry of the gallery of a plant: a day, an optional note,
// and the photos of that day.
type Moment struct {
	ID        string    `db:"id"`
	PlantID   string    `db:"plant_id"`
	TakenOn   time.Time `db:"taken_on"`
	Note      *string   `db:"note"`
	Photos    []Photo   `db:"-"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Photo represents one file of the plant photo kind that a moment holds.
type Photo struct {
	ID        string    `db:"id"`
	MomentID  string    `db:"moment_id"`
	FileID    string    `db:"file_id"`
	CreatedAt time.Time `db:"created_at"`
}
