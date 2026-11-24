package service_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/magti/config"
	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/internal/fakeportal"
	"github.com/abgeo/maroid/plugins/magti/internal/fakes"
	"github.com/abgeo/maroid/plugins/magti/model"
	"github.com/abgeo/maroid/plugins/magti/service"
)

// collectorRig runs a collector against the fake portal and the fakes.
type collectorRig struct {
	portal    *fakeportal.Portal
	secrets   *fakes.Secrets
	store     *fakes.Store
	collector *service.Collector
}

func newCollectorRig(t *testing.T, cookie string, store *fakes.Store) *collectorRig {
	t.Helper()

	portal := fakeportal.New(t)
	client := service.NewAPIClient(&config.Config{
		BaseURL:        portal.URL(),
		RequestTimeout: 5 * time.Second,
	})
	secrets := fakes.NewSecrets(cookie)
	account := service.NewAccount(client, fakes.Credentials(), secrets)

	return &collectorRig{
		portal:    portal,
		secrets:   secrets,
		store:     store,
		collector: service.NewCollector(account, client, store),
	}
}

func previousDay() model.Period {
	return model.DayPeriod(model.Day{Year: 2026, Month: time.November, Day: 24})
}

// MAGTIACC-SC-013: A collection of the previous day sends five step 3 calls with the day
// as both bounds.
func TestACollectionReadsTheFiveSeriesOfTheDay(t *testing.T) {
	t.Parallel()

	rig := newCollectorRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))

	_, err := rig.collector.Collect(fakes.Acting(t.Context()), previousDay())
	require.NoError(t, err)

	var series []string

	for _, call := range rig.portal.CallsTo("/request/statistics.php") {
		if call.Form.Get("step") != "3" {
			continue
		}

		assert.Equal(t, "24/11/2026", call.Form.Get("date[from]"))
		assert.Equal(t, "24/11/2026", call.Form.Get("date[to]"))
		assert.Equal(t, fakeportal.Mobile, call.Form.Get("mobile-number"))
		series = append(series, call.Form.Get("key")+"/"+call.Form.Get("charge-type"))
	}

	assert.ElementsMatch(t, []string{
		"MB/expense", "MIN/expense", "SMS/expense", "GEL/expense", "GEL/income",
	}, series)
}

// MAGTIACC-SC-014: With no session, no assignment, or a home line, a collection answers
// its error and calls no portal.
func TestACollectionWithNothingToCollectCallsNoPortal(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		cookie string
		store  *fakes.Store
		want   error
	}{
		"no session": {
			store: fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile),
			want:  errs.ErrNotConnected,
		},
		"no assignment": {
			cookie: fakeportal.SessionCookie,
			store:  fakes.NewStore("", ""),
			want:   errs.ErrNotAssigned,
		},
		"a home line": {
			cookie: fakeportal.SessionCookie,
			store:  fakes.NewStore(fakeportal.HomeLine, model.SubscriberKindHomeLine),
			want:   errs.ErrHomeLine,
		},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rig := newCollectorRig(t, setup.cookie, setup.store)

			_, err := rig.collector.Collect(fakes.Acting(t.Context()), previousDay())
			require.ErrorIs(t, err, setup.want)
			assert.Empty(t, rig.portal.Calls())
		})
	}
}

// MAGTIACC-SC-015: A subscriber that the account no longer holds fails the collection, and
// stores nothing.
func TestAnAbsentSubscriberStoresNothing(t *testing.T) {
	t.Parallel()

	rig := newCollectorRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))
	rig.portal.Set(func(p *fakeportal.Portal) {
		p.Subscribers = p.Subscribers[1:]
		p.Selected = fakeportal.HomeLine
	})

	_, err := rig.collector.Collect(fakes.Acting(t.Context()), previousDay())
	require.ErrorIs(t, err, errs.ErrSubscriberAbsent)
	assert.Zero(t, rig.store.Writes)
	assert.Empty(t, rig.portal.CallsTo("/request/statistics.php"))
}

