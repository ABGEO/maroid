package dto

import "github.com/abgeo/maroid/plugins/gwp/model"

// ConnectionResponse is the answer of a read of the connection.
type ConnectionResponse struct {
	State             string             `json:"state"`
	CredentialsStored bool               `json:"credentials_stored"`
	Customers         []CustomerResponse `json:"customers,omitzero"`
}

// CustomerResponse is one customer of the water account.
type CustomerResponse struct {
	CustomerNumber string `json:"customer_number"`
	CommunalNumber string `json:"communal_number,omitempty"`
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
			CommunalNumber: customer.CommunalNumber,
			Address:        customer.Address,
			Assigned:       customer.Assigned,
		})
	}

	return response
}
