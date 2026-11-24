package service_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/magti/config"
	"github.com/abgeo/maroid/plugins/magti/dto"
	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/internal/fakeportal"
	"github.com/abgeo/maroid/plugins/magti/model"
	"github.com/abgeo/maroid/plugins/magti/service"
)

const (
	username = "599000001"
	password = "secret-password"
)

func newClient(t *testing.T) (*service.APIClient, *fakeportal.Portal) {
	t.Helper()

	portal := fakeportal.New(t)

	return service.NewAPIClient(&config.Config{
		BaseURL:        portal.URL(),
		RequestTimeout: 5 * time.Second,
	}), portal
}

func validSession() model.Session {
	return model.Session{
		Cookies: map[string]string{fakeportal.CookieName: fakeportal.SessionCookie},
	}
}

// MAGTIACC-SC-005: The sign in sends the subscriber, the password, and the token of the
// form, and the session holds the cookie that the portal sets.
func TestTheSignInSendsTheFormAndKeepsTheCookie(t *testing.T) {
	t.Parallel()

	client, portal := newClient(t)
	portal.Set(func(p *fakeportal.Portal) { p.Valid = map[string]bool{} })

	session, err := client.SignIn(context.Background(), username, password)
	require.NoError(t, err)

	assert.Equal(t, fakeportal.SessionCookie, session.Cookies[fakeportal.CookieName])

	signIns := portal.CallsTo("/request/log-in.php")
	require.Len(t, signIns, 1)
	assert.Equal(t, username, signIns[0].Form.Get("subscriber"))
	assert.Equal(t, password, signIns[0].Form.Get("password"))
	assert.Equal(t, fakeportal.LogInToken, signIns[0].Form.Get("csrf-token-login"))
	assert.Equal(t, fakeportal.AnonCookie, signIns[0].Cookie)
}

// MAGTIACC-SC-005, MAGTIACC-SC-006: Each refusal of the sign in names its own error.
func TestTheSignInNamesEachRefusal(t *testing.T) {
	t.Parallel()

	cases := map[string]error{
		"invalid-credentials":      errs.ErrCredentialsRejected,
		"two-factor-code-required": errs.ErrCodeRequired,
		"locked":                   errs.ErrAccountLocked,
		"invalid-token":            errs.ErrRequestFailed,
	}

	for message, want := range cases {
		t.Run(message, func(t *testing.T) {
			t.Parallel()

			client, portal := newClient(t)
			portal.Set(func(p *fakeportal.Portal) { p.LogInMessage = message })

			_, err := client.SignIn(context.Background(), username, password)
			require.ErrorIs(t, err, want)
		})
	}
}

// MAGTIACC-SC-003: The subscribers come from two menus, each with its kind and label,
// and a domain is never selected.
func TestTheSubscribersComeWithTheirKinds(t *testing.T) {
	t.Parallel()

	client, portal := newClient(t)

	subscribers, err := client.Subscribers(context.Background(), validSession())
	require.NoError(t, err)

	assert.ElementsMatch(t, []model.Subscriber{
		{Number: fakeportal.Mobile, Kind: model.SubscriberKindMobile},
		{Number: fakeportal.HomeLine, Label: "House", Kind: model.SubscriberKindHomeLine},
		{Number: fakeportal.Domain, Kind: model.SubscriberKindOther},
	}, subscribers)
	assert.Equal(t, []string{fakeportal.HomeLine}, portal.Selections())
}

// MAGTIACC-SC-003: A subscriber that the two menus do not settle is selected for its kind.
func TestASubscriberBeyondTheTwoMenusIsSelected(t *testing.T) {
	t.Parallel()

	client, portal := newClient(t)
	portal.Set(func(p *fakeportal.Portal) {
		p.Subscribers = append(p.Subscribers, fakeportal.Subscriber{
			Number: "599000002",
			Source: "request/accounts.php",
		})
	})

	subscribers, err := client.Subscribers(context.Background(), validSession())
	require.NoError(t, err)

	require.Len(t, subscribers, 4)
	assert.Equal(t, []string{fakeportal.HomeLine, "599000002"}, portal.Selections())
}

// MAGTIACC-SC-008: A page that shows the sign in form, and the body false of a data call,
// each mean an ended session.
func TestAnEndedSessionIsRejected(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t)
	ended := model.Session{Cookies: map[string]string{fakeportal.CookieName: "OLD"}}

	_, err := client.Subscribers(context.Background(), ended)
	require.ErrorIs(t, err, errs.ErrSessionRejected)

	_, err = client.MobileState(context.Background(), ended, fakeportal.Mobile)
	require.ErrorIs(t, err, errs.ErrSessionRejected)
}

// MAGTIACC-SC-008: A redirect of a data call means an ended session.
func TestARedirectOfADataCallIsRejected(t *testing.T) {
	t.Parallel()

	client, portal := newClient(t)
	portal.Set(func(p *fakeportal.Portal) {
		p.Override = func(w http.ResponseWriter, r *http.Request, _ int) bool {
			if r.URL.Path != "/request/accounts.php" {
				return false
			}

			http.Redirect(w, r, "/ka/mymagti/", http.StatusFound)

			return true
		}
	})

	_, err := client.MobileState(context.Background(), validSession(), fakeportal.Mobile)
	require.ErrorIs(t, err, errs.ErrSessionRejected)
}

