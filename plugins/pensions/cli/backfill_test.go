package cli_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pensions/cli"
	"github.com/abgeo/maroid/plugins/pensions/errs"
	"github.com/abgeo/maroid/plugins/pensions/model"
)

const (
	userA = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"

	monthAugust  = "2026-08"
	monthOctober = "2026-10"
)

// recordingCollector records each month and the acting user of each collection.
type recordingCollector struct {
	mu     sync.Mutex
	months []model.Month
	users  []string
	err    error
}

func (r *recordingCollector) Collect(
	ctx context.Context,
	month model.Month,
) (model.Collection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.months = append(r.months, month)
	r.users = append(r.users, pluginapi.ActingUserFromContext(ctx))

	if r.err != nil {
		return model.Collection{}, r.err
	}

	return model.Collection{Contributions: 3, AssetValues: 2, UnitPrices: 1985, Shares: 1}, nil
}

// clock reads 27 November 2026, at noon of UTC.
func clock() time.Time {
	return time.Date(2026, time.November, 27, 12, 0, 0, 0, time.UTC)
}

func run(t *testing.T, collector cli.Collector, args ...string) (string, error) {
	t.Helper()

	command := cli.NewBackfill(slog.New(slog.DiscardHandler), collector, clock).Command()

	var out bytes.Buffer

	command.SetOut(&out)
	command.SetErr(&out)
	command.SetArgs(args)

	err := command.ExecuteContext(t.Context())

	return out.String(), err
}

func monthOf(month time.Month) model.Month {
	return model.Month{Year: 2026, Month: month}
}

// PENSION-SC-025: The backfill collects each month of the period as the user, and prints
// one line for each month.
func TestTheBackfillCollectsEachMonthAsTheUser(t *testing.T) {
	t.Parallel()

	collector := &recordingCollector{}

	out, err := run(t, collector, "--user", userA, "--from", monthAugust, "--to", monthOctober)

	require.NoError(t, err)
	assert.Equal(
		t,
		[]model.Month{monthOf(time.August), monthOf(time.September), monthOf(time.October)},
		collector.months,
	)
	assert.Equal(t, []string{userA, userA, userA}, collector.users)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.Len(t, lines, 3)
	assert.Equal(
		t,
		"2026-08: 3 contributions, 2 asset values, 1985 unit prices, 1 shares",
		lines[0],
	)
}

// PENSION-SC-026: A reversed period, and a period whose last month has not ended for
// every user, fail before any collection.
func TestTheBackfillRefusesAPeriodThatIsNotOver(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		from, to string
		want     error
	}{
		"reversed":  {from: monthOctober, to: monthAugust, want: errs.ErrReversedPeriod},
		"not ended": {from: monthOctober, to: "2026-11", want: errs.ErrPeriodNotEnded},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			collector := &recordingCollector{}

			_, err := run(t, collector, "--user", userA, "--from", setup.from, "--to", setup.to)

			require.ErrorIs(t, err, setup.want)
			assert.Empty(t, collector.months)
		})
	}
}

// PENSION-SC-026: A bad user or a month that does not parse fails before any collection.
func TestTheBackfillRefusesABadFlag(t *testing.T) {
	t.Parallel()

	for name, args := range map[string][]string{
		"user":  {"--user", "nobody", "--from", monthAugust, "--to", monthOctober},
		"month": {"--user", userA, "--from", "2026-13", "--to", monthOctober},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			collector := &recordingCollector{}

			_, err := run(t, collector, args...)

			require.Error(t, err)
			assert.Empty(t, collector.months)
		})
	}
}

// PENSION-SC-027: A user with no connection fails the backfill at the first month.
func TestTheBackfillFailsForAUserWithNoConnection(t *testing.T) {
	t.Parallel()

	collector := &recordingCollector{err: errs.ErrNotConnected}

	_, err := run(t, collector, "--user", userA, "--from", monthAugust, "--to", monthOctober)

	require.ErrorIs(t, err, errs.ErrNotConnected)
	assert.Equal(t, []model.Month{monthOf(time.August)}, collector.months)
}
