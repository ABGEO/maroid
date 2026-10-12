package widget

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/gwp/errs"
	"github.com/abgeo/maroid/plugins/gwp/model"
	"github.com/abgeo/maroid/plugins/gwp/permission"
)

const (
	connectionPage = "/account"
	dateLayout     = "02/01/2006"
)

// UsageReader reads the last reading of each meter of the assigned customer of the
// acting workspace from the readings that Maroid holds.
type UsageReader interface {
	LastReadingOfEachMeter(ctx context.Context) ([]model.Reading, error)
}

// NewUsage builds the widget of the usage of the last month: one line for each meter
// of the assigned customer.
func NewUsage(readings UsageReader) *pluginapi.FuncWidget {
	return pluginapi.NewWidget(
		pluginapi.WidgetMeta{
			Key:        "usage",
			Title:      "Water usage",
			Kind:       pluginapi.WidgetKindList,
			Size:       pluginapi.WidgetSizeM,
			Permission: permission.AccountRead,
		},
		func(ctx context.Context) (pluginapi.WidgetAnswer, error) {
			last, err := readings.LastReadingOfEachMeter(ctx)
			if errors.Is(err, errs.ErrNotAssigned) {
				return pluginapi.NotSetUpAnswer(
					"Connect the water account and assign a customer.",
					connectionPage,
				), nil
			}

			if err != nil {
				return pluginapi.WidgetAnswer{}, fmt.Errorf("reading the usage: %w", err)
			}

			if len(last) == 0 {
				return pluginapi.EmptyAnswer("No reading yet."), nil
			}

			items := make([]pluginapi.ListItem, 0, len(last))
			for i := range last {
				items = append(items, itemOf(&last[i]))
			}

			return pluginapi.ListAnswer(pluginapi.ListData{Items: items}), nil
		},
	)
}

func itemOf(reading *model.Reading) pluginapi.ListItem {
	return pluginapi.ListItem{
		Label: fmt.Sprintf("Meter %s: %s m³", reading.SerialNumber, volume(reading.Usage())),
		Meta: fmt.Sprintf(
			"%s to %s, %s",
			volume(reading.PreviousValue),
			volume(reading.Value),
			reading.ReadingDate.Format(dateLayout),
		),
	}
}

func volume(cubicMetres float64) string {
	return strconv.FormatFloat(cubicMetres, 'f', 1, 64)
}
