package widget

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/errs"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/permission"
)

const (
	connectionPage   = "/account"
	transactionsPage = "/transactions"
	dateLayout       = "02/01/2006"
)

// LatestReadingReader reads the latest reading of the assigned customer of the acting
// workspace from the transactions that Maroid holds.
type LatestReadingReader interface {
	LatestReading(ctx context.Context) (*model.Transaction, error)
}

// NewLastReading builds the widget of the latest reading: its usage, its charge, its
// meter value, and its date.
func NewLastReading(readings LatestReadingReader) *pluginapi.FuncWidget {
	return pluginapi.NewWidget(
		pluginapi.WidgetMeta{
			Key:        "last_reading",
			Title:      "Gas usage",
			Kind:       pluginapi.WidgetKindStat,
			Size:       pluginapi.WidgetSizeS,
			Permission: permission.AccountRead,
		},
		func(ctx context.Context) (pluginapi.WidgetAnswer, error) {
			latest, err := readings.LatestReading(ctx)
			if errors.Is(err, errs.ErrNotAssigned) {
				return pluginapi.NotSetUpAnswer(
					"Connect the gas account and assign a customer.",
					connectionPage,
				), nil
			}

			if err != nil {
				return pluginapi.WidgetAnswer{}, fmt.Errorf("reading the latest reading: %w", err)
			}

			if latest == nil {
				return pluginapi.EmptyAnswer("No reading yet."), nil
			}

			return pluginapi.StatAnswer(pluginapi.StatData{
				Value: wholeCubicMetres(latest.Consumption),
				Unit:  "m³",
				Caption: fmt.Sprintf(
					"%s ₾, meter %s m³, %s",
					strconv.FormatFloat(latest.Amount, 'f', 2, 64),
					wholeCubicMetres(latest.MeterValue),
					latest.Date.Format(dateLayout),
				),
				Link: transactionsPage,
			}), nil
		},
	)
}

func wholeCubicMetres(cubicMetres float64) string {
	return strconv.FormatFloat(cubicMetres, 'f', 0, 64)
}
