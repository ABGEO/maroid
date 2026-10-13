package dto

import (
	"time"

	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
)

// TransactionResponse is one transaction of the transactions page.
type TransactionResponse struct {
	ID              string   `json:"id"`
	TransactionDate string   `json:"transaction_date"`
	Operation       string   `json:"operation"`
	Amount          float64  `json:"amount"`
	Balance         float64  `json:"balance"`
	Usage           *float64 `json:"usage,omitempty"`
	MeterValue      *float64 `json:"meter_value,omitempty"`
	BillURL         string   `json:"bill_url,omitempty"`
	MeterPhotoURL   string   `json:"meter_photo_url,omitempty"`
}

// NewTransactionResponseList converts the transactions to their answers. fileBase is
// the address of the gas service that a stored path of a file follows.
func NewTransactionResponseList(
	transactions []model.Transaction,
	fileBase string,
) []TransactionResponse {
	responses := make([]TransactionResponse, 0, len(transactions))

	for _, transaction := range transactions {
		response := TransactionResponse{
			ID:              transaction.ID,
			TransactionDate: transaction.Date.Format(time.DateOnly),
			Operation:       transaction.Operation,
			Amount:          transaction.Amount,
			Balance:         transaction.Balance,
			BillURL:         fileURL(fileBase, transaction.BillPath),
			MeterPhotoURL:   fileURL(fileBase, transaction.MeterPhotoPath),
		}

		if transaction.IsReading() {
			usage := transaction.Consumption
			response.Usage = &usage
		}

		if transaction.MeterValue > 0 {
			meterValue := transaction.MeterValue
			response.MeterValue = &meterValue
		}

		responses = append(responses, response)
	}

	return responses
}

func fileURL(fileBase string, path *string) string {
	if path == nil {
		return ""
	}

	return fileBase + *path
}
