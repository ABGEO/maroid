//go:build measure

package job_test

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/jasmine/config"
	"github.com/abgeo/maroid/plugins/jasmine/job"
)

const (
	measuredWorkspaces = 50
	sendDelay          = 300 * time.Millisecond
)

// measuredGarden holds 500 active plants in 10 environments, one care interval of
// each of the 8 types for each plant, and 200 entries for each plant spread over
// 365 days.
const measuredGarden = `
INSERT INTO environments (number, name)
SELECT n, 'Room ' || n FROM generate_series(1, 10) AS n;

INSERT INTO plants (number, common_name, environment_id)
SELECT n, 'Plant ' || n, e.id
FROM generate_series(1, 500) AS n
JOIN environments e ON e.number = 1 + n % 10;

INSERT INTO care_intervals (plant_id, type, days)
SELECT p.id, t.type, 1 + (p.number + t.position) % 30
FROM plants p
CROSS JOIN unnest(ARRAY['watering', 'fertilizing', 'pruning', 'repotting',
                        'pest_treatment', 'cleaning', 'misting', 'other'])
     WITH ORDINALITY AS t (type, position);

INSERT INTO maintenance_entries (plant_id, type, performed_at)
SELECT p.id,
       (ARRAY['watering', 'fertilizing', 'pruning', 'repotting',
              'pest_treatment', 'cleaning', 'misting', 'other'])[1 + (p.number + s) % 8],
       now() - make_interval(secs => (p.number * 7919 + s * 104729) % 31536000)
FROM plants p
CROSS JOIN generate_series(1, 200) AS s;`

// slowSender waits for each send, as a slow notification route does, and records
// the moment the last send ends.
type slowSender struct {
	mu       sync.Mutex
	sends    int
	lastSent time.Time
}

var _ pluginapi.PluginNotifier = (*slowSender)(nil)

func (d *slowSender) Notify(
	_ context.Context,
	_ string,
	_ pluginapi.Message,
) error {
	time.Sleep(sendDelay)

	d.mu.Lock()
	defer d.mu.Unlock()

	d.sends++
	d.lastSent = time.Now()

	return nil
}

// GARDEN-SC-033: The reminder of 50 workspaces of 500 plants each, run one
// workspace after the other with a notifier that takes 300 ms for each send,
// reaches the notifier for the last workspace in less than 60 seconds.
//
//nolint:paralleltest // a measurement runs alone, so no other test takes its time
func TestTheRemindersOfFiftyWorkspacesEndWithinOneMinute(t *testing.T) {
	database := gardenUnderTest(t, nil)

	workspaces := []string{workspaceH, workspaceG}

	require.NoError(t, database.WithTx(actingIn(workspaceH), func(tx *sqlx.Tx) error {
		var added []string
		if err := tx.Select(
			&added,
			`INSERT INTO public.workspaces (id) SELECT uuidv7() FROM generate_series(1, $1) RETURNING id`,
			measuredWorkspaces-len(workspaces),
		); err != nil {
			return fmt.Errorf("adding the workspaces: %w", err)
		}

		workspaces = append(workspaces, added...)

		return nil
	}))

	for _, workspace := range workspaces {
		require.NoError(t, database.WithTx(actingIn(workspace), func(tx *sqlx.Tx) error {
			if _, err := tx.Exec(measuredGarden); err != nil {
				return fmt.Errorf("seeding the measured garden: %w", err)
			}

			return nil
		}))
	}

	require.NoError(t, database.WithTx(actingIn(workspaceH), func(tx *sqlx.Tx) error {
		if _, err := tx.Exec("ANALYZE"); err != nil {
			return fmt.Errorf("analyzing the tables: %w", err)
		}

		return nil
	}))

	sink := &slowSender{}
	settings := &config.Config{
		CronSchedule: config.CronSchedule{CareReminder: "0 9 * * *"},
	}
	notifier := sink
	reminder := job.NewCareReminder(settings, slog.New(slog.DiscardHandler), database, notifier)

	start := time.Now()

	for _, workspace := range workspaces {
		require.NoError(t, reminder.Run(actingIn(workspace)))
	}

	elapsed := sink.lastSent.Sub(start)

	t.Logf(
		"the reminders of %d workspaces ended %s after the first run started",
		len(workspaces),
		elapsed,
	)
	require.Equal(t, measuredWorkspaces, sink.sends)
	assert.Less(t, elapsed, time.Minute)
}
