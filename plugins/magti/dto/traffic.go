package dto

import (
	"time"

	"github.com/abgeo/maroid/plugins/magti/model"
)

// TrafficDayResponse is the traffic of one day. A series that the portal answered no
// value for is absent.
type TrafficDayResponse struct {
	Day            string   `json:"day"`
	DataMB         *float64 `json:"data_mb,omitempty"`
	CallInSeconds  *int     `json:"call_in_seconds,omitempty"`
	CallOutSeconds *int     `json:"call_out_seconds,omitempty"`
	SMSSent        *int     `json:"sms_sent,omitempty"`
	SMSReceived    *int     `json:"sms_received,omitempty"`
}

// NewTrafficDayResponseList converts the days to their answers.
func NewTrafficDayResponseList(days []model.TrafficDay) []TrafficDayResponse {
	responses := make([]TrafficDayResponse, 0, len(days))

	for _, day := range days {
		responses = append(responses, TrafficDayResponse{
			Day:            day.Day.Format(time.DateOnly),
			DataMB:         day.DataMB,
			CallInSeconds:  day.CallInSeconds,
			CallOutSeconds: day.CallOutSeconds,
			SMSSent:        day.SMSSent,
			SMSReceived:    day.SMSReceived,
		})
	}

	return responses
}
