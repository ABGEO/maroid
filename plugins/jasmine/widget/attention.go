package widget

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/jasmine/model"
	"github.com/abgeo/maroid/plugins/jasmine/permission"
	"github.com/abgeo/maroid/plugins/jasmine/repository"
)

// NewAttention builds the widget of the plants that need care: each care interval
// of the attention list, in its order.
func NewAttention(db *pluginapi.PluginDB) *pluginapi.FuncWidget {
	return pluginapi.NewWidget(
		pluginapi.WidgetMeta{
			Key:        "attention",
			Title:      "Plants that need care",
			Kind:       pluginapi.WidgetKindList,
			Size:       pluginapi.WidgetSizeM,
			Permission: permission.PlantsRead,
		},
		func(ctx context.Context) (pluginapi.WidgetAnswer, error) {
			var intervals []model.CareInterval

			err := db.WithTx(ctx, func(tx *sqlx.Tx) error {
				var listErr error

				intervals, listErr = repository.NewCareInterval(tx).ListAttention(ctx)
				if listErr != nil {
					return fmt.Errorf("listing the care that is due: %w", listErr)
				}

				return nil
			})
			if err != nil {
				return pluginapi.WidgetAnswer{}, fmt.Errorf("reading the attention list: %w", err)
			}

			if len(intervals) == 0 {
				return pluginapi.EmptyAnswer("No plant needs care."), nil
			}

			items := make([]pluginapi.ListItem, 0, len(intervals))
			for i := range intervals {
				items = append(items, itemOf(&intervals[i]))
			}

			return pluginapi.ListAnswer(pluginapi.ListData{Items: items, More: "/attention"}), nil
		},
	)
}

func itemOf(care *model.CareInterval) pluginapi.ListItem {
	var state model.CareState
	if care.State != nil {
		state = *care.State
	}

	return pluginapi.ListItem{
		Label: care.Plant.DisplayID() + " " + care.Plant.CommonName,
		Meta: fmt.Sprintf(
			"%s, %s, every %s, %s",
			strings.ReplaceAll(string(care.Type), "_", " "),
			days(care.DaysSince),
			days(care.Days),
			strings.ReplaceAll(string(state), "_", " "),
		),
		Link: "/plants/" + care.Plant.ID,
	}
}

func days(count int) string {
	if count == 1 {
		return "1 day"
	}

	return fmt.Sprintf("%d days", count)
}
