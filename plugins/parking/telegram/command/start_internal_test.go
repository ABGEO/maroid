package command

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/parking/dto"
)

// PARKACC-SC-025: A start for a parking account with no vehicle ends with the reason.
func TestValidateStartRefusesAnAccountWithNoVehicle(t *testing.T) {
	t.Parallel()

	message := validateStart(&startState{person: &dto.Person{BalanceAmount: 10}})

	require.Contains(t, message, "no vehicle")
}
