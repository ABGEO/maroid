package cli_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/magti/cli"
	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/internal/fakeportal"
	"github.com/abgeo/maroid/plugins/magti/internal/fakes"
	"github.com/abgeo/maroid/plugins/magti/model"
)

func run(t *testing.T, rig *fakes.Rig, arguments ...string) (string, error) {
	t.Helper()

	command := cli.NewBackfill(slog.New(slog.DiscardHandler), rig.Collector()).Command()
	output := new(bytes.Buffer)
	command.SetOut(output)
	command.SetErr(io.Discard)
	command.SetArgs(arguments)

	err := command.ExecuteContext(t.Context())

	return output.String(), err
}

func periods(rig *fakes.Rig) []string {
	var asked []string

	for _, call := range rig.Portal.CallsTo("/request/statistics.php") {
		if call.Form.Get("step") == "3" && call.Form.Get("key") == "MB" {
			asked = append(asked, call.Form.Get("date[from]")+" "+call.Form.Get("date[to]"))
		}
	}

	return asked
}

// MAGTIACC-SC-019: The command collects each calendar month of the period, cut to the
// period, and prints the counts of each month.
func TestTheBackfillCollectsEachMonth(t *testing.T) {
	t.Parallel()

	rig := fakes.NewRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))
	rig.Portal.Set(func(portal *fakeportal.Portal) {
		portal.Override = func(w http.ResponseWriter, r *http.Request, _ int) bool {
			if r.URL.Path != "/request/statistics.php" || r.PostFormValue("step") != "3" ||
				strings.HasSuffix(r.PostFormValue("date[from]"), "/2026") {
				return false
			}

			_, _ = w.Write([]byte(`{"name":"","key":"","history":[]}`))

			return true
		}
		portal.History[fakeportal.HistoryKey("MB", "expense")] = []any{
			fakeportal.UsageRow(1767304800000, 120, 0),
			fakeportal.UsageRow(1767391200000, 80, 0),
		}
	})

	output, err := run(t, rig,
		"--workspace", fakes.WorkspaceW, "--from", "2025-11-15", "--to", "2026-01-10")
	require.NoError(t, err)

	assert.Equal(t, []string{
		"15/11/2025 30/11/2025",
		"01/12/2025 31/12/2025",
		"01/01/2026 10/01/2026",
	}, periods(rig))
	assert.Equal(t, "2025-11-15 to 2025-11-30: 0 days, 0 new money entries\n"+
		"2025-12-01 to 2025-12-31: 0 days, 0 new money entries\n"+
		"2026-01-01 to 2026-01-10: 2 days, 0 new money entries\n", output)
}

// MAGTIACC-SC-020: A reversed period fails before any call, and a home line or no session
// fails the command with its error.
func TestTheBackfillRefusesWhatItCannotCollect(t *testing.T) {
	t.Parallel()

	rig := fakes.NewRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))

	_, err := run(t, rig,
		"--workspace", fakes.WorkspaceW, "--from", "2026-02-01", "--to", "2026-01-01")
	require.ErrorIs(t, err, errs.ErrReversedPeriod)
	assert.Empty(t, rig.Portal.Calls())

	_, err = run(t, rig, "--workspace", "W", "--from", "2026-01-01", "--to", "2026-01-02")
	require.Error(t, err)
	assert.Empty(t, rig.Portal.Calls())

	homeLine := fakes.NewRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.HomeLine, model.SubscriberKindHomeLine))

	_, err = run(t, homeLine,
		"--workspace", fakes.WorkspaceW, "--from", "2026-01-01", "--to", "2026-01-02")
	require.ErrorIs(t, err, errs.ErrHomeLine)

	notConnected := fakes.NewRig(t, "",
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))

	_, err = run(t, notConnected,
		"--workspace", fakes.WorkspaceW, "--from", "2026-01-01", "--to", "2026-01-02")
	require.ErrorIs(t, err, errs.ErrNotConnected)
}
