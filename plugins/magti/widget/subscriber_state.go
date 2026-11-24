package widget

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/model"
	"github.com/abgeo/maroid/plugins/magti/permission"
)

const (
	accountPage    = "/account"
	subscriberPage = "/subscriber"
	unitLari       = "₾"
)

// StateReader reads the subscriber state of the acting workspace from the portal.
type StateReader interface {
	Read(ctx context.Context) (model.State, error)
}

// NewSubscriberState builds the widget of the subscriber state: the balance, with what
// the plan leaves and the days to its renewal.
func NewSubscriberState(states StateReader) *pluginapi.FuncWidget {
	return pluginapi.NewWidget(
		pluginapi.WidgetMeta{
			Key:        "subscriber-state",
			Title:      "Magti",
			Kind:       pluginapi.WidgetKindStat,
			Size:       pluginapi.WidgetSizeS,
			Permission: permission.AccountRead,
		},
		func(ctx context.Context) (pluginapi.WidgetAnswer, error) {
			state, err := states.Read(ctx)
			if errors.Is(err, errs.ErrNotConnected) || errors.Is(err, errs.ErrNotAssigned) {
				return pluginapi.NotSetUpAnswer(
					"Connect the Magti account and assign a subscriber.",
					accountPage,
				), nil
			}

			if err != nil {
				return pluginapi.WidgetAnswer{}, fmt.Errorf("reading the subscriber state: %w", err)
			}

			return pluginapi.StatAnswer(pluginapi.StatData{
				Value:   state.Balance.String(),
				Unit:    unitLari,
				Caption: caption(state),
				Link:    subscriberPage,
			}), nil
		},
	)
}

func caption(state model.State) string {
	switch {
	case state.Plan == nil:
		return "No active plan"
	case state.Kind == model.SubscriberKindHomeLine:
		return fmt.Sprintf(
			"%s %s plan, %s",
			state.Plan.Price,
			unitLari,
			renewal(state.Plan.DaysLeft),
		)
	}

	dataMB := 0.0
	if state.Allowances != nil {
		dataMB = state.Allowances.DataMB
	}

	return fmt.Sprintf(
		"%s MB left, %s",
		strconv.FormatFloat(dataMB, 'f', 0, 64),
		renewal(state.Plan.DaysLeft),
	)
}

func renewal(daysLeft int) string {
	switch daysLeft {
	case 0:
		return "renews today"
	case 1:
		return "renews in 1 day"
	}

	return "renews in " + strconv.Itoa(daysLeft) + " days"
}
