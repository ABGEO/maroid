package dto

import "github.com/abgeo/maroid/plugins/magti/model"

// ConnectionResponse is the answer of a read of the connection.
type ConnectionResponse struct {
	State             string               `json:"state"`
	CredentialsStored bool                 `json:"credentials_stored"`
	Subscribers       []SubscriberResponse `json:"subscribers,omitzero"`
}

// SubscriberResponse is one subscriber of the telecom account. It holds no name, no
// personal number, and no contact of the holder.
type SubscriberResponse struct {
	SubscriberNumber string `json:"subscriber_number"`
	Label            string `json:"label"`
	Kind             string `json:"kind"`
	Assigned         bool   `json:"assigned"`
}

// AssignmentRequest names the subscriber to assign to the workspace.
type AssignmentRequest struct {
	SubscriberNumber string `json:"subscriber_number"`
}

// NewConnectionResponse converts the connection to its answer. A connected answer
// always carries the list of the subscribers, even an empty one.
func NewConnectionResponse(connection *model.Connection) ConnectionResponse {
	response := ConnectionResponse{
		State:             string(connection.State),
		CredentialsStored: connection.CredentialsStored,
	}

	if connection.State != model.ConnectionStateConnected {
		return response
	}

	response.Subscribers = make([]SubscriberResponse, 0, len(connection.Subscribers))

	for _, subscriber := range connection.Subscribers {
		response.Subscribers = append(response.Subscribers, SubscriberResponse{
			SubscriberNumber: subscriber.Number,
			Label:            subscriber.Label,
			Kind:             string(subscriber.Kind),
			Assigned:         subscriber.Assigned,
		})
	}

	return response
}
