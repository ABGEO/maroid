package model

// VolumeUnit names the unit of a volume of fuel.
type VolumeUnit string

const (
	// VolumeLitre is the litre.
	VolumeLitre VolumeUnit = "l"
	// VolumeGallonUS is the gallon of the United States.
	VolumeGallonUS VolumeUnit = "gal_us"
	// VolumeGallonImperial is the imperial gallon.
	VolumeGallonImperial VolumeUnit = "gal_imp"
)

// ConsumptionShape names how a consumption reads: a distance per volume, as km/L, or a
// volume per 100 of the distance, as L/100 km.
type ConsumptionShape string

const (
	// ShapeDistancePerVolume reads as km/L or mpg.
	ShapeDistancePerVolume ConsumptionShape = "distance_per_volume"
	// ShapeVolumePer100 reads as L/100 km.
	ShapeVolumePer100 ConsumptionShape = "volume_per_100"
)

// The bounds of a consumption format of the car log service.
const (
	FirstConsumptionFormat ConsumptionFormat = 1
	LastConsumptionFormat  ConsumptionFormat = 12
)

// ConsumptionFormat is the consumption format of the car log settings, 1 to 12. The
// formats 1 to 3 and 7 to 9 read as a distance per volume. The formats 4 to 6 and 10 to 12
// read as a volume per 100 of the distance.
type ConsumptionFormat int

// Valid reports whether the format is one that the car log service names.
func (f ConsumptionFormat) Valid() bool {
	return f >= FirstConsumptionFormat && f <= LastConsumptionFormat
}

// Shape answers how a consumption of the format reads.
func (f ConsumptionFormat) Shape() ConsumptionShape {
	const formatsOfOneShape = 3

	if (int(f)-1)/formatsOfOneShape%2 == 1 {
		return ShapeVolumePer100
	}

	return ShapeDistancePerVolume
}

// CarLogSettings holds the units and the currency of the car log account.
type CarLogSettings struct {
	VolumeUnit        VolumeUnit        `db:"volume_unit"`
	ConsumptionFormat ConsumptionFormat `db:"consumption_format"`
	Currency          string            `db:"currency"`
}
