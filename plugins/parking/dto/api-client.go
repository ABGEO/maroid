// Package dto defines data transfer objects used for API requests and responses.
package dto

import (
	"encoding/json"

	"github.com/abgeo/maroid/plugins/parking/model"
)

// APIResponse is a generic API response wrapper.
type APIResponse[T any] struct {
	//nolint:tagliatelle
	Result struct {
		DataType string `json:"dataType"`
		Data     T      `json:"data"`
	} `json:"result"`
}

// ParkingLot represents a single parking lot returned from the API.
//
//nolint:tagliatelle
type ParkingLot struct {
	Polygon      string `json:"polygon"`
	Address      string `json:"address"`
	UniqueNumber string `json:"uniqueNumber"`
}

// ParkingPlace represents a single parking place returned from the API.
//
//nolint:tagliatelle
type ParkingPlace struct {
	ID           int    `json:"id"`
	UniqueNumber string `json:"uniqueNumber"`
	Address      string `json:"address"`
	FreeParking  bool   `json:"freeParking"`
}

// APIRequest is a generic API request wrapper.
type APIRequest[T any] struct {
	Data T `json:"data"`
}

// StartParkingData holds the fields for a start-parking request.
//
//nolint:tagliatelle
type StartParkingData struct {
	PlaceNo   string `json:"placeNo"`
	VehicleID int    `json:"vehicleId"`
	Type      string `json:"type"`
}

// Person represents the authenticated person's profile from the API.
//
//nolint:tagliatelle
type Person struct {
	FirstName            string  `json:"firstName"`
	LastName             string  `json:"lastName"`
	BalanceAmount        float64 `json:"balanceAmount"`
	DailyFreeParkingLeft int     `json:"dailyFreeParkingLeft"`
}

// ActiveSession represents the current parking session from GET /parking.
//
//nolint:tagliatelle
type ActiveSession struct {
	ID                 int        `json:"id"`
	Difference         int        `json:"difference"`
	ParkingType        string     `json:"parkingType"`
	IncludeFreeParking bool       `json:"includeFreeParking"`
	FixedAmount        float64    `json:"fixedAmount"`
	ParkingPlace       ParkingLot `json:"parkingPlace"`
	StartDate          string     `json:"startDate"`
	EndDate            *string    `json:"endDate"`
}

// ParkingSession represents an active parking session returned from the API.
//
//nolint:tagliatelle
type ParkingSession struct {
	ID                 int        `json:"id"`
	IncludeFreeParking bool       `json:"includeFreeParking"`
	StartDate          string     `json:"startDate"`
	EndDate            *string    `json:"endDate"`
	Status             string     `json:"status"`
	ParkingPlace       ParkingLot `json:"parkingPlace"`
}

// SignInData holds the credentials of a sign in.
//
//nolint:tagliatelle
type SignInData struct {
	PrivateLogin bool   `json:"privateLogin"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

// SignInResult is the answer of a sign in.
//
//nolint:tagliatelle
type SignInResult struct {
	UUID   string `json:"UUID"`
	Status string `json:"status"`
	TwoFA  struct {
		PhoneVerification bool `json:"PHONE_VERIFICATION"`
	} `json:"TWO_FA"`
}

// CodeRequestData names the sign in whose code the parking service sends.
//
//nolint:tagliatelle
type CodeRequestData struct {
	TargetType string `json:"targetType"`
	UUID       string `json:"uuid"`
}

// CodeRequestResult is the answer of a request of a code. Delay holds the seconds
// before the next request, which the parking service sends as a string.
type CodeRequestResult struct {
	Delay  json.Number `json:"delay"`
	Phone  string      `json:"phone"`
	Status string      `json:"status"`
}

// CodeSubmitData carries the code of a sign in.
//
//nolint:tagliatelle
type CodeSubmitData struct {
	TargetType string `json:"targetType"`
	UUID       string `json:"uuid"`
	Token      string `json:"token"`
}

// CodeSubmitResult is the answer of a code. The session token travels in a header.
type CodeSubmitResult struct {
	Status string `json:"status"`
}

// Vehicle is one vehicle of the parking account.
//
//nolint:tagliatelle
type Vehicle struct {
	ID        int    `json:"id"`
	GovNumber string `json:"govNumber"`
	Name      string `json:"name"`
}

// ToModel converts the vehicle to its domain entity.
func (v Vehicle) ToModel() model.Vehicle {
	return model.Vehicle{
		ID:    v.ID,
		Name:  v.Name,
		Plate: v.GovNumber,
	}
}
