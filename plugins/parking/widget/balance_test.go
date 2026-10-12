package widget_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/parking/model"
	"github.com/abgeo/maroid/plugins/parking/service"
	"github.com/abgeo/maroid/plugins/parking/widget"
)

var errServiceDown = errors.New("the parking service answered 503")

// fakeConnections answers each Status from the list in turn.
type fakeConnections struct {
	service.ConnectionService

	statuses []*model.Connection
	err      error
}

func (f *fakeConnections) Status(context.Context) (*model.Connection, error) {
	if f.err != nil {
		return nil, f.err
	}

	status := f.statuses[0]
	f.statuses = f.statuses[1:]

	return status, nil
}

func connected(balance float64, freeLeft int) *model.Connection {
	return &model.Connection{
		State: model.ConnectionStateConnected,
		Account: &model.Account{
			FirstName:        "Nino",
			LastName:         "Beridze",
			Balance:          balance,
			FreeParkingsLeft: freeLeft,
		},
	}
}

// WIDGET-SC-014: A connected account answers its balance in GEL, the free parkings
// left today, and the connection page.
func TestTheBalanceWidgetAnswersTheBalance(t *testing.T) {
	t.Parallel()

	balance := widget.NewBalance(&fakeConnections{
		statuses: []*model.Connection{connected(4.5, 4), connected(4.5, 1)},
	})

	first, err := balance.Load(t.Context())
	require.NoError(t, err)

	second, err := balance.Load(t.Context())
	require.NoError(t, err)

	require.Equal(t, pluginapi.StatAnswer(pluginapi.StatData{
		Value:   "4.50",
		Unit:    "GEL",
		Caption: "4 free parkings left today",
		Link:    "/account",
	}), first)
	require.Equal(t, "1 free parking left today", second.Stat.Caption)

	require.Equal(t, pluginapi.WidgetMeta{
		Key:        "balance",
		Title:      "Parking balance",
		Kind:       pluginapi.WidgetKindStat,
		Size:       pluginapi.WidgetSizeS,
		Permission: "sessions.read",
	}, balance.Meta())
}

// WIDGET-SC-015: With no connection the widget asks for one, and a failure of the
// parking service answers the error.
func TestTheBalanceWidgetAsksForAConnection(t *testing.T) {
	t.Parallel()

	notConnected := widget.NewBalance(&fakeConnections{
		statuses: []*model.Connection{{State: model.ConnectionStateNotConnected}},
	})

	answer, err := notConnected.Load(t.Context())
	require.NoError(t, err)
	require.Equal(t, pluginapi.NotSetUpAnswer("Connect your parking account.", "/account"), answer)

	_, err = widget.NewBalance(&fakeConnections{err: errServiceDown}).Load(t.Context())
	require.ErrorIs(t, err, errServiceDown)
}
