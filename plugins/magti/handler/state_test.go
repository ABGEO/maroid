package handler_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/magti/internal/fakeportal"
	"github.com/abgeo/maroid/plugins/magti/internal/fakes"
	"github.com/abgeo/maroid/plugins/magti/model"
	"github.com/abgeo/maroid/plugins/magti/widget"
)

const pathState = "/subscriber-states/self"

type moneyAnswer struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type stateAnswer struct {
	SubscriberNumber string       `json:"subscriber_number"`
	Kind             string       `json:"kind"`
	Status           string       `json:"status"`
	Balance          moneyAnswer  `json:"balance"`
	Speed            string       `json:"speed"`
	Plan             *planAnswer  `json:"plan"`
	Allowances       *allowAnswer `json:"allowances"`
}

type planAnswer struct {
	Name     string      `json:"name"`
	Price    moneyAnswer `json:"price"`
	RenewsOn string      `json:"renews_on"`
	DaysLeft int         `json:"days_left"`
}

type allowAnswer struct {
	DataMB      float64 `json:"data_mb"`
	CallSeconds int     `json:"call_seconds"`
	SMS         int     `json:"sms"`
}

func readState() request {
	return request{method: http.MethodGet, pattern: pathState}
}

// MAGTIACC-SC-021: The state of a mobile subscriber holds the balance, the plan with its
// renewal, and the allowances, read from the portal at each read.
func TestTheStateOfAMobileSubscriber(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))

	for range 2 {
		answer := decodeAs[stateAnswer](t, rig.do(t, readState()))

		assert.Equal(t, moneyAnswer{Amount: 0.03, Currency: lari}, answer.Balance)
		require.NotNil(t, answer.Plan)
		assert.Equal(t, "2026-12-13", answer.Plan.RenewsOn)
		assert.Equal(t, 18, answer.Plan.DaysLeft)
		assert.Equal(t, moneyAnswer{Amount: 17.30, Currency: lari}, answer.Plan.Price)
		assert.Equal(t, &allowAnswer{DataMB: 2589.29, CallSeconds: 3376, SMS: 100},
			answer.Allowances)
	}

	assert.Len(t, rig.Portal.CallsTo(pathAccounts), 2)
	assert.Equal(t, []string{fakeportal.Mobile, fakeportal.Mobile}, rig.Portal.Selections())
}

// MAGTIACC-SC-021: The state of a home line holds the plan and the speed, and no
// allowances. A mobile subscriber with no plan answers no plan.
func TestTheStateOfAHomeLineAndOfNoPlan(t *testing.T) {
	t.Parallel()

	home := newRouteRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.HomeLine, model.SubscriberKindHomeLine))

	answer := decodeAs[stateAnswer](t, home.do(t, readState()))
	assert.Equal(t, "home_line", answer.Kind)
	assert.Equal(t, "70 Mbps", answer.Speed)
	assert.Nil(t, answer.Allowances)
	require.NotNil(t, answer.Plan)
	assert.Equal(t, 20, answer.Plan.DaysLeft)
	assert.InDelta(t, 40.0, answer.Plan.Price.Amount, 1e-9)

	noPlan := newRouteRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))
	noPlan.Portal.Set(func(portal *fakeportal.Portal) {
		portal.Accounts["activeCocktail"] = map[string]any{}
	})

	answer = decodeAs[stateAnswer](t, noPlan.do(t, readState()))
	assert.Nil(t, answer.Plan)
	assert.Empty(t, noPlan.Portal.CallsTo("/request/cocktail-package.php"))
}

// MAGTIACC-SC-022: No assignment, no session, and a failure of the portal each answer
// their problem.
func TestTheStateNamesWhatIsMissing(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		cookie string
		store  *fakes.Store
		fail   bool
		status int
		slug   string
	}{
		"no assignment": {
			cookie: fakeportal.SessionCookie,
			store:  fakes.NewStore("", ""),
			status: http.StatusConflict,
			slug:   "not-assigned",
		},
		"no session": {
			store:  fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile),
			status: http.StatusConflict,
			slug:   "not-connected",
		},
		"a failure of the portal": {
			cookie: fakeportal.SessionCookie,
			store:  fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile),
			fail:   true,
			status: http.StatusBadGateway,
			slug:   "service-unavailable",
		},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rig := newRouteRig(t, setup.cookie, setup.store)
			if setup.fail {
				rig.Portal.Set(func(portal *fakeportal.Portal) {
					portal.Override = func(w http.ResponseWriter, r *http.Request, _ int) bool {
						if r.URL.Path != pathAccounts {
							return false
						}

						w.WriteHeader(http.StatusServiceUnavailable)

						return true
					}
				})
			}

			recorder := rig.do(t, readState())
			assert.Equal(t, setup.status, recorder.Code)
			assert.Equal(t, problemPrefix+setup.slug, problemTypeOf(t, recorder))
		})
	}
}

