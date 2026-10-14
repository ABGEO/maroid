package cli_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/telmico/cli"
	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/model"
)

const (
	workspaceW = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"

	flagWorkspace = "--workspace"
	flagFrom      = "--from"
	flagTo        = "--to"

	october = "2026-10"
)

// collected is one call of the fake collector, with the scope that it ran in.
type collected struct {
	month     string
	workspace string
	user      string
}

// fakeCollector records each call, and answers err when it is set.
type fakeCollector struct {
	mu    sync.Mutex
	calls []collected
	err   error
}

func (f *fakeCollector) Collect(ctx context.Context, month model.Month) (model.Collection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, collected{
		month:     month.String(),
		workspace: pluginapi.ActingWorkspaceFromContext(ctx),
		user:      pluginapi.ActingUserFromContext(ctx),
	})

	if f.err != nil {
		return model.Collection{}, f.err
	}

	return model.Collection{Count: 3}, nil
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

// ELECACC-SC-019: The command collects each month of the period in the acting workspace
// of the flag, with no acting user, and prints the count of each month.
func TestTheBackfillCollectsEachMonth(t *testing.T) {
	t.Parallel()

	collector := new(fakeCollector)

	output, err := run(
		t,
		collector,
		flagWorkspace,
		workspaceW,
		flagFrom,
		"2025-10",
		flagTo,
		"2026-09",
	)
	require.NoError(t, err)

	require.Len(t, collector.calls, 12)
	assert.Equal(t, "2025-10", collector.calls[0].month)
	assert.Equal(t, "2026-09", collector.calls[11].month)

	for _, call := range collector.calls {
		assert.Equal(t, workspaceW, call.workspace)
		assert.Empty(t, call.user)
	}

	assert.Contains(t, output, "2025-10: 3 transactions\n")
	assert.Contains(t, output, "2026-09: 3 transactions\n")
}

// ELECACC-SC-020: A workspace with no connection or no assignment fails the command
// with the cause, and the first failure stops it.
func TestTheBackfillFailsWhenTheWorkspaceIsNotSetUp(t *testing.T) {
	t.Parallel()

	for _, cause := range []error{errs.ErrNotConnected, errs.ErrNotAssigned} {
		collector := &fakeCollector{err: cause}

		_, err := run(
			t,
			collector,
			flagWorkspace,
			workspaceW,
			flagFrom,
			"2026-01",
			flagTo,
			"2026-03",
		)

		require.ErrorIs(t, err, cause)
		assert.Len(t, collector.calls, 1)
	}
}

// ELECACC-SC-020: A reversed period, a month that does not parse, or a workspace that
// is no UUID fails before any collection.
func TestTheBackfillRefusesBrokenFlags(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		arguments []string
		want      error
	}{
		"reversed": {
			arguments: []string{flagWorkspace, workspaceW, flagFrom, "2026-09", flagTo, "2026-08"},
			want:      errs.ErrReversedPeriod,
		},
		"day in month": {
			arguments: []string{
				flagWorkspace,
				workspaceW,
				flagFrom,
				"2026-09-01",
				flagTo,
				"2026-10",
			},
		},
		"no zero": {
			arguments: []string{flagWorkspace, workspaceW, flagFrom, "2026-9", flagTo, october},
		},
		"no UUID": {
			arguments: []string{flagWorkspace, "W", flagFrom, "2026-09", flagTo, october},
		},
		"no flags": {arguments: nil},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			collector := new(fakeCollector)

			_, err := run(t, collector, setup.arguments...)

			require.Error(t, err)

			if setup.want != nil {
				require.ErrorIs(t, err, setup.want)
			}

			assert.Empty(t, collector.calls)
		})
	}
}
