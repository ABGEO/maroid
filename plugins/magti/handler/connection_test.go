package handler_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/plugins/magti/internal/fakeportal"
	"github.com/abgeo/maroid/plugins/magti/internal/fakes"
	"github.com/abgeo/maroid/plugins/magti/model"
)

const (
	pathConnection = "/connections/self"
	pathAssignment = "/subscriber-assignments/self"
)

type connectionAnswer struct {
	State             string             `json:"state"`
	CredentialsStored bool               `json:"credentials_stored"`
	Subscribers       []subscriberAnswer `json:"subscribers"`
}

type subscriberAnswer struct {
	SubscriberNumber string `json:"subscriber_number"`
	Label            string `json:"label"`
	Kind             string `json:"kind"`
	Assigned         bool   `json:"assigned"`
}

func readConnection() request {
	return request{method: http.MethodGet, pattern: pathConnection}
}

func assign(number string) request {
	return request{
		method:  http.MethodPut,
		pattern: pathAssignment,
		body:    `{"subscriber_number":"` + number + `"}`,
	}
}

func threeSubscribers(assigned string) []subscriberAnswer {
	return []subscriberAnswer{
		{
			SubscriberNumber: fakeportal.Mobile,
			Kind:             "mobile",
			Assigned:         assigned == fakeportal.Mobile,
		},
		{
			SubscriberNumber: fakeportal.HomeLine,
			Label:            "House",
			Kind:             "home_line",
			Assigned:         assigned == fakeportal.HomeLine,
		},
		{SubscriberNumber: fakeportal.Domain, Kind: "other"},
	}
}

// MAGTIACC-SC-002: With no session and no settings, the page reads "not connected" and
// calls no portal.
func TestTheConnectionReadsNotConnected(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, "", fakes.NewStore("", ""))
	rig.Settings.Values = nil

	answer := decodeAs[connectionAnswer](t, rig.do(t, readConnection()))

	assert.Equal(t, connectionAnswer{State: "not_connected"}, answer)
	assert.Empty(t, rig.Portal.Calls())
}

// MAGTIACC-SC-003: A connected workspace lists every subscriber with its kind and its
// label, marks the assigned one, and selects only the subscriber whose kind the first
// menu does not give.
func TestTheConnectionListsTheSubscribers(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.HomeLine, model.SubscriberKindHomeLine))

	recorder := rig.do(t, readConnection())
	answer := decodeAs[connectionAnswer](t, recorder)

	assert.Equal(t, "connected", answer.State)
	assert.True(t, answer.CredentialsStored)
	assert.ElementsMatch(t, threeSubscribers(fakeportal.HomeLine), answer.Subscribers)
	assert.Equal(t, []string{fakeportal.HomeLine}, rig.Portal.Selections())
	assert.NotEmpty(t, recorder.Header().Get(precondition.ETagHeader))
}

// MAGTIACC-SC-004: A failure of the portal answers 502, and the session stays.
func TestTheConnectionReportsThePortalUnavailable(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, fakeportal.SessionCookie, fakes.NewStore("", ""))
	rig.Portal.Set(func(portal *fakeportal.Portal) {
		portal.Override = func(w http.ResponseWriter, r *http.Request, _ int) bool {
			if r.URL.Path != "/ka/mymagti/" {
				return false
			}

			w.WriteHeader(http.StatusServiceUnavailable)

			return true
		}
	})

	recorder := rig.do(t, readConnection())

	assert.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Equal(t, problemPrefix+"service-unavailable", problemTypeOf(t, recorder))
	assert.Equal(t, fakeportal.SessionCookie, rig.Secrets.Cookie())
}

// MAGTIACC-SC-005: A connect signs in with the form, stores the session, and lists the
// subscribers. A new session keeps the assignment, and a rejection stores no session.
func TestTheConnectSignsInAndStoresTheSession(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, "OLD", fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))

	answer := decodeAs[connectionAnswer](t, rig.do(t, request{
		method:  http.MethodPut,
		pattern: pathConnection,
	}))

	assert.Equal(t, "connected", answer.State)
	assert.ElementsMatch(t, threeSubscribers(fakeportal.Mobile), answer.Subscribers)
	assert.Equal(t, fakeportal.SessionCookie, rig.Secrets.Cookie())

	signIns := rig.Portal.CallsTo("/request/log-in.php")
	require.Len(t, signIns, 1)
	assert.Equal(t, fakes.Username, signIns[0].Form.Get("subscriber"))
	assert.Equal(t, fakes.Password, signIns[0].Form.Get("password"))
	assert.Equal(t, fakeportal.LogInToken, signIns[0].Form.Get("csrf-token-login"))

	rejected := newRouteRig(t, "", fakes.NewStore("", ""))
	rejected.Portal.Set(func(portal *fakeportal.Portal) {
		portal.LogInMessage = refused
	})

	recorder := rejected.do(t, request{method: http.MethodPut, pattern: pathConnection})
	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Equal(t, problemPrefix+"credentials-rejected", problemTypeOf(t, recorder))
	assert.Empty(t, rejected.Secrets.Session())
}

