package dto

import "github.com/abgeo/maroid/plugins/drivvo/model"

// ConnectionResponse is the answer of a read of the connection.
type ConnectionResponse struct {
	State             string                   `json:"state"`
	CredentialsStored bool                     `json:"credentials_stored"`
	Vehicles          []ServiceVehicleResponse `json:"vehicles,omitzero"`
}

// ServiceVehicleResponse is one vehicle of the car log account, read from the service. It
// holds nothing of the account holder.
type ServiceVehicleResponse struct {
	Name   string `json:"name"`
	Make   string `json:"make"`
	Model  string `json:"model"`
	Active bool   `json:"active"`
}

// NewConnectionResponse converts the connection to its answer. A connected answer always
// carries the list of the vehicles, even an empty one.
func NewConnectionResponse(connection *model.Connection) ConnectionResponse {
	response := ConnectionResponse{
		State:             string(connection.State),
		CredentialsStored: connection.CredentialsStored,
	}

	if connection.State != model.ConnectionStateConnected {
		return response
	}

	response.Vehicles = make([]ServiceVehicleResponse, 0, len(connection.Vehicles))

	for _, vehicle := range connection.Vehicles {
		response.Vehicles = append(response.Vehicles, ServiceVehicleResponse{
			Name:   vehicle.Name,
			Make:   vehicle.Make,
			Model:  vehicle.Model,
			Active: vehicle.Active,
		})
	}

	return response
}
