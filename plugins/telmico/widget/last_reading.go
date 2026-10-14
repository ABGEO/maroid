package widget

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/model"
	"github.com/abgeo/maroid/plugins/telmico/permission"
)

const (
	connectionPage   = "/account"
	transactionsPage = "/transactions"
	dateLayout       = "02/01/2006"
)

// LatestReadingReader reads the charges of the latest reading of the assigned customer
// of the acting workspace from the transactions that Maroid holds.
type LatestReadingReader interface {
	LatestReading(ctx context.Context) ([]model.Transaction, error)
}

// NewLastReading builds the widget of the latest reading: its usage, its charge, its
// meter value, and its date.
func NewLastReading(readings LatestReadingReader) *pluginapi.FuncWidget {
	return pluginapi.NewWidget(
		pluginapi.WidgetMeta{
			Key:        "last-reading",
			Title:      "Electricity usage",
			Kind:       pluginapi.WidgetKindStat,
			Size:       pluginapi.WidgetSizeS,
			Permission: permission.AccountRead,
		},
		func(ctx context.Context) (pluginapi.WidgetAnswer, error) {
			charges, err := readings.LatestReading(ctx)
			if errors.Is(err, errs.ErrNotAssigned) {
				return pluginapi.NotSetUpAnswer(
					"Connect the electricity account and assign a customer.",
					connectionPage,
				), nil
			}

			if err != nil {
				return pluginapi.WidgetAnswer{}, fmt.Errorf("reading the latest reading: %w", err)
			}

			readingsOf := model.ReadingsOf(charges)
			if len(readingsOf) == 0 {
				return pluginapi.EmptyAnswer("No reading yet."), nil
			}

			latest := readingsOf[0]

			return pluginapi.StatAnswer(pluginapi.StatData{
				Value: model.KilowattHours(latest.Usage),
				Unit:  "kWh",
				Caption: fmt.Sprintf(
					"%s ₾, meter %s kWh, %s",
					strconv.FormatFloat(latest.Charge, 'f', 2, 64),
					model.KilowattHours(latest.MeterValue),
					latest.Date.Format(dateLayout),
				),
				Link: transactionsPage,
			}), nil
		},
	)
}
