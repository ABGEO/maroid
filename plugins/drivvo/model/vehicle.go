package model

// DistanceUnit names the unit of a distance and of an odometer value.
type DistanceUnit string

const (
	// DistanceKilometre is the kilometre.
	DistanceKilometre DistanceUnit = "km"
	// DistanceMile is the mile.
	DistanceMile DistanceUnit = "mi"
)

// Vehicle is one vehicle of the car log account.
type Vehicle struct {
	ID           string       `db:"id"`
	SourceID     int32        `db:"source_id"`
	Name         string       `db:"name"`
	Make         string       `db:"make"`
	Model        string       `db:"model"`
	Year         *int32       `db:"year"`
	DistanceUnit DistanceUnit `db:"distance_unit"`
	Principal    bool         `db:"principal"`
	Active       bool         `db:"active"`
}
