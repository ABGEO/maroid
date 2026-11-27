package cli_test

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/parking/cli"
	"github.com/abgeo/maroid/plugins/parking/errs"
	"github.com/abgeo/maroid/plugins/parking/model"
)

const userA = "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e77"

// recordingCollector records the reach and the acting user of each collection.
type recordingCollector struct {
	mu      sync.Mutex
	reaches []model.Reach
	users   []string
	answer  model.Collection
	err     error
}

func (r *recordingCollector) Collect(
	ctx context.Context,
	reach model.Reach,
) (model.Collection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reaches = append(r.reaches, reach)
	r.users = append(r.users, pluginapi.ActingUserFromContext(ctx))

	return r.answer, r.err
}

func run(t *testing.T, collector cli.Collector, args ...string) (string, error) {
	t.Helper()

	command := cli.NewBackfill(slog.New(slog.DiscardHandler), collector).Command()

	var out bytes.Buffer

	command.SetOut(&out)
	command.SetErr(&out)
	command.SetArgs(args)

	err := command.ExecuteContext(t.Context())

	return out.String(), err
}

// PARKHIST-SC-014: The backfill collects the full history as the user, and prints the
// new rows.
func TestTheBackfillPrintsTheNewRows(t *testing.T) {
	t.Parallel()

	collector := &recordingCollector{answer: model.Collection{Sessions: 438, Transactions: 393}}

	out, err := run(t, collector, "--user", userA)

	require.NoError(t, err)
	assert.Equal(t, "438 new sessions, 393 new transactions\n", out)
	assert.Equal(t, []model.Reach{model.ReachFull()}, collector.reaches)
	assert.Equal(t, []string{userA}, collector.users)
}

// PARKHIST-SC-015: A user with no connection fails the backfill. A user that is not a
// UUID fails before any collection.
func TestTheBackfillFailsForNoConnectionAndABadUser(t *testing.T) {
	t.Parallel()

	collector := &recordingCollector{err: errs.ErrNotConnected}

	_, err := run(t, collector, "--user", userA)
	require.ErrorIs(t, err, errs.ErrNotConnected)

	_, err = run(t, collector, "--user", "abc")
	require.ErrorIs(t, err, errs.ErrInvalidUser)
	assert.Len(t, collector.reaches, 1)
}
