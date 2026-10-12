package widget

import (
	"context"
	"fmt"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/parking/errs"
	"github.com/abgeo/maroid/plugins/parking/model"
	"github.com/abgeo/maroid/plugins/parking/permission"
	"github.com/abgeo/maroid/plugins/parking/service"
)

const connectionPage = "/account"

// NewBalance builds the widget of the balance of the parking account of the acting
// user, which also reads the free parkings left today.
func NewBalance(connections service.ConnectionService) *pluginapi.FuncWidget {
	return pluginapi.NewWidget(
		pluginapi.WidgetMeta{
			Key:        "balance",
			Title:      "Parking balance",
			Kind:       pluginapi.WidgetKindStat,
			Size:       pluginapi.WidgetSizeS,
			Permission: permission.SessionsRead,
		},
		func(ctx context.Context) (pluginapi.WidgetAnswer, error) {
			connection, err := connections.Status(ctx)
			if err != nil {
				return pluginapi.WidgetAnswer{}, fmt.Errorf(
					"reading the parking connection: %w",
					err,
				)
			}

			if connection.State != model.ConnectionStateConnected {
				return pluginapi.NotSetUpAnswer(
					"Connect your parking account.",
					connectionPage,
				), nil
			}

			if connection.Account == nil {
				return pluginapi.WidgetAnswer{}, errs.ErrNoAccount
			}

			return pluginapi.StatAnswer(pluginapi.StatData{
				Value:   fmt.Sprintf("%.2f", connection.Account.Balance),
				Unit:    "GEL",
				Caption: freeParkingsLeft(connection.Account.FreeParkingsLeft),
				Link:    connectionPage,
			}), nil
		},
	)
}

func freeParkingsLeft(count int) string {
	if count == 1 {
		return "1 free parking left today"
	}

	return fmt.Sprintf("%d free parkings left today", count)
}
