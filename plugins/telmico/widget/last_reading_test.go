package widget_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/model"
	"github.com/abgeo/maroid/plugins/telmico/permission"
	"github.com/abgeo/maroid/plugins/telmico/widget"
)

// heldCharges answers the charges that it holds, or err.
type heldCharges struct {
	charges []model.Transaction
	err     error
}

func (h heldCharges) LatestReading(context.Context) ([]model.Transaction, error) {
	return h.charges, h.err
}

func chargeOf(meter, usage, amount float64) model.Transaction {
	return model.Transaction{
		CustomerNumber: "8370764",
		Date:           time.Date(2026, time.April, 28, 0, 0, 0, 0, model.Tbilisi()),
		MeterValue:     new(meter),
		Usage:          new(usage),
		Amount:         new(amount),
	}
}

// ELECACC-SC-023: The widget shows the summed usage of the split reading in kWh, with
// the summed charge, the highest meter value, and the date, and links to the
// transactions page.
func TestTheWidgetShowsTheLatestReading(t *testing.T) {
	t.Parallel()

	last := widget.NewLastReading(heldCharges{charges: []model.Transaction{
		chargeOf(1163, 13.2, 2.52),
		chargeOf(1317, 153.3, 36.87),
	}})

	meta := last.Meta()
	assert.Equal(t, "last-reading", meta.Key)
	assert.Equal(t, "Electricity usage", meta.Title)
	assert.Equal(t, pluginapi.WidgetKindStat, meta.Kind)
	assert.Equal(t, pluginapi.WidgetSizeS, meta.Size)
	assert.Equal(t, permission.AccountRead, meta.Permission)

	answer, err := last.Load(context.Background())
	require.NoError(t, err)

	assert.Equal(t, pluginapi.StatAnswer(pluginapi.StatData{
		Value:   "166.5",
		Unit:    "kWh",
		Caption: "39.39 ₾, meter 1317 kWh, 28/04/2026",
		Link:    "/transactions",
	}), answer)
}

// ELECACC-SC-024: With no assignment the widget is not set up and links to the account
// page. With no reading it is empty.
func TestTheWidgetNamesItsState(t *testing.T) {
	t.Parallel()

	answer, err := widget.NewLastReading(heldCharges{err: errs.ErrNotAssigned}).
		Load(context.Background())
	require.NoError(t, err)
	assert.Equal(t, pluginapi.NotSetUpAnswer(
		"Connect the electricity account and assign a customer.",
		"/account",
	), answer)

	answer, err = widget.NewLastReading(heldCharges{}).Load(context.Background())
	require.NoError(t, err)
	assert.Equal(t, pluginapi.EmptyAnswer("No reading yet."), answer)
}
