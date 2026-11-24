package widget_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/model"
	"github.com/abgeo/maroid/plugins/magti/permission"
	"github.com/abgeo/maroid/plugins/magti/widget"
)

// heldState answers the state that it holds, or err.
type heldState struct {
	state model.State
	err   error
}

func (h heldState) Read(context.Context) (model.State, error) {
	return h.state, h.err
}

func mobileState(daysLeft int) model.State {
	renewsAt := time.Date(2026, time.December, 13, 1, 50, 47, 0, model.Tbilisi())

	return model.State{
		Kind:    model.SubscriberKindMobile,
		Balance: 3,
		Plan: &model.Plan{
			Name:     "Cocktail 17",
			Price:    1730,
			RenewsAt: &renewsAt,
			DaysLeft: daysLeft,
		},
		Allowances: &model.Allowances{DataMB: 2589.29, CallSeconds: 3376, SMS: 100},
	}
}

func load(t *testing.T, held heldState) pluginapi.WidgetAnswer {
	t.Helper()

	answer, err := widget.NewSubscriberState(held).Load(context.Background())
	require.NoError(t, err)

	return answer
}

// MAGTIACC-SC-023: The widget shows the balance in lari, what the plan leaves, and the
// days to its renewal, and links to the subscriber page.
func TestTheWidgetShowsTheSubscriberState(t *testing.T) {
	t.Parallel()

	state := widget.NewSubscriberState(heldState{state: mobileState(18)})

	meta := state.Meta()
	assert.Equal(t, "subscriber-state", meta.Key)
	assert.Equal(t, "Magti", meta.Title)
	assert.Equal(t, pluginapi.WidgetKindStat, meta.Kind)
	assert.Equal(t, pluginapi.WidgetSizeS, meta.Size)
	assert.Equal(t, permission.AccountRead, meta.Permission)

	assert.Equal(t, pluginapi.StatAnswer(pluginapi.StatData{
		Value:   "0.03",
		Unit:    "₾",
		Caption: "2589 MB left, renews in 18 days",
		Link:    "/subscriber",
	}), load(t, heldState{state: mobileState(18)}))
}

// MAGTIACC-SC-023: The caption of a home line names the price, a mobile subscriber with
// no plan names none, and the last days read as one day and today.
func TestTheWidgetCaptionFollowsTheKind(t *testing.T) {
	t.Parallel()

	homeLine := model.State{
		Kind:    model.SubscriberKindHomeLine,
		Balance: 2770,
		Plan:    &model.Plan{Price: 4000, DaysLeft: 20},
	}

	noPlan := mobileState(0)
	noPlan.Plan = nil

	cases := map[string]struct {
		state   model.State
		caption string
	}{
		"a home line":     {homeLine, "40.00 ₾ plan, renews in 20 days"},
		"no plan":         {noPlan, "No active plan"},
		"one day left":    {mobileState(1), "2589 MB left, renews in 1 day"},
		"the renewal day": {mobileState(0), "2589 MB left, renews today"},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			answer := load(t, heldState{state: setup.state})
			require.NotNil(t, answer.Stat)
			assert.Equal(t, setup.caption, answer.Stat.Caption)
		})
	}
}

// MAGTIACC-SC-023: With no assignment or no session the widget is not set up, and a
// failure of the portal is the error of the widget.
func TestTheWidgetNamesItsState(t *testing.T) {
	t.Parallel()

	notSetUp := pluginapi.NotSetUpAnswer(
		"Connect the Magti account and assign a subscriber.",
		"/account",
	)

	assert.Equal(t, notSetUp, load(t, heldState{err: errs.ErrNotAssigned}))
	assert.Equal(t, notSetUp, load(t, heldState{err: errs.ErrNotConnected}))

	_, err := widget.NewSubscriberState(heldState{err: errs.ErrRequestFailed}).
		Load(context.Background())
	require.ErrorIs(t, err, errs.ErrRequestFailed)
}
