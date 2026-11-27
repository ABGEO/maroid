package dto_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/parking/dto"
	"github.com/abgeo/maroid/plugins/parking/errs"
	"github.com/abgeo/maroid/plugins/parking/model"
)

const (
	walletNumber   = "GE00TB0000000000000099"
	personID       = "731942"
	sessionPayload = `{"result":{"dataType":"PAGE","data":[{"id":45304769,"status":"FINISHED",` +
		`"type":"BOTH","govNumber":"AB071EO","parkingPlace":{"uniqueNumber":"A1637",` +
		`"address":"12 Example Street"},"startDate":"2026-11-21T13:26:08.347292",` +
		`"endDate":"2026-11-21T14:14:21.809195","amount":0.55,"fixedAmount":0.55,` +
		`"walletNumber":"` + walletNumber + `","startDateFormatted":"21/11/2026"}],` +
		`"page":{"total":468,"number":0,"size":20}}}`
)

func entryOf(t *testing.T, body string) dto.PaymentEntry {
	t.Helper()

	var entry dto.PaymentEntry
	require.NoError(t, json.Unmarshal([]byte(body), &entry))

	return entry
}

// PARKHIST-SC-012: A row of the capture decodes with its times read in Asia/Tbilisi.
func TestASessionReadsItsTimesInTbilisi(t *testing.T) {
	t.Parallel()

	var page dto.Page[dto.HistorySession]
	require.NoError(t, json.Unmarshal([]byte(sessionPayload), &page))

	assert.Equal(t, dto.PageMeta{Total: 468, Number: 0, Size: 20}, page.Meta)
	require.Len(t, page.Items, 1)

	session, err := page.Items[0].ToModel()
	require.NoError(t, err)

	assert.Equal(t, model.Session{
		ExternalID:  45304769,
		LotNumber:   "A1637",
		Address:     "12 Example Street",
		Plate:       "AB071EO",
		ParkingType: "BOTH",
		StartedAt:   time.Date(2026, time.November, 21, 9, 26, 8, 347292000, time.UTC),
		EndedAt:     time.Date(2026, time.November, 21, 10, 14, 21, 809195000, time.UTC),
		Amount:      0.55,
	}, session)
}

// PARKHIST-SC-012: A session in an unknown layout or with a negative amount does not decode.
func TestAnUnexpectedSessionFails(t *testing.T) {
	t.Parallel()

	end := "2026-11-21T14:14:21"
	badEnd := "21/11/2026"

	for name, session := range map[string]dto.HistorySession{
		"bad end":         {ID: 1, Status: "FINISHED", StartDate: end, EndDate: &badEnd},
		"negative amount": {ID: 2, Status: "FINISHED", StartDate: end, EndDate: &end, Amount: -1},
		"active":          {ID: 3, Status: "ACTIVE", StartDate: end},
	} {
		_, err := session.ToModel()
		assert.ErrorIs(t, err, errs.ErrUnexpectedAnswer, name)
	}
}

// PARKHIST-SC-013: Each kind of the ledger maps to a kind of Maroid, and keeps its value.
func TestATransactionMapsItsKind(t *testing.T) {
	t.Parallel()

	const moment = `"operationDate":"2026-11-21T10:14:21.827+0000"`

	occurred := time.Date(2026, time.November, 21, 10, 14, 21, 827000000, time.UTC)
	cases := []struct {
		body        string
		kind        model.TransactionKind
		service     string
		description string
		session     *int64
	}{
		{
			body: `{"id":1,"operationType":"FILL_BALANCE","amount":10,"currentBalance":19.08,` +
				`"additionalInformationForMobileEng":" Fill Balance ",` + moment + `}`,
			kind: model.TransactionKindTopUp, service: "FILL_BALANCE", description: "Fill Balance",
		},
		{
			body: `{"id":2,"operationType":"ZONE_PARKING_PAYMENT","amount":0.55,` +
				`"currentBalance":9.08,"parkingId":45304769,` +
				`"additionalInformationForMobileEng":"Zone Parking",` + moment + `}`,
			kind: model.TransactionKindParkingPayment, service: "ZONE_PARKING_PAYMENT",
			description: "Zone Parking", session: new(int64(45304769)),
		},
		{
			body: `{"id":3,"operationType":"CARD_PAYMENT","amount":50,"currentBalance":4,` +
				`"cardId":88,"additionalInformationForMobileEng":"Permit",` + moment + `}`,
			kind:        model.TransactionKindPermitPurchase,
			service:     "CARD_PAYMENT",
			description: "Permit",
		},
		{
			body: `{"id":4,"operationType":"REFUND","amount":2,"currentBalance":6,` +
				`"additionalInformationForMobileEng":"Refund",` + moment + `}`,
			kind: model.TransactionKindOther, service: "REFUND", description: "Refund",
		},
	}

	for _, example := range cases {
		transaction, err := entryOf(t, example.body).ToModel()
		require.NoError(t, err)

		assert.Equal(t, example.kind, transaction.Kind, example.service)
		assert.Equal(t, example.service, transaction.ServiceKind)
		assert.Equal(t, example.description, transaction.Description)
		assert.Equal(t, example.session, transaction.SessionExternalID)
		assert.Equal(t, occurred, transaction.OccurredAt)
	}
}

// PARKHIST-SC-013: A moment in an unknown layout or a negative amount does not decode.
func TestAnUnexpectedTransactionFails(t *testing.T) {
	t.Parallel()

	for name, entry := range map[string]dto.PaymentEntry{
		"bad moment":      {ID: 1, OperationDate: "2026-11-21"},
		"negative amount": {ID: 2, OperationDate: "2026-11-21T10:14:21.827+0000", Amount: -1},
	} {
		_, err := entry.ToModel()
		assert.ErrorIs(t, err, errs.ErrUnexpectedAnswer, name)
	}
}

// PARKHIST-SC-017: The wallet number and the personal identifier of the capture reach no
// model.
func TestNoPersonalValueReachesAModel(t *testing.T) {
	t.Parallel()

	var page dto.Page[dto.HistorySession]
	require.NoError(t, json.Unmarshal([]byte(sessionPayload), &page))

	session, err := page.Items[0].ToModel()
	require.NoError(t, err)

	transaction, err := entryOf(t, `{"id":5,"personId":`+personID+`,"walletNumber":"`+
		walletNumber+`","operationType":"FILL_BALANCE","amount":1,"currentBalance":1,`+
		`"operationDate":"2026-11-21T10:14:21.827+0000"}`).ToModel()
	require.NoError(t, err)

	encoded, err := json.Marshal([]any{session, transaction})
	require.NoError(t, err)

	assert.NotContains(t, string(encoded), walletNumber)
	assert.NotContains(t, string(encoded), personID)
}
