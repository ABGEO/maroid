package dto

// BaseResponse represents the base structure for API responses.
type BaseResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ListResponse represents a generic list response from the API.
type ListResponse[T any] struct {
	BaseResponse

	Items []T `json:"items"`
}

// AuthResponse represents the response returned after authentication.
type AuthResponse struct {
	BaseResponse

	Token          string `json:"token"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	Mobile         string `json:"mobile"`
	Email          string `json:"email"`
	PersonalNumber string `json:"personalNo"`
}

// Customer represents a customer entity in the API.
type Customer struct {
	ID                    int64   `json:"id"`
	CustomerNumber        string  `json:"customerNo"`
	UtilityCustomerNumber string  `json:"commCustomerNo"`
	City                  string  `json:"city"`
	District              string  `json:"district"`
	Street                string  `json:"street"`
	Building              string  `json:"building"`
	Balance               float64 `json:"balance"`
}

// ReadingResponse represents the response containing reading information.
type ReadingResponse struct {
	CustomerNumber string        `json:"customerNo"`
	Customer       string        `json:"name"`
	Address        string        `json:"address"`
	Items          []ReadingItem `json:"items"`
}

// ReadingItem represents a single reading item in the reading response.
type ReadingItem struct {
	Name            string  `json:"name"`
	Address         string  `json:"address"`
	MeterNumber     string  `json:"meterNum"`
	SerialNumber    string  `json:"serialNumber"`
	CustomerNumber  string  `json:"customerNo"`
	LastReadingDate string  `json:"lastReadingDate"`
	LastReading     float64 `json:"lastReading"`
	PreviousReading float64 `json:"prevReadValue"`
	MaxNumber       int     `json:"maxNumber"`
	SMSCode         string  `json:"smsCode"`
}
