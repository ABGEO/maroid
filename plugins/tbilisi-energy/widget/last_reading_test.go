package widget_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/errs"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/permission"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/widget"
)

// widgetKeyPattern is the pattern of a widget key that the hub enforces at load
// (WIDGET-FR-005).
var widgetKeyPattern = regexp.MustCompile(`^[a-z0-9-]{1,64}$`)

// fakeReader answers the latest reading, or the error.
type fakeReader struct {
	latest *model.Transaction
	err    error
}

func (f fakeReader) LatestReading(context.Context) (*model.Transaction, error) {
	return f.latest, f.err
}

// GASACC-SC-025: The widget shows the usage of the latest reading, with its charge,
// its meter value, and its date, and links to the transactions page.
func TestTheWidgetShowsTheLatestReading(t *testing.T) {
	t.Parallel()

	reader := fakeReader{latest: &model.Transaction{
		Date:        time.Date(2026, time.September, 28, 0, 0, 0, 0, time.UTC),
		Consumption: 21,
		MeterValue:  1532,
		Amount:      11.33,
	}}

	lastReading := widget.NewLastReading(reader)
	answer, err := lastReading.Load(context.Background())

	require.NoError(t, err)
	require.Equal(t, pluginapi.StatAnswer(pluginapi.StatData{
		Value:   "21",
		Unit:    "m³",
		Caption: "11.33 ₾, meter 1532 m³, 28/09/2026",
		Link:    "/transactions",
	}), answer)

	meta := lastReading.Meta()
	require.Equal(t, "last-reading", meta.Key)
	require.Regexp(t, widgetKeyPattern, meta.Key)
	require.Equal(t, pluginapi.WidgetKindStat, meta.Kind)
	require.Equal(t, pluginapi.WidgetSizeS, meta.Size)
	require.Equal(t, permission.AccountRead, meta.Permission)
}

// GASACC-SC-026: A workspace with no assigned customer answers "not set up" with the
// link to the connection page. A customer with no reading answers "empty".
func TestTheWidgetAsksForTheSetup(t *testing.T) {
	t.Parallel()

	notSetUp, err := widget.NewLastReading(fakeReader{err: errs.ErrNotAssigned}).
		Load(context.Background())
	require.NoError(t, err)
	require.Equal(t, pluginapi.WidgetStateNotSetUp, notSetUp.State)
	require.Equal(t, "/account", notSetUp.Link)

	empty, err := widget.NewLastReading(fakeReader{}).Load(context.Background())
	require.NoError(t, err)
	require.Equal(t, pluginapi.WidgetStateEmpty, empty.State)
}
