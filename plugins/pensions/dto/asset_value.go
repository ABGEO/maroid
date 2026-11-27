package dto

import (
	"time"

	"github.com/abgeo/maroid/plugins/pensions/model"
)

// AssetValueResponse is the asset value of the account on one date.
type AssetValueResponse struct {
	ValuedOn   string        `json:"valued_on"`
	AssetValue MoneyResponse `json:"asset_value"`
	Units      float64       `json:"units"`
	UnitPrice  float64       `json:"unit_price"`
}

// NewAssetValueResponseList converts the asset values to their answers.
func NewAssetValueResponseList(values []model.AssetValue) []AssetValueResponse {
	responses := make([]AssetValueResponse, 0, len(values))
	for _, value := range values {
		responses = append(responses, AssetValueResponse{
			ValuedOn:   value.ValuedOn.Format(time.DateOnly),
			AssetValue: NewMoneyResponse(value.AssetValue),
			Units:      value.Units,
			UnitPrice:  value.UnitPrice,
		})
	}

	return responses
}
