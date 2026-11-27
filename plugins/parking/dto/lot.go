package dto

import "github.com/abgeo/maroid/plugins/parking/model"

// LotUsageResponse counts the sessions of one parking lot.
type LotUsageResponse struct {
	LotNumber string        `json:"lot_number"`
	Address   string        `json:"address"`
	Count     int           `json:"count"`
	Amount    MoneyResponse `json:"amount"`
}

// NewLotUsageResponseList converts the parking lots to their answers.
func NewLotUsageResponseList(usages []model.LotUsage) []LotUsageResponse {
	responses := make([]LotUsageResponse, 0, len(usages))

	for _, usage := range usages {
		responses = append(responses, LotUsageResponse{
			LotNumber: usage.LotNumber,
			Address:   usage.Address,
			Count:     usage.Count,
			Amount:    newMoney(usage.Amount),
		})
	}

	return responses
}
