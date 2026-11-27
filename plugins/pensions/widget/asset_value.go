package widget

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pensions/model"
	"github.com/abgeo/maroid/plugins/pensions/permission"
)

const (
	accountPage  = "/account"
	overviewPage = "/overview"
	dateLayout   = "02/01/2006"
	lariSign     = "₾"

	// groupSize is the count of digits between two separators of thousands.
	groupSize = 3
)

// SummaryReader reads the stored summary of the acting user and whether they are connected.
type SummaryReader interface {
	Summary(ctx context.Context) (model.SummaryView, error)
}

// NewAssetValue builds the widget of the asset value: the value, the gain, and the date of
// the collection that stored them. A stored summary shows even after the end of the
// connection, because the collected data stays.
func NewAssetValue(summaries SummaryReader) *pluginapi.FuncWidget {
	return pluginapi.NewWidget(
		pluginapi.WidgetMeta{
			Key:        "asset-value",
			Title:      "Pension",
			Kind:       pluginapi.WidgetKindStat,
			Size:       pluginapi.WidgetSizeS,
			Permission: permission.AccountRead,
		},
		func(ctx context.Context) (pluginapi.WidgetAnswer, error) {
			view, err := summaries.Summary(ctx)
			if err != nil {
				return pluginapi.WidgetAnswer{}, fmt.Errorf("reading the summary: %w", err)
			}

			switch {
			case view.Summary != nil:
			case view.Connected:
				return pluginapi.EmptyAnswer(
					"The first collection runs on the first day of the next month.",
				), nil
			default:
				return pluginapi.NotSetUpAnswer("Connect the pension account.", accountPage), nil
			}

			summary := view.Summary
			collectedOn := summary.CollectedAt.In(pluginapi.ActingTimeZone(ctx)).Format(dateLayout)

			return pluginapi.StatAnswer(pluginapi.StatData{
				Value:   Lari(summary.AssetValue),
				Unit:    lariSign,
				Caption: fmt.Sprintf("Gain %s %s, %s", Lari(summary.Gain), lariSign, collectedOn),
				Link:    overviewPage,
			}), nil
		},
	)
}

// Lari writes an amount with two decimals and a comma between each group of thousands,
// 12,480.55.
func Lari(amount float64) string {
	text := strconv.FormatFloat(amount, 'f', 2, 64)

	sign := ""
	if strings.HasPrefix(text, "-") {
		sign, text = "-", text[1:]
	}

	whole, decimals, _ := strings.Cut(text, ".")

	var grouped strings.Builder

	for i, digit := range whole {
		if i > 0 && (len(whole)-i)%groupSize == 0 {
			grouped.WriteByte(',')
		}

		grouped.WriteRune(digit)
	}

	return sign + grouped.String() + "." + decimals
}