// MAGTIACC-SC-021: The state of a mobile subscriber holds the balance, the plan, and the
// allowances.
func TestTheMobileStateHoldsThePlanAndTheAllowances(t *testing.T) {
	t.Parallel()

	client, portal := newClient(t)

	state, err := client.MobileState(context.Background(), validSession(), fakeportal.Mobile)
	require.NoError(t, err)

	assert.Equal(t, "Active", state.Status)
	assert.Equal(t, model.Tetri(3), state.Balance)
	require.NotNil(t, state.Allowances)
	assert.InDelta(t, 2589.29, state.Allowances.DataMB, 1e-9)
	assert.Equal(t, 3376, state.Allowances.CallSeconds)
	assert.Equal(t, 100, state.Allowances.SMS)
	require.NotNil(t, state.Plan)
	assert.Equal(t, model.Tetri(1730), state.Plan.Price)
	assert.Equal(t, "2026-12-13", model.DayOf(*state.Plan.RenewsAt).String())

	packages := portal.CallsTo("/request/cocktail-package.php")
	require.Len(t, packages, 1)
	assert.Equal(t, "77", packages[0].Query.Get("package-id"))
	assert.Equal(t, "1", packages[0].Query.Get("auto-top-up"))
	assert.Equal(t, "3", packages[0].Query.Get("validityId"))
}

// MAGTIACC-SC-021: A mobile subscriber with no active plan answers no plan, and no read
// of the plan.
func TestAMobileSubscriberWithNoPlanAnswersNoPlan(t *testing.T) {
	t.Parallel()

	client, portal := newClient(t)
	portal.Set(func(p *fakeportal.Portal) {
		p.Accounts["activeCocktail"] = map[string]any{"untilDate": ""}
	})

	state, err := client.MobileState(context.Background(), validSession(), fakeportal.Mobile)
	require.NoError(t, err)

	assert.Nil(t, state.Plan)
	assert.Empty(t, portal.CallsTo("/request/cocktail-package.php"))
}

// MAGTIACC-SC-021: The state of a home line holds the plan, the balance, and the speed,
// and no allowances.
func TestTheHomeLineStateHoldsThePlanAndTheSpeed(t *testing.T) {
	t.Parallel()

	client, portal := newClient(t)

	state, err := client.HomeLineState(context.Background(), validSession(), fakeportal.HomeLine)
	require.NoError(t, err)

	assert.Equal(t, model.SubscriberKindHomeLine, state.Kind)
	assert.Equal(t, model.Tetri(2770), state.Balance)
	assert.Equal(t, "70 Mbps", state.Speed)
	assert.Nil(t, state.Allowances)
	require.NotNil(t, state.Plan)
	assert.Equal(t, model.Tetri(4000), state.Plan.Price)
	assert.Equal(t, 20, state.Plan.DaysLeft)
	assert.Equal(
		t,
		fakeportal.HomeLine,
		portal.CallsTo("/request/account-data.php")[0].Query.Get("subscriber-number"),
	)
}

// MAGTIACC-SC-022: A data call that answers 503 is a failed request.
func TestAFailedDataCallIsARequestFailure(t *testing.T) {
	t.Parallel()

	client, portal := newClient(t)
	portal.Set(func(p *fakeportal.Portal) {
		p.Override = func(w http.ResponseWriter, r *http.Request, _ int) bool {
			if r.URL.Path != "/request/accounts.php" {
				return false
			}

			w.WriteHeader(http.StatusServiceUnavailable)

			return true
		}
	})

	_, err := client.MobileState(context.Background(), validSession(), fakeportal.Mobile)
	require.ErrorIs(t, err, errs.ErrRequestFailed)
}

// MAGTIACC-SC-013: The statistics open with the terms and step 1, and each series is a
// step 3 call for the period.
func TestTheStatisticsOpenAndReadAPeriod(t *testing.T) {
	t.Parallel()

	client, portal := newClient(t)
	portal.Set(func(p *fakeportal.Portal) {
		p.History[fakeportal.HistoryKey("MB", "expense")] = []any{
			fakeportal.UsageRow(1795464000000, 812.4, 0),
			fakeportal.TotalRow(812.4),
		}
	})

	ctx := context.Background()

	token, err := client.OpenStatistics(ctx, validSession(), fakeportal.Mobile)
	require.NoError(t, err)

	day := model.Day{Year: 2026, Month: time.November, Day: 24}
	query := service.StatisticsQuery{
		Number: fakeportal.Mobile,
		Token:  token,
		Period: model.DayPeriod(day),
	}

	rows, err := client.Usage(ctx, validSession(), query, dto.SeriesData)
	require.NoError(t, err)
	assert.Len(t, rows, 2)

	money, err := client.Money(ctx, validSession(), query, service.ChargeIncome)
	require.NoError(t, err)
	assert.Empty(t, money)

	assert.Len(t, portal.CallsTo("/request/accept-terms.php"), 1)

	calls := portal.CallsTo("/request/statistics.php")
	require.Len(t, calls, 3)
	assert.Equal(t, "1", calls[0].Form.Get("step"))
	assert.Equal(t, fakeportal.Mobile, calls[0].Form.Get("mobile-number"))
	assert.Equal(t, "3", calls[1].Form.Get("step"))
	assert.Equal(t, "24/11/2026", calls[1].Form.Get("date[from]"))
	assert.Equal(t, "24/11/2026", calls[1].Form.Get("date[to]"))
	assert.Equal(t, "GEL", calls[2].Form.Get("key"))
	assert.Equal(t, "income", calls[2].Form.Get("charge-type"))
}
