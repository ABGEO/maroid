package migrator

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

const (
	// planAttempts is how often the test rebuilds one plan. The order of a Go map
	// is random for each iteration, so a single build proves nothing.
	planAttempts = 10

	pluginGWP     = "dev.maroid.gwp"
	pluginJasmine = "dev.maroid.jasmine"
	pluginTelasi  = "dev.maroid.telasi"
)

// newPlanner returns a Migrator whose registry holds the core and three plugins,
// registered out of order.
func newPlanner(t *testing.T) *Migrator {
	t.Helper()

	migrationRegistry := registry.NewMigrationRegistry()

	for _, component := range []string{pluginTelasi, TargetCore, pluginJasmine, pluginGWP} {
		require.NoError(t, migrationRegistry.Register(component, fstest.MapFS{}))
	}

	return &Migrator{migrationRegistry: migrationRegistry}
}

// APIFMT-SC-016: The plan applies the core before any plugin, and it answers one
// order however the map iterates.
func TestThePlanAppliesTheCoreFirst(t *testing.T) {
	t.Parallel()

	planner := newPlanner(t)
	want := []string{TargetCore, pluginGWP, pluginJasmine, pluginTelasi}

	for range planAttempts {
		plan, err := planner.buildMigrationPlan(TargetAll)
		require.NoError(t, err)
		require.Equal(t, want, plan.order)
	}
}

// APIFMT-SC-016: A plan that names one target holds that target alone, so the
// order of the whole set does not reach it.
func TestThePlanOfOneTargetHoldsThatTarget(t *testing.T) {
	t.Parallel()

	planner := newPlanner(t)

	corePlan, err := planner.buildMigrationPlan(TargetCore)
	require.NoError(t, err)
	require.Equal(t, []string{TargetCore}, corePlan.order)

	pluginPlan, err := planner.buildMigrationPlan(pluginJasmine)
	require.NoError(t, err)
	require.Equal(t, []string{pluginJasmine}, pluginPlan.order)

	_, err = planner.buildMigrationPlan("dev.maroid.absent")
	require.Error(t, err)
}