// MAGTIACC-SC-006: A code by phone and a locked account each answer 422 with their own
// type, and an unknown answer is a failure of the portal.
func TestTheConnectNamesEachRefusal(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		status int
		slug   string
	}{
		"two-factor-code-required": {http.StatusUnprocessableEntity, "code-required"},
		"locked":                   {http.StatusUnprocessableEntity, "account-locked"},
		"invalid-token":            {http.StatusBadGateway, "service-unavailable"},
	}

	for message, want := range cases {
		t.Run(message, func(t *testing.T) {
			t.Parallel()

			rig := newRouteRig(t, "", fakes.NewStore("", ""))
			rig.Portal.Set(func(portal *fakeportal.Portal) { portal.LogInMessage = message })

			recorder := rig.do(t, request{method: http.MethodPut, pattern: pathConnection})
			assert.Equal(t, want.status, recorder.Code)
			assert.Equal(t, problemPrefix+want.slug, problemTypeOf(t, recorder))
			assert.Empty(t, rig.Secrets.Session())
		})
	}
}

// MAGTIACC-SC-007: An end of the connection removes the session and the assignment, and
// a repeat answers 204 too.
func TestTheDisconnectRemovesTheSessionAndTheAssignment(t *testing.T) {
	t.Parallel()

	store := fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile)
	store.Days["2026-11-24"] = model.TrafficDay{}
	rig := newRouteRig(t, fakeportal.SessionCookie, store)

	for range 2 {
		recorder := rig.do(t, request{method: http.MethodDelete, pattern: pathConnection})
		assert.Equal(t, http.StatusNoContent, recorder.Code)
	}

	assert.Empty(t, rig.Secrets.Session())

	_, err := store.Assignment(t.Context())
	require.Error(t, err)
	assert.Len(t, store.Days, 1)
	assert.NotNil(t, rig.Settings.Values)
}

// MAGTIACC-SC-008: An ended session signs in once, stores the new session, and the read
// runs again.
func TestAnEndedSessionSignsInOnceForTheState(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))
	rig.Portal.Set(func(portal *fakeportal.Portal) {
		portal.Override = func(w http.ResponseWriter, r *http.Request, count int) bool {
			if r.URL.Path != pathAccounts || count > 1 {
				return false
			}

			_, _ = w.Write([]byte("false"))

			return true
		}
	})

	recorder := rig.do(t, request{method: http.MethodGet, pattern: "/subscriber-states/self"})
	assert.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	assert.Len(t, rig.Portal.CallsTo("/request/log-in.php"), 1)
	assert.Equal(t, fakeportal.SessionCookie, rig.Secrets.Cookie())

	repeated := newRouteRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))
	repeated.Portal.Set(func(portal *fakeportal.Portal) {
		portal.Override = func(w http.ResponseWriter, r *http.Request, _ int) bool {
			if r.URL.Path != pathAccounts {
				return false
			}

			http.Redirect(w, r, "/ka/mymagti/", http.StatusFound)

			return true
		}
	})

	recorder = repeated.do(t, request{method: http.MethodGet, pattern: "/subscriber-states/self"})
	assert.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Equal(t, problemPrefix+"service-unavailable", problemTypeOf(t, recorder))
}

// MAGTIACC-SC-009: A session that the sign in cannot renew reads as not connected, the
// session goes, and the assignment stays.
func TestARefusedSignInReadsNotConnected(t *testing.T) {
	t.Parallel()

	store := fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile)
	rig := newRouteRig(t, "ENDED", store)
	rig.Portal.Set(func(portal *fakeportal.Portal) {
		portal.LogInMessage = refused
	})

	answer := decodeAs[connectionAnswer](t, rig.do(t, readConnection()))

	assert.Equal(t, "not_connected", answer.State)
	assert.Empty(t, rig.Secrets.Session())

	_, err := store.Assignment(t.Context())
	require.NoError(t, err)
}

// MAGTIACC-SC-010: Each assignment answers the new number and its validator, and a write
// with a stale validator answers 412.
func TestTheAssignmentAnswersItsValidator(t *testing.T) {
	t.Parallel()

	store := fakes.NewStore("", "")
	rig := newRouteRig(t, fakeportal.SessionCookie, store)

	first := rig.do(t, assign(fakeportal.Mobile))
	assert.ElementsMatch(t, threeSubscribers(fakeportal.Mobile),
		decodeAs[connectionAnswer](t, first).Subscribers)

	validator := first.Header().Get(precondition.ETagHeader)
	require.NotEmpty(t, validator)

	second := rig.do(t, assign(fakeportal.HomeLine))
	assert.ElementsMatch(t, threeSubscribers(fakeportal.HomeLine),
		decodeAs[connectionAnswer](t, second).Subscribers)
	assert.NotEqual(t, validator, second.Header().Get(precondition.ETagHeader))

	stale := assign(fakeportal.Mobile)
	stale.ifMatch = validator
	recorder := rig.do(t, stale)
	assert.Equal(t, http.StatusPreconditionFailed, recorder.Code)

	held, err := store.Assignment(t.Context())
	require.NoError(t, err)
	assert.Equal(t, model.SubscriberKindHomeLine, held.Kind)
}