// MAGTIACC-SC-028: A portal that holds the read answers 502 no later than half a second
// after state_timeout.
func TestTheStateEndsAtTheStateTimeout(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))
	rig.Portal.Set(func(portal *fakeportal.Portal) {
		portal.Override = func(_ http.ResponseWriter, r *http.Request, _ int) bool {
			if r.URL.Path == pathAccounts {
				select {
				case <-r.Context().Done():
				case <-time.After(10 * time.Second):
				}
			}

			return false
		}
	})

	started := time.Now()
	recorder := rig.do(t, readState())
	elapsed := time.Since(started)

	assert.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Equal(t, problemPrefix+"service-unavailable", problemTypeOf(t, recorder))
	assert.Contains(t, recorder.Body.String(), "no answer in time")
	assert.Less(t, elapsed, 4500*time.Millisecond)
}

// MAGTIACC-SC-030: No answer holds the password, the cookie, a token, or the personal
// data of the holder.
func TestNoAnswerHoldsASecretOrThePersonalData(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))

	calls := []request{
		readConnection(),
		readState(),
		assign(fakeportal.Mobile),
		{method: http.MethodPut, pattern: pathConnection},
		{method: http.MethodGet, pattern: pathTraffic, query: "from=2026-11-01&to=2026-11-24"},
		{method: http.MethodGet, pattern: pathMoneyEntries},
		{method: http.MethodGet, pattern: pathMonthTotals, query: "from=2026-01"},
	}

	secrets := []string{
		fakes.Password,
		fakeportal.SessionCookie + `"`,
		fakeportal.LogInToken,
		fakeportal.StatisticsTok,
		fakeportal.ContractToken,
		fakeportal.HolderName,
		fakeportal.PersonalNumber,
	}

	for _, call := range calls {
		recorder := rig.do(t, call)
		body := recorder.Body.String()

		for _, secret := range secrets {
			assert.NotContains(t, body, secret,
				"%s %s answers %q", call.method, call.pattern, secret)
		}
	}

	answer, err := widgetOf(rig).Load(fakes.Acting(t.Context()))
	require.NoError(t, err)
	assert.NotContains(t, answer.Stat.Caption, fakeportal.HolderName)
}

// MAGTIACC-SC-031: A read of the connection and a read of the state at the same time
// never select over each other.
func TestTheConnectionAndTheStateNeverSelectOverEachOther(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))
	rig.Portal.Set(func(portal *fakeportal.Portal) {
		portal.Override = func(_ http.ResponseWriter, r *http.Request, _ int) bool {
			if r.URL.Path == "/mymagti" {
				time.Sleep(100 * time.Millisecond)
			}

			return false
		}
	})

	done := make(chan int, 2)

	for _, call := range []request{readConnection(), readState()} {
		go func() { done <- rig.do(t, call).Code }()
	}

	assert.Equal(t, http.StatusOK, <-done)
	assert.Equal(t, http.StatusOK, <-done)

	calls := rig.Portal.Calls()
	for index, call := range calls {
		if call.Path != pathAccounts {
			continue
		}

		selection := index - 1
		for selection >= 0 && calls[selection].Path == "/ka/mymagti/" {
			selection--
		}

		require.GreaterOrEqual(t, selection, 0)
		assert.Equal(t, "/mymagti", calls[selection].Path)
		assert.Equal(t, fakeportal.Mobile, calls[selection].Query.Get("subscriber-number"))
	}
}

func widgetOf(rig *routeRig) *pluginapi.FuncWidget {
	return widget.NewSubscriberState(rig.states)
}

// MAGTIACC-SC-021: An answer of accounts.php that holds no balance answers 502, never a
// balance of zero.
func TestAStateWithNoBalanceFails(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))
	rig.Portal.Set(func(portal *fakeportal.Portal) {
		portal.Accounts["balances"] = []any{}
	})

	recorder := rig.do(t, readState())
	assert.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Equal(t, problemPrefix+"service-unavailable", problemTypeOf(t, recorder))
}

// MAGTIACC-SC-023: A session that the sign in cannot renew leaves the widget not set up.
func TestTheWidgetOfARefusedSignInIsNotSetUp(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, "ENDED", fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))
	rig.Portal.Set(func(portal *fakeportal.Portal) {
		portal.LogInMessage = refused
	})

	answer, err := widgetOf(rig).Load(fakes.Acting(t.Context()))
	require.NoError(t, err)
	assert.Equal(t, pluginapi.NotSetUpAnswer(
		"Connect the Magti account and assign a subscriber.",
		"/account",
	), answer)
}
