// Package dto defines data transfer objects used for API requests and responses.
package dto

// APIResponse is a generic API response wrapper.
type APIResponse[T any] struct {
	Result struct {
		DataType string `json:"dataType"`
		Data     T      `json:"data"`
	} `json:"result"`
}

// ParkingLot represents a single parking lot returned from the API.
type ParkingLot struct {
	Polygon      string `json:"polygon"`
	Address      string `json:"address"`
	UniqueNumber string `json:"uniqueNumber"`
}

// ParkingPlace represents a single parking place returned from the API.
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
type StartParkingData struct {
	PlaceNo   string `json:"placeNo"`
	VehicleID int    `json:"vehicleId"`
	Type      string `json:"type"`
}

// Person represents the authenticated person's profile from the API.
type Person struct {
	FirstName            string  `json:"firstName"`
	LastName             string  `json:"lastName"`
	BalanceAmount        float64 `json:"balanceAmount"`
	DailyFreeParkingLeft int     `json:"dailyFreeParkingLeft"`
}

// ActiveSession represents the current parking session from GET /parking.
type ActiveSession struct {
	ID                 int        `json:"id"`
	Difference         int        `json:"difference"`
	ParkingType        string     `json:"parkingType"`
	IncludeFreeParking bool       `json:"includeFreeParking"`
	FixedAmount        int        `json:"fixedAmount"`
	ParkingPlace       ParkingLot `json:"parkingPlace"`
	StartDate          string     `json:"startDate"`
	EndDate            *string    `json:"endDate"`
}

// ParkingSession represents an active parking session returned from the API.
type ParkingSession struct {
	ID                 int        `json:"id"`
	IncludeFreeParking bool       `json:"includeFreeParking"`
	StartDate          string     `json:"startDate"`
	EndDate            *string    `json:"endDate"`
	Status             string     `json:"status"`
	ParkingPlace       ParkingLot `json:"parkingPlace"`
}
