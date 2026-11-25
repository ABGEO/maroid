package dto_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/drivvo/dto"
	"github.com/abgeo/maroid/plugins/drivvo/errs"
	"github.com/abgeo/maroid/plugins/drivvo/model"
)

func decode[T any](t *testing.T, body string) T {
	t.Helper()

	var value T
	require.NoError(t, json.Unmarshal([]byte(body), &value))

	return value
}

// dayOf2026 answers a day of 2026 at midnight UTC.
func dayOf2026(month time.Month, date int) time.Time {
	return time.Date(2026, month, date, 0, 0, 0, 0, time.UTC)
}

// DRIVVO-SC-018: A moment of the service is a time of Tbilisi, held in UTC, and a due day
// of an instalment keeps its date.
func TestAMomentIsATimeOfTbilisiAndADueDayKeepsItsDate(t *testing.T) {
	t.Parallel()

	expense := decode[dto.Expense](t, `{
		"id_despesa": 501, "data": "2026-11-24 18:30:00",
		"parcelas": [{"numero": 1, "data_vencimento": "2026-12-01 00:00:00", "valor": 10}]
	}`)

	entry, err := expense.Read()
	require.NoError(t, err)

	assert.Equal(t, time.Date(2026, 11, 24, 14, 30, 0, 0, time.UTC), entry.OccurredAt)
	assert.Equal(t, time.UTC, entry.OccurredAt.Location())
	require.Len(t, entry.Instalments, 1)
	assert.Equal(t, dayOf2026(12, 1), entry.Instalments[0].DueOn)
}

// DRIVVO-SC-019: A refuel holds a line for each fuel above zero, sums the lines with a free
// line at zero, and keeps the three values of the charge.
func TestARefuelHoldsItsFuelLinesAndItsCharge(t *testing.T) {
	t.Parallel()

	refuel := decode[dto.Refuel](t, `{
		"id_abastecimento": 101, "data": "2026-11-20 09:00:00", "odometro": 152300,
		"id_combustivel": 1, "preco": 3, "volume": 0, "valor_total": 90, "tanque_cheio": true,
		"id_combustivel_dois": 2, "preco_dois": 3, "volume_dois": 5, "valor_total_dois": 15,
		"sem_custo_dois": true, "id_combustivel_tres": 0, "valor_total_tres": 99,
		"bateria_inicial_pct": 20, "bateria_final_pct": 80, "duracao_minutos": 45,
		"esqueceu_anterior": true, "id_posto_combustivel": 4, "id_forma_pagamento": 3,
		"observacao": "Highway"
	}`)

	entry, err := refuel.Read()
	require.NoError(t, err)

	assert.Equal(t, model.KindRefuel, entry.Kind)
	require.NotNil(t, entry.Amount)
	assert.Equal(t, model.AmountOf(90), *entry.Amount)
	assert.Equal(t, []model.Line{
		{
			Position: 1, Item: 1, Amount: model.AmountOf(90), Fuel: true,
			UnitPrice: model.AmountOf(3), Volume: 30, FullTank: true,
		},
		{
			Position: 2, Item: 2, Amount: model.AmountOf(15), Fuel: true,
			UnitPrice: model.AmountOf(3), Volume: 5, Free: true,
		},
	}, entry.Lines)
	require.NotNil(t, entry.BatteryStartPct)
	require.NotNil(t, entry.BatteryEndPct)
	require.NotNil(t, entry.ChargeMinutes)
	assert.InDelta(t, 20.0, *entry.BatteryStartPct, 0)
	assert.InDelta(t, 80.0, *entry.BatteryEndPct, 0)
	assert.Equal(t, int32(45), *entry.ChargeMinutes)
	assert.True(t, entry.MissedRefuel)
	assert.Equal(t, int32(4), entry.Station)
	assert.Equal(t, int32(3), entry.PaymentMethod)
	assert.Equal(t, "Highway", entry.Note)
}

