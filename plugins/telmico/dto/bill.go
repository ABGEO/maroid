package dto

import (
	"time"

	"github.com/abgeo/maroid/plugins/telmico/model"
)

// BillResponse is one bill of the bills page.
type BillResponse struct {
	ID         string `json:"id"`
	Month      string `json:"month"`
	BillNumber string `json:"bill_number"`
	IssuedOn   string `json:"issued_on"`
	Document   string `json:"document"`
}

// NewBillResponseList converts the bills to their answers. documentOf answers the
// absolute address of the document of a bill.
func NewBillResponseList(bills []model.Bill, documentOf func(id string) string) []BillResponse {
	responses := make([]BillResponse, 0, len(bills))

	for _, bill := range bills {
		responses = append(responses, BillResponse{
			ID:         bill.ID,
			Month:      model.MonthOf(bill.Month).String(),
			BillNumber: bill.BillNumber,
			IssuedOn:   bill.IssuedAt.In(model.Tbilisi()).Format(time.DateOnly),
			Document:   documentOf(bill.ID),
		})
	}

	return responses
}
