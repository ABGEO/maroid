package job_test

import (
	"bytes"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/internal/fakeportal"
	"github.com/abgeo/maroid/plugins/magti/internal/fakes"
	"github.com/abgeo/maroid/plugins/magti/job"
	"github.com/abgeo/maroid/plugins/magti/model"
)

// runAt is 2026-11-25 09:00 in Tbilisi.
func runAt() time.Time {
	return time.Date(2026, time.November, 25, 5, 0, 0, 0, time.UTC)
}

func jobOf(rig *fakes.Rig) (*job.TrafficCollector, *bytes.Buffer) {
	logs := new(bytes.Buffer)
	logger := slog.New(slog.NewTextHandler(logs, nil))

	return job.NewTrafficCollector(rig.Config, logger, rig.Collector(), runAt), logs
}

// MAGTIACC-SC-013: The job runs at 09:00 each day, for the previous day of Tbilisi.
func TestTheJobCollectsThePreviousDay(t *testing.T) {
	t.Parallel()

	rig := fakes.NewRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))
	collector, _ := jobOf(rig)

	assert.Equal(t, "0 9 * * *", collector.Meta().Schedule)
	require.NoError(t, collector.Run(fakes.Acting(t.Context())))

	steps := 0

	for _, call := range rig.Portal.CallsTo("/request/statistics.php") {
		if call.Form.Get("step") == "3" {
			steps++

			assert.Equal(t, "24/11/2026", call.Form.Get("date[from]"))
			assert.Equal(t, "24/11/2026", call.Form.Get("date[to]"))
		}
	}

	assert.Equal(t, 5, steps)
}

// MAGTIACC-SC-014: A workspace with no session, no assignment, or a home line is skipped
// with no call of the portal.
func TestTheJobSkipsAWorkspaceWithNothingToCollect(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		cookie string
		store  *fakes.Store
	}{
		"no session": {store: fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile)},
		"no assignment": {
			cookie: fakeportal.SessionCookie,
			store:  fakes.NewStore("", ""),
		},
		"a home line": {
			cookie: fakeportal.SessionCookie,
			store:  fakes.NewStore(fakeportal.HomeLine, model.SubscriberKindHomeLine),
		},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rig := fakes.NewRig(t, setup.cookie, setup.store)
			collector, logs := jobOf(rig)

			require.NoError(t, collector.Run(fakes.Acting(t.Context())))
			assert.Empty(t, rig.Portal.Calls())
			assert.Contains(t, logs.String(), "skipping the workspace")
		})
	}
}

// MAGTIACC-SC-009: A refused sign in fails the run with its error, and the session goes.
func TestTheJobReportsARefusedSignIn(t *testing.T) {
	t.Parallel()

	cases := map[string]error{
		"invalid-credentials": errs.ErrCredentialsRejected,
		"locked":              errs.ErrAccountLocked,
	}

	for message, want := range cases {
		t.Run(message, func(t *testing.T) {
			t.Parallel()

			rig := fakes.NewRig(t, "ENDED",
				fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))
			rig.Portal.Set(func(portal *fakeportal.Portal) { portal.LogInMessage = message })
			collector, _ := jobOf(rig)

			require.ErrorIs(t, collector.Run(fakes.Acting(t.Context())), want)
			assert.Empty(t, rig.Secrets.Session())
		})
	}
}

// MAGTIACC-SC-015: An assigned subscriber that the account no longer holds fails the run,
// and stores nothing.
func TestTheJobReportsAnAbsentSubscriber(t *testing.T) {
	t.Parallel()

	rig := fakes.NewRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))
	rig.Portal.Set(func(portal *fakeportal.Portal) {
		portal.Subscribers = portal.Subscribers[1:]
		portal.Selected = fakeportal.HomeLine
	})
	collector, _ := jobOf(rig)

	require.ErrorIs(t, collector.Run(fakes.Acting(t.Context())), errs.ErrSubscriberAbsent)
	assert.Zero(t, rig.Store.Writes)
}
