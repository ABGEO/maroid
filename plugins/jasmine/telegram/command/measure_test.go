//go:build measure

package command_test

import (
	"context"
	"testing"
	"time"

	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/jasmine/telegram/command"
)

const (
	measuredRuns = 5
	sendDelay    = 100 * time.Millisecond
	answerBound  = 2 * time.Second
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

// slowBot waits for each send, as a slow Telegram does.
type slowBot struct {
	bot
}

func (b *slowBot) SendMessage(
	ctx context.Context,
	params *telego.SendMessageParams,
) (*telego.Message, error) {
	time.Sleep(sendDelay)

	return b.bot.SendMessage(ctx, params)
}

// GARDEN-SC-040: The command of a workspace with 500 plants and 100000 entries
// returns from its last send in less than 2 seconds.
//
//nolint:paralleltest // a measurement runs alone, so no other test takes its time
func TestTheCareCommandAnswersWithinTwoSeconds(t *testing.T) {
	database := gardenUnderTest(t, map[string]string{workspaceH: measuredGarden})

	var slowest time.Duration

	for range measuredRuns {
		sink := &slowBot{}
		start := time.Now()

		require.NoError(
			t,
			command.NewCare(database, sink, pages{}, pluginapi.ParsePluginID(pluginID)).
				Reply(actingIn(t, workspaceH), commandMessage()),
		)

		slowest = max(slowest, time.Since(start))

		require.NotEmpty(t, sink.texts)
		t.Logf("messages: %d", len(sink.texts))
	}

	t.Logf("slowest answer: %s", slowest)
	assert.Less(t, slowest, answerBound)
}
