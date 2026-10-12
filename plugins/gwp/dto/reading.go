package dto

import (
	"time"

	"github.com/abgeo/maroid/plugins/gwp/model"
)

// MeterReadingResponse is one reading of the readings page.
type MeterReadingResponse struct {
	ID           string   `json:"id"`
	SerialNumber string   `json:"serial_number"`
	ReadingDate  string   `json:"reading_date"`
	Value        float64  `json:"value"`
	Usage        *float64 `json:"usage,omitempty"`
}

// NewMeterReadingResponseList converts the readings to their answers.
func NewMeterReadingResponseList(readings []model.Reading) []MeterReadingResponse {
	responses := make([]MeterReadingResponse, 0, len(readings))

	for _, reading := range readings {
		responses = append(responses, MeterReadingResponse{
			ID:           reading.ID,
			SerialNumber: reading.SerialNumber,
			ReadingDate:  reading.ReadingDate.Format(time.DateOnly),
			Value:        reading.Value,
			Usage:        reading.Usage,
		})
	}

	return responses
}
