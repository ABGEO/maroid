package dto_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/tbilisi-energy/dto"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
)

const homeAnswer = `{
	"user": {"userName": "ABGEO", "personalId": "62909012413"},
	"customers": [{
		"custNumber": "757235",
		"controlNumber": 992,
		"customerName": "თაკალანძე თემურ",
		"address": "ნუცუბიძე, ნუცუბიძის 4 მ/რ",
		"personalN": "62909012413",
		"balance": 0.00
	}],
	"success": true,
	"message": null,
	"code": 0
}`

// GASACC-SC-003: The customer number joins the number and the control number.
func TestTheCustomerNumberJoinsTheControlNumber(t *testing.T) {
	t.Parallel()

	var answer dto.HomeResponse
	require.NoError(t, json.Unmarshal([]byte(homeAnswer), &answer))

	require.Len(t, answer.Customers, 1)
	require.Equal(t, "757235-992", answer.Customers[0].Number())
	require.Equal(t, "თაკალანძე თემურ", answer.Customers[0].Name)
}

// GASACC-SC-033: The decoded customer holds no personal number.
func TestTheCustomerHoldsNoPersonalNumber(t *testing.T) {
	t.Parallel()

	var answer dto.HomeResponse
	require.NoError(t, json.Unmarshal([]byte(homeAnswer), &answer))

	encoded, err := json.Marshal(answer)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "62909012413")
}

// GASACC-SC-028: A reading maps its files. A payment maps none.
func TestATransactionMapsToTheModel(t *testing.T) {
	t.Parallel()

	reading := dto.Transaction{
		Consumption:        21,
		Amount:             11.33,
		MeterReading:       1532,
		Balance:            11.33,
		OperationID:        125,
		OperationName:      "რეალური ჩვენება",
		OperationDate:      "2026-09-28T00:00:00",
		BillingDocumentURL: "api/Customer/GetCustomerBillPdf/x.pdf",
	}
	payment := dto.Transaction{OperationID: 196, OperationDate: "2026-10-01T00:00:00"}

	mapped, err := reading.MapToModel("757235-992")
	require.NoError(t, err)
	require.Equal(t, "757235-992", mapped.CustomerNumber)
	require.Equal(t, 125, mapped.TypeID)
	require.Equal(t, "2026-09-28", mapped.Date.Format(time.DateOnly))
	require.Equal(t, model.Tbilisi().String(), mapped.Date.Location().String())
	require.InDelta(t, 21.0, mapped.Consumption, 0)
	require.InDelta(t, 1532.0, mapped.MeterValue, 0)
	require.NotNil(t, mapped.BillPath)
	require.Equal(t, "api/Customer/GetCustomerBillPdf/x.pdf", *mapped.BillPath)
	require.Nil(t, mapped.MeterPhotoPath)

	mappedPayment, err := payment.MapToModel("757235-992")
	require.NoError(t, err)
	require.Nil(t, mappedPayment.BillPath)
	require.Nil(t, mappedPayment.MeterPhotoPath)
}

// GASACC-SC-012: The request names the period as two dates.
func TestTheRequestNamesThePeriod(t *testing.T) {
	t.Parallel()

	period := model.PreviousMonth(time.Date(2026, time.October, 13, 12, 0, 0, 0, model.Tbilisi()))

	request := dto.NewTransactionsRequest("757235-992", period)

	require.Equal(t, dto.TransactionsRequest{
		CustomerNumber: "757235-992",
		DateFrom:       "2026-09-01",
		DateTo:         "2026-09-30",
	}, request)
}