// MAGTIACC-SC-011: A number that the account does not hold, and a subscriber that is not
// a phone line, are refused, and the assignment stays.
func TestTheAssignmentRefusesWhatItCannotFollow(t *testing.T) {
	t.Parallel()

	store := fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile)
	rig := newRouteRig(t, fakeportal.SessionCookie, store)

	for number, slug := range map[string]string{
		"599999999":       "subscriber-unknown",
		fakeportal.Domain: "subscriber-unsupported",
	} {
		recorder := rig.do(t, assign(number))
		assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
		assert.Equal(t, problemPrefix+slug, problemTypeOf(t, recorder))
	}

	held, err := store.Assignment(t.Context())
	require.NoError(t, err)
	assert.Equal(t, fakeportal.Mobile, held.SubscriberNumber)

	tooLarge := assign(strings.Repeat("9", 5000))
	assert.Equal(t, http.StatusRequestEntityTooLarge, rig.do(t, tooLarge).Code)
}

// MAGTIACC-SC-012: An assignment collects nothing.
func TestTheAssignmentCollectsNothing(t *testing.T) {
	t.Parallel()

	store := fakes.NewStore("", "")
	rig := newRouteRig(t, fakeportal.SessionCookie, store)

	assert.Equal(t, http.StatusOK, rig.do(t, assign(fakeportal.Mobile)).Code)
	assert.Empty(t, rig.Portal.CallsTo("/request/statistics.php"))
	assert.Zero(t, store.Writes)
}

// MAGTIACC-SC-008: A cookie that a page sets joins the stored session.
func TestACookieThatAPageSetsIsStored(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, fakeportal.SessionCookie, fakes.NewStore("", ""))
	rig.Portal.Set(func(portal *fakeportal.Portal) { portal.Rotate = "K2" })

	assert.Equal(t, http.StatusOK, rig.do(t, readConnection()).Code)
	assert.Equal(t, "K2", rig.Secrets.Cookie())
}

// MAGTIACC-SC-009: A route that reads the portal answers a refused sign in as not
// connected, and the session goes.
func TestARefusedSignInAnswersNotConnected(t *testing.T) {
	t.Parallel()

	for _, call := range []request{readState(), assign(fakeportal.Mobile)} {
		t.Run(call.method+" "+call.pattern, func(t *testing.T) {
			t.Parallel()

			rig := newRouteRig(t, "ENDED",
				fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))
			rig.Portal.Set(func(portal *fakeportal.Portal) {
				portal.LogInMessage = refused
			})

			recorder := rig.do(t, call)
			assert.Equal(t, http.StatusConflict, recorder.Code)
			assert.Equal(t, problemPrefix+"not-connected", problemTypeOf(t, recorder))
			assert.Empty(t, rig.Secrets.Session())
		})
	}
}

// MAGTIACC-SC-009: With no settings, a session that the portal rejects reads as not
// connected, and the session goes.
func TestARejectedSessionWithNoSettingsGoes(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, "ENDED", fakes.NewStore("", ""))
	rig.Settings.Values = nil

	answer := decodeAs[connectionAnswer](t, rig.do(t, readConnection()))

	assert.Equal(t, connectionAnswer{State: "not_connected"}, answer)
	assert.Empty(t, rig.Secrets.Session())
}

// MAGTIACC-SC-031: An end of the connection waits for a collection that signs in again,
// so no session stays after both.
func TestTheEndOfTheConnectionWaitsForASignIn(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, "ENDED", fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))
	signingIn := make(chan struct{})

	rig.Portal.Set(func(portal *fakeportal.Portal) {
		portal.Override = func(_ http.ResponseWriter, r *http.Request, count int) bool {
			if r.URL.Path == "/request/log-in.php" && count == 1 {
				close(signingIn)
				time.Sleep(300 * time.Millisecond)
			}

			return false
		}
	})

	collected := make(chan error, 1)

	go func() {
		_, err := rig.Collector().Collect(
			fakes.Acting(t.Context()),
			model.DayPeriod(model.Day{Year: 2026, Month: time.November, Day: 24}),
		)
		collected <- err
	}()

	<-signingIn

	assert.Equal(t, http.StatusNoContent,
		rig.do(t, request{method: http.MethodDelete, pattern: pathConnection}).Code)
	require.NoError(t, <-collected)
	assert.Empty(t, rig.Secrets.Session())
}
