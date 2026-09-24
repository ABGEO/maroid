package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/testdb"
)

// maxDriftSeconds bounds the distance between the stored moment and the moment
// of the write. The trigger and the comparison run milliseconds apart, so a
// drift above this is an offset and not a delay.
const maxDriftSeconds = 5.0

// APIFMT-SC-015: A write in a session zone that is not UTC stores the instant of
// that write.
func TestTheTriggerStoresTheInstantOfTheWrite(t *testing.T) {
	t.Parallel()

	// One zone ahead of UTC and one behind, so a correction that holds for one
	// sign only still fails here.
	zonesAwayFromUTC := []string{"Asia/Tbilisi", "America/New_York"}

	instance := testdb.Start(t)
	require.NoError(t, newMigrator(t, instance).Up())

	for _, zone := range zonesAwayFromUTC {
		t.Run(zone, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			conn, err := instance.DB.Connx(ctx)
			require.NoError(t, err)

			t.Cleanup(func() { _ = conn.Close() })

			_, err = conn.ExecContext(ctx, "SET TIME ZONE '"+zone+"';")
			require.NoError(t, err)

			var id string

			require.NoError(t, conn.GetContext(ctx, &id, `
				INSERT INTO public.users (first_name)
				VALUES ('Temuri')
				RETURNING id;`))

			_, err = conn.ExecContext(ctx, `
				UPDATE public.users SET last_name = 'Takalandze' WHERE id = $1;`, id)
			require.NoError(t, err)

			var drift float64

			require.NoError(t, conn.GetContext(ctx, &drift, `
				SELECT abs(extract(epoch FROM (now() - updated_at)))
				FROM public.users
				WHERE id = $1;`, id))

			require.Less(t, drift, maxDriftSeconds,
				"the trigger stored a moment %.0f seconds from the write", drift)
		})
	}
}
