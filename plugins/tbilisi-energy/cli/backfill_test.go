package cli_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/cli"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/errs"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
)

const (
	workspaceW = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"

	flagWorkspace = "--workspace"
	flagFrom      = "--from"
	flagTo        = "--to"
	newYear       = "2026-01-01"
)

// collected is one call of the fake collector, with the scope that it ran in.
type collected struct {
	period    string
	workspace string
	user      string
}

// fakeCollector records each call, and answers err when it is set.
type fakeCollector struct {
	mu    sync.Mutex
	calls []collected
	err   error
}

func (f *fakeCollector) Collect(
	ctx context.Context,
	period model.Period,
) (model.Collection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, collected{
		period:    period.From.Format(time.DateOnly) + ".." + period.To.Format(time.DateOnly),
		workspace: pluginapi.ActingWorkspaceFromContext(ctx),
		user:      pluginapi.ActingUserFromContext(ctx),
	})

	if f.err != nil {
		return model.Collection{}, f.err
	}

	return model.Collection{Count: 2}, nil
}

func run(t *testing.T, collector *fakeCollector, arguments ...string) (string, error) {
	t.Helper()

	command := cli.NewBackfill(slog.New(slog.DiscardHandler), collector).Command()
	output := new(bytes.Buffer)
	command.SetOut(output)
	command.SetErr(io.Discard)
	command.SetArgs(arguments)

	err := command.ExecuteContext(t.Context())

	return output.String(), err
}

// GASACC-SC-019: The command collects each calendar month of the period in the acting
// workspace of the flag, with no acting user.
func TestTheBackfillCollectsEachMonth(t *testing.T) {
	t.Parallel()

	collector := new(fakeCollector)

	output, err := run(t, collector,
		flagWorkspace, workspaceW, flagFrom, "2025-10-15", flagTo, "2026-01-10")

	require.NoError(t, err)
	require.Equal(t, []collected{
		{period: "2025-10-15..2025-10-31", workspace: workspaceW},
		{period: "2025-11-01..2025-11-30", workspace: workspaceW},
		{period: "2025-12-01..2025-12-31", workspace: workspaceW},
		{period: "2026-01-01..2026-01-10", workspace: workspaceW},
	}, collector.calls)
	require.Contains(t, output, "2025-10-15..2025-10-31: 2 transactions")
}

// GASACC-SC-020: A reversed period, a date that does not parse, or a workspace that is
// not a UUID fails before any collection.
func TestTheBackfillRefusesABadFlag(t *testing.T) {
	t.Parallel()

	cases := map[string][]string{
		"reversed period": {
			flagWorkspace,
			workspaceW,
			flagFrom,
			"2026-02-01",
			flagTo,
			newYear,
		},
		"bad date": {
			flagWorkspace,
			workspaceW,
			flagFrom,
			"2026-13-01",
			flagTo,
			"2026-12-31",
		},
		"bad workspace": {flagWorkspace, "abc", flagFrom, newYear, flagTo, "2026-01-31"},
		"no workspace":  {flagFrom, newYear, flagTo, "2026-01-31"},
	}

	for name, arguments := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			collector := new(fakeCollector)

			_, err := run(t, collector, arguments...)

			require.Error(t, err)
			require.Empty(t, collector.calls)
		})
	}
}

// GASACC-SC-021: A workspace with no connection or no assigned customer fails the
// command, which stops at the first month.
func TestTheBackfillFailsWhenTheWorkspaceIsNotSetUp(t *testing.T) {
	t.Parallel()

	for _, cause := range []error{errs.ErrNotConnected, errs.ErrNotAssigned} {
		collector := &fakeCollector{err: cause}

		_, err := run(t, collector,
			flagWorkspace, workspaceW, flagFrom, "2025-10-15", flagTo, "2026-01-10")

		require.ErrorIs(t, err, cause)
		require.Len(t, collector.calls, 1)
	}
}
