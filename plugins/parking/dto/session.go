package dto

import (
	"time"

	"github.com/abgeo/maroid/plugins/parking/model"
)

// SessionResponse is one row of the sessions page.
//
// PARKHIST-INV-002: The answer holds no wallet number.
type SessionResponse struct {
	ID              string        `json:"id"`
	LotNumber       string        `json:"lot_number"`
	Address         string        `json:"address"`
	Plate           string        `json:"plate"`
	StartedAt       time.Time     `json:"started_at"`
	EndedAt         time.Time     `json:"ended_at"`
	DurationSeconds int64         `json:"duration_seconds"`
	Amount          MoneyResponse `json:"amount"`
	Free            bool          `json:"free"`
}

// SessionTotalResponse sums every session of a filter.
type SessionTotalResponse struct {
	Count           int           `json:"count"`
	DurationSeconds int64         `json:"duration_seconds"`
	Amount          MoneyResponse `json:"amount"`
}

// PlateResponse is one plate of a stored session.
type PlateResponse struct {
	Plate string `json:"plate"`
}

// NewSessionResponseList converts the sessions to their answers.
//
// PARKHIST-FR-030, PARKHIST-DD-005: A session with an amount of 0 is a free parking.
func NewSessionResponseList(sessions []model.Session) []SessionResponse {
	responses := make([]SessionResponse, 0, len(sessions))

	for _, session := range sessions {
		responses = append(responses, SessionResponse{
			ID:              session.ID,
			LotNumber:       session.LotNumber,
			Address:         session.Address,
			Plate:           session.Plate,
			StartedAt:       session.StartedAt.UTC(),
			EndedAt:         session.EndedAt.UTC(),
			DurationSeconds: int64(session.EndedAt.Sub(session.StartedAt) / time.Second),
			Amount:          newMoney(session.Amount),
			Free:            session.Amount == 0,
		})
	}

	return responses
}

// NewSessionTotalResponse converts the total to its answer.
func NewSessionTotalResponse(total model.SessionTotal) SessionTotalResponse {
	return SessionTotalResponse{
		Count:           total.Count,
		DurationSeconds: total.DurationSeconds,
		Amount:          newMoney(total.Amount),
	}
}

// NewPlateResponseList converts the plates to their answers.
func NewPlateResponseList(plates []string) []PlateResponse {
	responses := make([]PlateResponse, 0, len(plates))
	for _, plate := range plates {
		responses = append(responses, PlateResponse{Plate: plate})
	}

	return responses
}

func newMoney(amount float64) MoneyResponse {
	return MoneyResponse{Amount: amount, Currency: currencyGEL}
}
