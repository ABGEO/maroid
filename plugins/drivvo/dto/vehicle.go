package dto

import (
	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// UnitsResponse names the units that a page shows for a vehicle.
type UnitsResponse struct {
	Distance         string `json:"distance"`
	Volume           string `json:"volume"`
	ConsumptionShape string `json:"consumption_shape"`
}

// VehicleResponse is one vehicle of the car log copy.
type VehicleResponse struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Make      string        `json:"make"`
	Model     string        `json:"model"`
	Year      *int32        `json:"year,omitempty"`
	Active    bool          `json:"active"`
	Principal bool          `json:"principal"`
	Default   bool          `json:"default"`
	Units     UnitsResponse `json:"units"`
}

// NewUnitsResponse answers the units of the vehicle under the settings.
func NewUnitsResponse(vehicle model.Vehicle, settings model.CarLogSettings) UnitsResponse {
	return UnitsResponse{
		Distance:         string(vehicle.DistanceUnit),
		Volume:           string(settings.VolumeUnit),
		ConsumptionShape: string(settings.ConsumptionFormat.Shape()),
	}
}

// NewVehicleResponseList converts the vehicles, and marks the default one.
func NewVehicleResponseList(
	vehicles []model.Vehicle,
	defaultID string,
	settings model.CarLogSettings,
) []VehicleResponse {
	answer := make([]VehicleResponse, 0, len(vehicles))

	for _, vehicle := range vehicles {
		answer = append(answer, VehicleResponse{
			ID:        vehicle.ID,
			Name:      vehicle.Name,
			Make:      vehicle.Make,
			Model:     vehicle.Model,
			Year:      vehicle.Year,
			Active:    vehicle.Active,
			Principal: vehicle.Principal,
			Default:   vehicle.ID == defaultID,
			Units:     NewUnitsResponse(vehicle, settings),
		})
	}

	return answer
}
