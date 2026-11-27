package widget_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pensions/model"
	"github.com/abgeo/maroid/plugins/pensions/widget"
)

// fixedView answers one summary view.
type fixedView struct {
	view model.SummaryView
}

func (f fixedView) Summary(context.Context) (model.SummaryView, error) {
	return f.view, nil
}

func storedSummary() *model.AccountSummary {
	return &model.AccountSummary{
		AssetValue:  12480.55,
		Gain:        1904.10,
		CollectedAt: time.Date(2026, time.November, 1, 6, 0, 0, 0, time.UTC),
	}
}

func inTbilisi(t *testing.T) context.Context {
	t.Helper()

	tbilisi, err := time.LoadLocation("Asia/Tbilisi")
	require.NoError(t, err)

	return pluginapi.ContextWithActingTimeZone(t.Context(), tbilisi)
}

// PENSION-SC-032: The widget shows the asset value, the gain, and the date of the
// collection in the time zone of the user, and links to the overview.
func TestTheWidgetShowsTheAssetValueAndTheGain(t *testing.T) {
	t.Parallel()

	answer, err := widget.NewAssetValue(fixedView{view: model.SummaryView{Connected: true, Summary: storedSummary()}}).
		Load(inTbilisi(t))

	require.NoError(t, err)
	assert.Equal(t, pluginapi.StatAnswer(pluginapi.StatData{
		Value:   "12,480.55",
		Unit:    "₾",
		Caption: "Gain 1,904.10 ₾, 01/11/2026",
		Link:    "/overview",
	}), answer)
}

// PENSION-SC-033: No summary and no session is not set up. No summary and a session is
// empty. A summary with no session still shows.
func TestTheWidgetNamesTheStateOfTheUser(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		view  model.SummaryView
		state pluginapi.WidgetState
	}{
		"user B": {view: model.SummaryView{}, state: pluginapi.WidgetStateNotSetUp},
		"user C": {view: model.SummaryView{Connected: true}, state: pluginapi.WidgetStateEmpty},
		"user D": {
			view:  model.SummaryView{Summary: storedSummary()},
			state: pluginapi.WidgetStateReady,
		},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			answer, err := widget.NewAssetValue(fixedView{view: setup.view}).Load(inTbilisi(t))

			require.NoError(t, err)
			assert.Equal(t, setup.state, answer.State)
		})
	}

	answer, err := widget.NewAssetValue(fixedView{}).Load(inTbilisi(t))
	require.NoError(t, err)
	assert.Equal(t, pluginapi.NotSetUpAnswer("Connect the pension account.", "/account"), answer)
}

// PENSION-SC-032: An amount takes a comma between each group of thousands.
func TestLariGroupsTheThousands(t *testing.T) {
	t.Parallel()

	for amount, want := range map[float64]string{
		0:         "0.00",
		999.999:   "1,000.00",
		12480.55:  "12,480.55",
		1234567.8: "1,234,567.80",
		-1904.1:   "-1,904.10",
	} {
		assert.Equal(t, want, widget.Lari(amount))
	}
}