// DRIVVO-SC-019: A refuel with no charge holds no value of a charge.
func TestARefuelWithNoChargeHoldsNoChargeValue(t *testing.T) {
	t.Parallel()

	refuel := decode[dto.Refuel](t, `{
		"id_abastecimento": 102, "data": "2026-11-20 09:00:00", "id_combustivel": 1,
		"bateria_inicial_pct": 0, "bateria_final_pct": null, "duracao_minutos": 0
	}`)

	entry, err := refuel.Read()
	require.NoError(t, err)

	assert.Nil(t, entry.BatteryStartPct)
	assert.Nil(t, entry.BatteryEndPct)
	assert.Nil(t, entry.ChargeMinutes)
}

// DRIVVO-SC-020: An expense holds each of its instalments, with its day and its amount.
func TestAnExpenseHoldsItsInstalments(t *testing.T) {
	t.Parallel()

	expense := decode[dto.Expense](t, `{
		"id_despesa": 502, "data": "2026-01-10 12:00:00", "titulo": "Insurance 2026",
		"tipos_despesa": [{"id_tipo_despesa": 7, "valor": 600}],
		"parcelas": [
			{"numero": 1, "data_vencimento": "2026-01-10 00:00:00", "valor": 200},
			{"numero": 2, "data_vencimento": "2026-02-10 00:00:00", "valor": 200},
			{"numero": 3, "data_vencimento": "2026-03-10 00:00:00", "valor": 200},
			{"numero": 0, "data_vencimento": "", "valor": 0}
		]
	}`)

	entry, err := expense.Read()
	require.NoError(t, err)

	assert.Equal(t, "Insurance 2026", entry.Title)
	require.NotNil(t, entry.Amount)
	assert.Equal(t, model.AmountOf(600), *entry.Amount)
	assert.Equal(t, []model.Line{{Position: 1, Item: 7, Amount: model.AmountOf(600)}}, entry.Lines)
	assert.Equal(t, []model.Instalment{
		{Number: 1, DueOn: dayOf2026(1, 10), Amount: model.AmountOf(200)},
		{Number: 2, DueOn: dayOf2026(2, 10), Amount: model.AmountOf(200)},
		{Number: 3, DueOn: dayOf2026(3, 10), Amount: model.AmountOf(200)},
	}, entry.Instalments)
}

// DRIVVO-SC-020: A maintenance holds a line for each type in the order of the service, and
// an income takes valor with its instalments.
func TestAMaintenanceAndAnIncomeHoldTheirAmounts(t *testing.T) {
	t.Parallel()

	maintenance := decode[dto.Maintenance](t, `{
		"id_servico": 201, "data": "2026-01-05 10:00:00", "id_local": 6,
		"tipos_servico": [{"id_tipo_servico": 5, "valor": 100}, {"id_tipo_servico": 8, "valor": 200}],
		"parcelas": [{"numero": 1, "data_vencimento": "2026-01-15", "valor": 150},
		             {"numero": 2, "data_vencimento": "2026-02-15", "valor": 150}]
	}`)

	entry, err := maintenance.Read()
	require.NoError(t, err)

	require.NotNil(t, entry.Amount)
	assert.Equal(t, model.AmountOf(300), *entry.Amount)
	assert.Equal(t, int32(6), entry.Place)
	require.Len(t, entry.Lines, 2)
	assert.Equal(t, int32(8), entry.Lines[1].Item)
	assert.Len(t, entry.Instalments, 2)

	income := decode[dto.Income](t, `{
		"id_receita": 601, "data": "2026-03-02 10:00:00", "id_tipo_receita": 9, "valor": 50,
		"parcelas": [{"numero": 1, "data_vencimento": "2026-03-02", "valor": 50}]
	}`)

	incomeEntry, err := income.Read()
	require.NoError(t, err)

	require.NotNil(t, incomeEntry.Amount)
	assert.Equal(t, model.AmountOf(50), *incomeEntry.Amount)
	assert.Equal(t, int32(9), incomeEntry.IncomeType)
	assert.Len(t, incomeEntry.Instalments, 1)
}

