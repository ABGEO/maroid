package dto_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/telmico/dto"
	"github.com/abgeo/maroid/plugins/telmico/model"
)

const customerC1 = "8370764"

func decode[T any](t *testing.T, raw string) T {
	t.Helper()

	var value T
	require.NoError(t, json.Unmarshal([]byte(raw), &value))

	return value
}

// ELECACC-SC-016: Each kind of payment maps with the values it holds. The tariff of a
// payment, zero at the portal, maps to none.
func TestAPaymentMapsWithTheValuesItHolds(t *testing.T) {
	t.Parallel()

	charge := decode[dto.Payment](t, `{"operDate":"2026-04-28","name":"თელმიკო",
		"transType":"დარიცხვა ჩვენების მიხედვით","counterValue":1317,"kiloValue":153.30,
		"amount":36.87,"debt":39.39,"counterNumber":"021242007390","dueDate":"2026-05-15",
		"tariffValue":0.24053120}`)

	mapped, err := charge.MapToModel(customerC1)
	require.NoError(t, err)

	assert.Equal(t, customerC1, mapped.CustomerNumber)
	assert.Equal(t, "დარიცხვა ჩვენების მიხედვით", mapped.Operation)
	assert.Equal(t, time.Date(2026, time.April, 28, 0, 0, 0, 0, model.Tbilisi()), mapped.Date)
	assert.InDelta(t, 1317.0, *mapped.MeterValue, 0)
	assert.InDelta(t, 153.3, *mapped.Usage, 0)
	assert.InDelta(t, 36.87, *mapped.Amount, 0)
	assert.InDelta(t, 39.39, *mapped.Balance, 0)
	assert.InDelta(t, 0.2405312, *mapped.Tariff, 0)
	assert.Equal(t, "021242007390", *mapped.MeterSerial)
	assert.Equal(t, time.Date(2026, time.May, 15, 0, 0, 0, 0, model.Tbilisi()), *mapped.DueDate)
	assert.True(t, mapped.IsCharge())

	payment := decode[dto.Payment](t, `{"operDate":"2026-04-01","name":null,
		"transType":"გადახდა","counterValue":null,"kiloValue":null,"amount":-48.57,
		"debt":0.00,"counterNumber":null,"dueDate":"2026-05-15","tariffValue":0}`)

	mapped, err = payment.MapToModel(customerC1)
	require.NoError(t, err)

	assert.Nil(t, mapped.MeterValue)
	assert.Nil(t, mapped.Usage)
	assert.Nil(t, mapped.Tariff)
	assert.Nil(t, mapped.MeterSerial)
	assert.InDelta(t, -48.57, *mapped.Amount, 0)
	assert.False(t, mapped.IsCharge())

	newMeter := decode[dto.Payment](
		t,
		`{"operDate":"2024-10-02","transType":"მრიცხველის დადგმის ჩვენება",
		"counterValue":3,"kiloValue":0,"amount":null,"debt":null,"counterNumber":"021242007390",
		"dueDate":"2024-11-15","tariffValue":0}`,
	)

	mapped, err = newMeter.MapToModel(customerC1)
	require.NoError(t, err)

	assert.InDelta(t, 3.0, *mapped.MeterValue, 0)
	assert.Nil(t, mapped.Amount)
	assert.False(t, mapped.IsCharge())
}

// ELECACC-SC-016: A date that does not parse fails the map.
func TestAPaymentWithABrokenDateFails(t *testing.T) {
	t.Parallel()

	payment := decode[dto.Payment](t, `{"operDate":"28.04.2026","transType":"გადახდა"}`)

	_, err := payment.MapToModel(customerC1)

	require.Error(t, err)
}

// ELECACC-SC-017: A receipt maps to the bill of the month that its number names, issued
// at a time of Tbilisi with any count of decimals.
func TestAReceiptMapsToTheBillOfItsMonth(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		created string
		want    time.Time
	}{
		"three decimals": {
			created: "2026-09-30T09:29:45.299",
			want:    time.Date(2026, time.September, 30, 9, 29, 45, 299_000_000, model.Tbilisi()),
		},
		"two decimals": {
			created: "2026-09-30T09:29:45.09",
			want:    time.Date(2026, time.September, 30, 9, 29, 45, 90_000_000, model.Tbilisi()),
		},
		"one decimal": {
			created: "2026-09-30T09:29:45.8",
			want:    time.Date(2026, time.September, 30, 9, 29, 45, 800_000_000, model.Tbilisi()),
		},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			receipt := decode[dto.Receipt](t, `{"billId":"f9ca32a7-ec95-4145-97af-cbef330af8dc",
				"billNumber":"8370764-2026-9","billCreateDate":"`+setup.created+`",
				"serialnumber":"021242007390","month":"სექტემბერი 2026"}`)

			bill, err := receipt.MapToModel(customerC1)
			require.NoError(t, err)

			assert.Equal(t, customerC1, bill.CustomerNumber)
			assert.Equal(t, "8370764-2026-9", bill.BillNumber)
			assert.Equal(
				t,
				uuid.MustParse("f9ca32a7-ec95-4145-97af-cbef330af8dc"),
				bill.ServiceBillID,
			)
			assert.Equal(
				t,
				time.Date(2026, time.September, 1, 0, 0, 0, 0, model.Tbilisi()),
				bill.Month,
			)
			assert.True(t, setup.want.Equal(bill.IssuedAt))
			assert.Equal(t, "021242007390", *bill.MeterSerial)
		})
	}
}

// ELECACC-SC-017: A receipt whose number names no month, or whose identifier is no
// UUID, fails the map.
func TestABrokenReceiptFails(t *testing.T) {
	t.Parallel()

	const (
		billID  = `"billId":"f9ca32a7-ec95-4145-97af-cbef330af8dc"`
		created = `"billCreateDate":"2026-09-30T09:29:45.299"`
		number  = `"billNumber":"8370764-2026-9"`
	)

	cases := map[string]string{
		"no month":      `{` + billID + `,"billNumber":"8370764",` + created + `}`,
		"month 13":      `{` + billID + `,"billNumber":"8370764-2026-13",` + created + `}`,
		"no identifier": `{"billId":"x",` + number + `,` + created + `}`,
		"no date":       `{` + billID + `,` + number + `,"billCreateDate":"30.09.2026"}`,
	}

	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			receipt := decode[dto.Receipt](t, raw)

			_, err := receipt.MapToModel(customerC1)

			require.Error(t, err)
		})
	}
}

// ELECACC-SC-033: The detail of an account decodes the address alone, so the name and
// the personal number of the holder never reach a value of Go.
func TestTheDetailDecodesNoHolder(t *testing.T) {
	t.Parallel()

	detail := decode[dto.AccountDetail](t, `{"account":"8370764","address":"Nutsubidze 4",
		"owner":"Holder Name","idNumber":"62909012413","emails":["a@b.c"],"phoneNumbers":["555"]}`)

	assert.Equal(t, dto.AccountDetail{Number: customerC1, Address: "Nutsubidze 4"}, detail)
}
