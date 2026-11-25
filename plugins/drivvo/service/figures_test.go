package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/drivvo/model"
	"github.com/abgeo/maroid/plugins/drivvo/repository"
	"github.com/abgeo/maroid/plugins/drivvo/service"
)

// heldVehicles is a CarLogReader that holds vehicles alone.
type heldVehicles []model.Vehicle

func (h heldVehicles) Vehicles(context.Context) ([]model.Vehicle, model.CarLogSettings, error) {
	return h, model.CarLogSettings{}, nil
}

func (heldVehicles) VehicleRows(context.Context, string) (model.VehicleRows, error) {
	return model.VehicleRows{}, nil
}

func (heldVehicles) LogEntries(
	context.Context, string, model.EntryFilter, page.Request, repository.Order,
) (page.Window[model.EntryRow], model.VehicleRows, error) {
	return page.Window[model.EntryRow]{}, model.VehicleRows{}, nil
}

func (heldVehicles) LogEntryTotals(
	context.Context, string, model.EntryFilter,
) (model.EntryTotals, model.CarLogSettings, error) {
	return model.EntryTotals{}, model.CarLogSettings{}, nil
}

func (heldVehicles) LogEntry(
	context.Context,
	string,
	string,
) (model.EntryRow, model.CarLogSettings, error) {
	return model.EntryRow{}, model.CarLogSettings{}, nil
}

// DRIVVO-SC-029: The default vehicle is the principal one even when another active vehicle
// comes first by name, then the first active one, then the first one.
func TestTheDefaultVehicleIsThePrincipalOne(t *testing.T) {
	t.Parallel()

	astra := model.Vehicle{ID: "a", Name: "Astra", Active: true}
	hatchback := model.Vehicle{ID: "h", Name: nameHatchback, Active: true, Principal: true}
	sedan := model.Vehicle{ID: "s", Name: "Old sedan"}

	cases := map[string]struct {
		vehicles heldVehicles
		want     string
	}{
		"principal": {heldVehicles{astra, hatchback, sedan}, "h"},
		"first active": {
			heldVehicles{astra, {ID: "h", Name: nameHatchback, Active: true}, sedan},
			"a",
		},
		"first": {heldVehicles{sedan}, "s"},
		"none":  {heldVehicles{}, ""},
	}

	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			list, err := service.NewFigures(want.vehicles, time.Now).Vehicles(t.Context())
			require.NoError(t, err)
			assert.Equal(t, want.want, list.Default)
		})
	}
}