// DRIVVO-SC-041: An odometer entry holds no amount.
func TestAnOdometerEntryHoldsNoAmount(t *testing.T) {
	t.Parallel()

	reading := decode[dto.OdometerEntry](
		t,
		`{"id_leitura": 701, "data": "2026-04-01 08:00:00", "odometro": 13000}`,
	)

	entry, err := reading.Read()
	require.NoError(t, err)

	assert.Equal(t, model.KindOdometer, entry.Kind)
	assert.Nil(t, entry.Amount)
	assert.InDelta(t, 13000.0, entry.Odometer, 0)
}

// DRIVVO-SC-018: A moment with no known form fails the read and names the entry.
func TestAMomentWithNoKnownFormFails(t *testing.T) {
	t.Parallel()

	reading := decode[dto.OdometerEntry](t, `{"id_leitura": 702, "data": "24/11/2026"}`)

	_, err := reading.Read()
	require.ErrorIs(t, err, errs.ErrRequestFailed)
	assert.Contains(t, err.Error(), "odometer 702")
}

// DRIVVO-SC-047: unico_repetir false names a reminder that repeats, with its interval in
// the unit of periodo.
func TestAReminderReadsItsRepeat(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		body string
		want model.Reminder
	}{
		"each 12 months or 10,000 km": {
			body: `{"id_lembrete": 1, "id_tipo_servico": 5, "data": "2026-03-01 00:00:00",
				"odometro": 150000, "unico_repetir": false, "repetir_distancia": 10000,
				"repetir_tempo": 12, "periodo": 2}`,
			want: model.Reminder{
				SourceID: 1, Kind: model.KindMaintenance, Item: 5, Repeats: true,
				DueOn: new(dayOf2026(3, 1)), DueOdometer: new(150000.0),
				RepeatEvery: new(int32(12)), RepeatUnit: new(model.RepeatMonth),
				RepeatDistance: new(10000.0),
			},
		},
		"each 30 days": {
			body: `{"id_lembrete": 2, "id_tipo_servico": 5, "unico_repetir": false,
				"repetir_tempo": 30, "periodo": 1}`,
			want: model.Reminder{
				SourceID: 2, Kind: model.KindMaintenance, Item: 5, Repeats: true,
				RepeatEvery: new(int32(30)), RepeatUnit: new(model.RepeatDay),
			},
		},
		"one time for an expense type": {
			body: `{"id_lembrete": 3, "tipo": true, "id_tipo_despesa": 7, "id_tipo_servico": 0,
				"data": "2026-12-07 00:00:00", "unico_repetir": true, "repetir_tempo": 1, "periodo": 3}`,
			want: model.Reminder{
				SourceID: 3, Kind: model.KindExpense, Item: 7, DueOn: new(dayOf2026(12, 7)),
			},
		},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			reminder, err := decode[dto.Reminder](t, setup.body).Read()
			require.NoError(t, err)
			assert.Equal(t, setup.want, reminder)
		})
	}
}

// DRIVVO-SC-005: The session ends at now plus expira_em_segundos, else at the end that the
// answer names in UTC, else one day after now.
func TestTheSessionEndsAtTheFirstKnownEnd(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 25, 6, 0, 0, 0, time.UTC)

	assert.Equal(t, now.Add(time.Hour), dto.LogInAnswer{ExpiraEmSegundos: 3600}.Expiry(now))
	assert.Equal(
		t,
		time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC),
		dto.LogInAnswer{TokenValidUntil: "2026-11-26 10:00:00"}.Expiry(now),
	)
	assert.Equal(t, now.Add(24*time.Hour), dto.LogInAnswer{}.Expiry(now))
}

// DRIVVO-SC-033: An empty currency is USD, as the service shows it.
func TestAnEmptyCurrencyIsUSD(t *testing.T) {
	t.Parallel()

	settings, err := dto.Settings{UnidadeMedida: 3, EficienciaCombustivel: 9}.Read()
	require.NoError(t, err)

	assert.Equal(t, model.CarLogSettings{
		VolumeUnit:        model.VolumeGallonImperial,
		ConsumptionFormat: 9,
		Currency:          "USD",
	}, settings)
}
