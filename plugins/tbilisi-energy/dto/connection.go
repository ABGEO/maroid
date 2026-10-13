package dto

import "github.com/abgeo/maroid/plugins/tbilisi-energy/model"

// ConnectionResponse is the answer of a read of the connection.
type ConnectionResponse struct {
	State             string             `json:"state"`
	CredentialsStored bool               `json:"credentials_stored"`
	Customers         []CustomerResponse `json:"customers,omitzero"`
}

// CustomerResponse is one customer of the gas account. It holds no personal number.
type CustomerResponse struct {
	CustomerNumber string `json:"customer_number"`
	Name           string `json:"name"`
	Address        string `json:"address"`
	Assigned       bool   `json:"assigned"`
}

// AssignmentRequest names the customer to assign to the workspace.
type AssignmentRequest struct {
	CustomerNumber string `json:"customer_number"`
}

// NewConnectionResponse converts the connection to its answer. A connected answer
// always carries the list of the customers, even an empty one.
func NewConnectionResponse(connection *model.Connection) ConnectionResponse {
	response := ConnectionResponse{
		State:             string(connection.State),
		CredentialsStored: connection.CredentialsStored,
	}

	if connection.State != model.ConnectionStateConnected {
		return response
	}

	response.Customers = make([]CustomerResponse, 0, len(connection.Customers))

	for _, customer := range connection.Customers {
		response.Customers = append(response.Customers, CustomerResponse{
			CustomerNumber: customer.Number,
			Name:           customer.Name,
			Address:        customer.Address,
			Assigned:       customer.Assigned,
		})
	}

	return response
}