// MAGTIACC-SC-009: A collection whose sign in again meets a refusal answers that refusal,
// and the session goes.
func TestARefusedSignInEndsTheCollection(t *testing.T) {
	t.Parallel()

	cases := map[string]error{
		"invalid-credentials": errs.ErrCredentialsRejected,
		"locked":              errs.ErrAccountLocked,
	}

	for message, want := range cases {
		t.Run(message, func(t *testing.T) {
			t.Parallel()

			rig := newCollectorRig(t, "ENDED",
				fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))
			rig.portal.Set(func(p *fakeportal.Portal) { p.LogInMessage = message })

			_, err := rig.collector.Collect(fakes.Acting(t.Context()), previousDay())
			require.ErrorIs(t, err, want)
			assert.Empty(t, rig.secrets.Session())
			assert.Zero(t, rig.store.Writes)
		})
	}
}

// MAGTIACC-SC-008: An ended session signs in once, stores the new session, and the
// collection runs again from its start.
func TestAnEndedSessionSignsInOnce(t *testing.T) {
	t.Parallel()

	rig := newCollectorRig(t, "ENDED",
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))

	_, err := rig.collector.Collect(fakes.Acting(t.Context()), previousDay())
	require.NoError(t, err)

	assert.Len(t, rig.portal.CallsTo("/request/log-in.php"), 1)
	assert.Equal(t, fakeportal.SessionCookie, rig.secrets.Cookie())
	assert.Equal(t, 1, rig.store.Writes)
}

// MAGTIACC-SC-016, MAGTIACC-SC-017: A collection maps the rows of the portal to the
// traffic of each day and to the money entries.
func TestACollectionMapsTheRows(t *testing.T) {
	t.Parallel()

	rig := newCollectorRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))
	rig.portal.Set(func(portal *fakeportal.Portal) {
		portal.History[fakeportal.HistoryKey("MB", "expense")] = []any{
			fakeportal.UsageRow(1795464000000, 812.4, 0),
			fakeportal.UsageRow(0, 3, 0),
			fakeportal.TotalRow(812.4),
		}
		portal.History[fakeportal.HistoryKey("MIN", "expense")] = []any{
			fakeportal.UsageRow(1795464000000, 12.5, 3),
		}
		portal.History[fakeportal.HistoryKey("GEL", "income")] = []any{
			fakeportal.MoneyRow("13/11/2026 01:50:48", "17.30", "Top up"),
		}
		portal.History[fakeportal.HistoryKey("GEL", "expense")] = []any{
			fakeportal.MoneyRow("13/11/2026 01:50:49", "2.00", "Plan"),
			fakeportal.MoneyRow("13/11/2026 01:50:49", "2.00", "SMS"),
		}
	})

	ctx := fakes.Acting(t.Context())

	first, err := rig.collector.Collect(ctx, previousDay())
	require.NoError(t, err)
	assert.Equal(t, model.Collection{Days: 1, NewEntries: 3}, first)

	second, err := rig.collector.Collect(ctx, previousDay())
	require.NoError(t, err)
	assert.Equal(t, 0, second.NewEntries)

	day := rig.store.Days["2026-11-24"]
	assert.Equal(t, fakeportal.Mobile, day.SubscriberNumber)
	assert.InDelta(t, 812.4, *day.DataMB, 1e-9)
	assert.Equal(t, 750, *day.CallOutSeconds)
	assert.Equal(t, 180, *day.CallInSeconds)
	assert.Nil(t, day.SMSSent)
}

// MAGTIACC-SC-031: Two calls of one workspace at the same time never select over each
// other, so the selection of a subscriber comes directly before the read of its state.
func TestCallsOfOneWorkspaceRunOneAfterTheOther(t *testing.T) {
	t.Parallel()

	rig := newCollectorRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))
	rig.portal.Set(func(p *fakeportal.Portal) {
		p.Override = func(_ http.ResponseWriter, r *http.Request, _ int) bool {
			if r.URL.Path == "/mymagti" {
				time.Sleep(100 * time.Millisecond)
			}

			return false
		}
	})

	ctx := fakes.Acting(t.Context())
	done := make(chan error, 2)

	for range 2 {
		go func() {
			_, err := rig.collector.Collect(ctx, previousDay())
			done <- err
		}()
	}

	require.NoError(t, <-done)
	require.NoError(t, <-done)

	calls := rig.portal.Calls()
	for index, call := range calls {
		if call.Path != "/request/accept-terms.php" {
			continue
		}

		require.Positive(t, index)
		assert.NotEqual(t, "/mymagti", calls[index-1].Path,
			"a selection of another call ran between the menu and the statistics")
	}
}
