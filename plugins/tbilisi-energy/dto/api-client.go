package dto

// TransactionsRequest represents a request to fetch transactions for a customer.
type TransactionsRequest struct {
	CustomerNumber string `json:"customerNumber"`
	DateFrom       string `json:"dateFrom,omitempty"`
	DateTo         string `json:"dateTo,omitempty"`
}

// BaseResponse represents the common fields returned in most API responses.
type BaseResponse struct {
	ID      int    `json:"id"      mapstructure:"id"`
	Success bool   `json:"success" mapstructure:"success"`
	Message string `json:"message" mapstructure:"message"`
	Code    int    `json:"code"    mapstructure:"code"`
}

// ErrorResponse represents an error returned by the API.
type ErrorResponse struct {
	Type    string         `json:"type"    mapstructure:"type"`
	Title   string         `json:"title"   mapstructure:"title"`
	Status  int            `json:"status"  mapstructure:"status"`
	TraceID string         `json:"traceId" mapstructure:"traceId"`
	Errors  map[string]any `json:"errors"  mapstructure:"errors"`
}

// AuthResponse represents the response returned after authentication.
type AuthResponse struct {
	BaseResponse

	RecaptchaRequested bool   `json:"recaptchaRequested"`
	Token              string `json:"token"`
}

// TransactionsResponse represents the response returned when fetching transactions.
type TransactionsResponse struct {
	BaseResponse

	Transactions []Transaction `json:"transactions"`
	PdfURL       string        `json:"pdfUrl"`
}

// Transaction represents a single transaction record.
type Transaction struct {
	Consumption            float64 `json:"consumption"`
	Amount                 float64 `json:"amount"`
	MeterReading           float64 `json:"meterReading"`
	Balance                float64 `json:"balance"`
	OperationID            int     `json:"operId"`
	OperationName          string  `json:"operationName"`
	OperationDate          string  `json:"operDate"`
	OperationDateTimestamp int     `json:"operDateTimeStump"`
	OperationDateString    string  `json:"operDateString"`
	BillingDocumentURL     string  `json:"billDocPath"`
	MeterPhotoURL          string  `json:"meterPhotoPath"`
}
