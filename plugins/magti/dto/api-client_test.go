package dto_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/magti/dto"
	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/model"
)

func decode[T any](t *testing.T, raw string) T {
	t.Helper()

	var value T
	require.NoError(t, json.Unmarshal([]byte(raw), &value))

	return value
}

// MAGTIACC-SC-016: The reader keeps the rows of known days, skips the total row and a row
// of day zero, and merges the series by day.
func TestTheUsageMergesByDay(t *testing.T) {
	t.Parallel()

	days := map[model.Day]*model.TrafficDay{}

	dto.MergeUsage(days, dto.SeriesData, decode[[]dto.UsageRow](t, `[
		{"day": 1795464000000, "chartData": 812.4},
		{"day": 0, "chartData": 3},
		{"sum": true, "chartData": 812.4}
	]`))
	dto.MergeUsage(days, dto.SeriesMinutes, decode[[]dto.UsageRow](t, `[
		{"day": 1795464000000, "chartData": "12.5", "chartDataIn": 3}
	]`))

	require.Len(t, days, 1)

	traffic := days[model.Day{Year: 2026, Month: time.November, Day: 24}]
	require.NotNil(t, traffic)
	assert.InDelta(t, 812.4, *traffic.DataMB, 1e-9)
	assert.Equal(t, 750, *traffic.CallOutSeconds)
	assert.Equal(t, 180, *traffic.CallInSeconds)
	assert.Nil(t, traffic.SMSSent)
	assert.Nil(t, traffic.SMSReceived)
}

// MAGTIACC-SC-017: A money row is a moment of Tbilisi, with the absolute amount.
func TestAMoneyRowIsAMomentOfTbilisi(t *testing.T) {
	t.Parallel()

	entries, err := dto.MoneyEntries(decode[[]dto.MoneyRow](t, `[
		{"transactionDate": "13/11/2026 01:50:48", "entryAmount": "-17.30", "actionType": "Top up"},
		{"sum": true, "entryAmount": "17.30"}
	]`), model.DirectionIn)
	require.NoError(t, err)

	require.Len(t, entries, 1)
	assert.Equal(t, "2026-11-12T21:50:48Z", entries[0].OccurredAt.Format(time.RFC3339))
	assert.Equal(t, model.Tetri(1730), entries[0].Amount)
	assert.Equal(t, "Top up", entries[0].Kind)
	assert.Equal(t, model.DirectionIn, entries[0].Direction)
}

// MAGTIACC-SC-021: A value that does not parse is a failed request.
func TestAValueThatDoesNotParseFails(t *testing.T) {
	t.Parallel()

	accounts := decode[dto.Accounts](t, `{"balances": [
		{"balance": "lots", "accountGroupConfigItems": [{"accountGroupKey": "BALANCE"}]}
	]}`)

	_, err := dto.MobileState("599000001", accounts, nil)
	require.ErrorIs(t, err, errs.ErrRequestFailed)

	minutes := decode[dto.Accounts](t, `{"balances": [
		{"balance": "0.03", "accountGroupConfigItems": [{"accountGroupKey": "BALANCE"}]},
		{"balance": "1:02:03", "accountGroupConfigItems": [{"accountGroupKey": "MINUTES"}]}
	]}`)

	state, err := dto.MobileState("599000001", minutes, nil)
	require.NoError(t, err)
	assert.Equal(t, 3723, state.Allowances.CallSeconds)
}

// MAGTIACC-SC-021: An answer that holds no balance is a failed request, never a balance
// of zero.
func TestAnAnswerWithNoBalanceFails(t *testing.T) {
	t.Parallel()

	mobile := decode[dto.Accounts](t, `{"balances": [
		{"balance": "100.0", "accountGroupConfigItems": [{"accountGroupKey": "SMS"}]}
	]}`)

	_, err := dto.MobileState("599000001", mobile, nil)
	require.ErrorIs(t, err, errs.ErrRequestFailed)

	homeLine := decode[dto.IspAccounts](t, `{"package_sum": 40, "balances": [
		{"groupKey": "ISP_INTERNET", "balance": "70 Mbps"}
	]}`)

	_, err = dto.HomeLineState("322000001", homeLine, dto.AccountData{})
	require.ErrorIs(t, err, errs.ErrRequestFailed)
}
