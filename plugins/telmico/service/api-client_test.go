package service_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/telmico/config"
	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/model"
	"github.com/abgeo/maroid/plugins/telmico/service"
)

const (
	username = "01001010101"
	password = "secret-password"

	cookieName      = "_tpcabinet_app_u_"
	anonymousCookie = "ANONYMOUS"
	cookieK         = "COOKIE-K"
	csrfOfForm      = "csrf-of-the-form"
	csrfT           = "csrf-T"
	customerC1      = "8370764"

	pathStart    = "/"
	pathLogin    = "/login"
	pathPrivate  = "/private"
	pathAccounts = "/private/accounts"

	formLimit = 1 << 20

	septemberForm = "2026-9"
)

// call is one request that the fake portal received.
type call struct {
	method string
	path   string
	query  string
	cookie string
	csrf   string
	form   url.Values
	accept string
}

// fakePortal plays the portal of the electricity service. Each handler answers one
// path, and the portal records each call.
type fakePortal struct {
	mu       sync.Mutex
	handlers map[string]http.HandlerFunc
	calls    []call
}

func (p *fakePortal) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	recorded := call{
		method: r.Method,
		path:   r.URL.Path,
		query:  r.URL.RawQuery,
		csrf:   r.Header.Get("X-Csrf-Token"),
		accept: r.Header.Get("Accept"),
	}

	if cookie, err := r.Cookie(cookieName); err == nil {
		recorded.cookie = cookie.Value
	}

	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		r.Body = http.MaxBytesReader(w, r.Body, formLimit)
		_ = r.ParseMultipartForm(formLimit) //nolint:gosec // G120: MaxBytesReader bounds the body.
		recorded.form = r.MultipartForm.Value
	} else if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, formLimit)
		_ = r.ParseForm()
		recorded.form = r.PostForm
	}

	p.mu.Lock()
	p.calls = append(p.calls, recorded)
	handle, found := p.handlers[r.URL.Path]
	p.mu.Unlock()

	if !found {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	handle(w, r)
}

func (p *fakePortal) recorded() []call {
	p.mu.Lock()
	defer p.mu.Unlock()

	return append([]call(nil), p.calls...)
}

func newClient(
	t *testing.T,
	handlers map[string]http.HandlerFunc,
) (*service.APIClient, *fakePortal) {
	t.Helper()

	fake := &fakePortal{handlers: handlers}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	return service.NewAPIClient(&config.Config{
		BaseURL:        server.URL,
		RequestTimeout: 5 * time.Second,
	}), fake
}

func htmlPage(csrf string) string {
	return `<!DOCTYPE html><html><head>
<meta name="_csrf" content="` + csrf + `"/>
<meta name="_csrf_header" content="X-CSRF-TOKEN"/>
</head><body><form action="login" method="POST"></form></body></html>`
}

func setCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func html(cookie, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if cookie != "" {
			setCookie(w, cookie)
		}

		w.Header().Set("Content-Type", "text/html;charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, body)
	}
}

func redirect(cookie, location string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if cookie != "" {
			setCookie(w, cookie)
		}

		w.Header().Set("Location", location)
		w.WriteHeader(http.StatusFound)
	}
}

func status(code int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
	}
}

func jsonBody(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, body)
	}
}

// signInHandlers answer a successful sign in. The redirects name http:// with a host
// that does not resolve, as the portal does, so a followed redirect fails the test.
func signInHandlers() map[string]http.HandlerFunc {
	return map[string]http.HandlerFunc{
		pathStart:   html(anonymousCookie, htmlPage(csrfOfForm)),
		pathLogin:   redirect(cookieK, "http://portal.invalid/private"),
		pathPrivate: html("", htmlPage(csrfT)),
	}
}

func session() model.Session {
	return model.Session{Cookie: cookieK, CSRFToken: csrfT}
}

// ELECACC-SC-005: The sign in reads the CSRF token of the form, posts the credentials
// with it, follows no redirect, and reads the CSRF token of the new session.
func TestTheSignInPostsTheFormAndReadsTheTokenOfTheNewSession(t *testing.T) {
	t.Parallel()

	client, fake := newClient(t, signInHandlers())

	signed, err := client.SignIn(t.Context(), username, password)

	require.NoError(t, err)
	assert.Equal(t, model.Session{Cookie: cookieK, CSRFToken: csrfT}, signed)

	calls := fake.recorded()
	require.Len(t, calls, 3)

	assert.Equal(t, http.MethodGet, calls[0].method)
	assert.Equal(t, pathStart, calls[0].path)

	assert.Equal(t, http.MethodPost, calls[1].method)
	assert.Equal(t, pathLogin, calls[1].path)
	assert.Equal(t, anonymousCookie, calls[1].cookie)
	assert.Equal(t, username, calls[1].form.Get("username"))
	assert.Equal(t, password, calls[1].form.Get("password"))
	assert.Equal(t, csrfOfForm, calls[1].form.Get("_csrf"))

	assert.Equal(t, http.MethodGet, calls[2].method)
	assert.Equal(t, pathPrivate, calls[2].path)
	assert.Equal(t, cookieK, calls[2].cookie)
}

// ELECACC-SC-006: A redirect to the error of bad credentials is a rejection. A redirect
// to the start page or a page with no CSRF token is a failure of the portal.
func TestTheSignInClassifiesTheAnswerOfThePortal(t *testing.T) {
	t.Parallel()

	for name, setup := range signInFailures() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client, _ := newClient(t, setup.handlers)

			_, err := client.SignIn(t.Context(), username, password)

			require.ErrorIs(t, err, setup.want)
		})
	}
}

type signInFailure struct {
	handlers map[string]http.HandlerFunc
	want     error
}

func signInFailures() map[string]signInFailure {
	return map[string]signInFailure{
		"bad credentials": {
			handlers: map[string]http.HandlerFunc{
				pathStart: html(anonymousCookie, htmlPage(csrfOfForm)),
				pathLogin: redirect(
					"",
					"http://portal.invalid/?tabId=password&error=error.bad_credentials",
				),
			},
			want: errs.ErrCredentialsRejected,
		},
		"redirect to the start page": {
			handlers: map[string]http.HandlerFunc{
				pathStart: html(anonymousCookie, htmlPage(csrfOfForm)),
				pathLogin: redirect("", "http://portal.invalid/"),
			},
			want: errs.ErrRequestFailed,
		},
		"no CSRF token on the first page": {
			handlers: map[string]http.HandlerFunc{
				pathStart: html(anonymousCookie, "<html><head></head></html>"),
			},
			want: errs.ErrRequestFailed,
		},
		"no cookie on the first page": {
			handlers: map[string]http.HandlerFunc{
				pathStart: html("", htmlPage(csrfOfForm)),
			},
			want: errs.ErrRequestFailed,
		},
		"no CSRF token in the new session": {
			handlers: map[string]http.HandlerFunc{
				pathStart:   html(anonymousCookie, htmlPage(csrfOfForm)),
				pathLogin:   redirect(cookieK, "http://portal.invalid/private"),
				pathPrivate: html("", "<html></html>"),
			},
			want: errs.ErrRequestFailed,
		},
		"the portal fails": {
			handlers: map[string]http.HandlerFunc{
				pathStart: status(http.StatusServiceUnavailable),
			},
			want: errs.ErrRequestFailed,
		},
	}
}

// ELECACC-SC-008: A call with an ended session is a rejected session, which the caller
// answers with a new sign in.
func TestARejectedSessionIsNamed(t *testing.T) {
	t.Parallel()

	cases := map[string]http.HandlerFunc{
		"401":      status(http.StatusUnauthorized),
		"403":      status(http.StatusForbidden),
		"redirect": redirect("", "http://portal.invalid/"),
	}

	for name, handle := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client, _ := newClient(t, map[string]http.HandlerFunc{pathAccounts: handle})

			_, err := client.Accounts(t.Context(), session())

			require.ErrorIs(t, err, errs.ErrSessionRejected)
		})
	}
}

// ELECACC-SC-004: A failure of the portal is a failed request, never an empty list.
func TestAFailureOfThePortalIsAFailedRequest(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, map[string]http.HandlerFunc{
		pathAccounts: status(http.StatusServiceUnavailable),
	})

	_, err := client.Accounts(t.Context(), session())

	require.ErrorIs(t, err, errs.ErrRequestFailed)
	require.NotErrorIs(t, err, errs.ErrSessionRejected)
}

// ELECACC-SC-003: The accounts and the detail carry the session, and the detail answers
// the address alone.
func TestTheAccountsCarryTheSession(t *testing.T) {
	t.Parallel()

	client, fake := newClient(t, map[string]http.HandlerFunc{
		pathAccounts: jsonBody(`[{"id":3457,"account":"8370764","name":"House","address":null}]`),
		"/private/account/8370764": jsonBody(`{"account":"8370764",` +
			`"address":"Nutsubidze 4","owner":"Holder Name","idNumber":"62909012413"}`),
	})

	accounts, err := client.Accounts(t.Context(), session())
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	assert.Equal(t, customerC1, accounts[0].Number)
	assert.Equal(t, "House", accounts[0].Name)

	detail, err := client.AccountDetail(t.Context(), session(), customerC1)
	require.NoError(t, err)
	assert.Equal(t, customerC1, detail.Number)
	assert.Equal(t, "Nutsubidze 4", detail.Address)

	for _, recorded := range fake.recorded() {
		assert.Equal(t, http.MethodPost, recorded.method)
		assert.Equal(t, cookieK, recorded.cookie)
		assert.Equal(t, csrfT, recorded.csrf)
	}
}

// ELECACC-SC-013: The payments and the receipts of one month go out with the month in
// the form of the portal, and every page is read.
func TestTheMonthIsReadPageByPage(t *testing.T) {
	t.Parallel()

	paged := func(first, second string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, formLimit)
			//nolint:gosec // G120: MaxBytesReader bounds the body.
			_ = r.ParseMultipartForm(formLimit)

			body := first
			if r.MultipartForm.Value["page"][0] == "1" {
				body = second
			}

			jsonBody(body)(w, r)
		}
	}

	client, fake := newClient(t, map[string]http.HandlerFunc{
		"/private/account/8370764/allpaymentsapi": paged(
			`{"items":[{"operDate":"2026-09-28","transType":"charge"}],"totalPages":2}`,
			`{"items":[{"operDate":"2026-09-01","transType":"payment"}],"totalPages":2}`,
		),
		"/private/account/8370764/receiptsapi": paged(
			`{"items":[{"billNumber":"8370764-2026-9"}],"totalPages":1}`,
			`{"items":[],"totalPages":1}`,
		),
	})

	september := model.Month{Year: 2026, Month: time.September}

	payments, err := client.Payments(t.Context(), session(), customerC1, september)
	require.NoError(t, err)
	require.Len(t, payments, 2)
	assert.Equal(t, "2026-09-01", payments[1].OperationDate)

	receipts, err := client.Receipts(t.Context(), session(), customerC1, september)
	require.NoError(t, err)
	require.Len(t, receipts, 1)

	calls := fake.recorded()
	require.Len(t, calls, 3)

	for _, recorded := range calls {
		assert.Equal(t, []string{"100"}, recorded.form["size"])
		assert.Equal(t, csrfT, recorded.csrf)
	}

	for _, payments := range calls[:2] {
		assert.Equal(t, []string{septemberForm}, payments.form["from_month"])
		assert.Equal(t, []string{septemberForm}, payments.form["to_month"])
		assert.Equal(t, []string{"operDate,desc"}, payments.form["sort"])
	}

	assert.Equal(t, []string{"0"}, calls[0].form["page"])
	assert.Equal(t, []string{"1"}, calls[1].form["page"])

	assert.Equal(t, []string{septemberForm}, calls[2].form["from_month"])
	assert.Equal(t, []string{"2026-10"}, calls[2].form["to_month"])
}

// ELECACC-SC-019: The receipts endpoint takes to_month as an exclusive bound, so the
// receipts of December end at January of the next year.
func TestTheReceiptsOfAMonthEndAtTheNextMonth(t *testing.T) {
	t.Parallel()

	client, fake := newClient(t, map[string]http.HandlerFunc{
		"/private/account/8370764/receiptsapi": jsonBody(`{"items":[],"totalPages":1}`),
	})

	_, err := client.Receipts(
		t.Context(),
		session(),
		customerC1,
		model.Month{Year: 2025, Month: time.December},
	)
	require.NoError(t, err)

	calls := fake.recorded()
	require.Len(t, calls, 1)
	assert.Equal(t, []string{"2025-12"}, calls[0].form["from_month"])
	assert.Equal(t, []string{"2026-1"}, calls[0].form["to_month"])
}

// ELECACC-SC-028: The document of a bill is the PDF that the portal answers. A body
// that is no PDF is a failed request.
func TestTheDocumentOfABillIsAPDF(t *testing.T) {
	t.Parallel()

	billID := uuid.MustParse("f9ca32a7-ec95-4145-97af-cbef330af8dc")
	path := "/private/receipt/" + billID.String()

	pdf := func(body string) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/pdf")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, body)
		}
	}

	client, fake := newClient(t, map[string]http.HandlerFunc{path: pdf("%PDF-1.3 content")})

	document, err := client.ReceiptDocument(t.Context(), session(), billID)
	require.NoError(t, err)
	assert.Equal(t, "%PDF-1.3 content", string(document))
	assert.Equal(t, cookieK, fake.recorded()[0].cookie)

	client, _ = newClient(t, map[string]http.HandlerFunc{path: pdf("<html>login</html>")})

	_, err = client.ReceiptDocument(t.Context(), session(), billID)
	require.ErrorIs(t, err, errs.ErrRequestFailed)

	client, _ = newClient(t, map[string]http.HandlerFunc{path: status(http.StatusUnauthorized)})

	_, err = client.ReceiptDocument(t.Context(), session(), billID)
	require.ErrorIs(t, err, errs.ErrSessionRejected)
}

// ELECACC-SC-032: One client serves every workspace, so it keeps no cookie of one
// session for the call of another.
func TestTheClientKeepsNoCookieBetweenCalls(t *testing.T) {
	t.Parallel()

	handlers := signInHandlers()
	handlers[pathAccounts] = jsonBody(`[]`)
	client, fake := newClient(t, handlers)

	_, err := client.SignIn(t.Context(), username, password)
	require.NoError(t, err)

	_, err = client.Accounts(context.Background(), model.Session{Cookie: "OTHER", CSRFToken: csrfT})
	require.NoError(t, err)

	calls := fake.recorded()
	assert.Equal(t, "OTHER", calls[len(calls)-1].cookie)
}

// ELECACC-SC-033: A failed answer that is a page of the portal stays out of the error,
// because the page carries a CSRF token. A failed JSON answer keeps its body.
func TestAFailureNamesNoTokenOfAPage(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, map[string]http.HandlerFunc{
		pathStart: html(anonymousCookie, htmlPage(csrfOfForm)),
		pathLogin: html("", htmlPage("csrf-of-a-page-that-answers-200")),
	})

	_, err := client.SignIn(t.Context(), username, password)

	require.ErrorIs(t, err, errs.ErrRequestFailed)
	assert.NotContains(t, err.Error(), "csrf-of-a-page-that-answers-200")
	assert.Contains(t, err.Error(), "[200]")

	client, _ = newClient(t, map[string]http.HandlerFunc{
		pathAccounts: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"error":"the database is down"}`)
		},
	})

	_, err = client.Accounts(t.Context(), session())

	require.ErrorIs(t, err, errs.ErrRequestFailed)
	assert.Contains(t, err.Error(), "the database is down")
}

// ELECACC-SC-004: A list that names more pages than the client reads is a failed
// request, so a broken answer cannot send requests without end.
func TestARunawayListFails(t *testing.T) {
	t.Parallel()

	client, fake := newClient(t, map[string]http.HandlerFunc{
		"/private/account/8370764/allpaymentsapi": jsonBody(`{"items":[],"totalPages":1000000}`),
	})

	_, err := client.Payments(
		t.Context(),
		session(),
		customerC1,
		model.Month{Year: 2026, Month: time.September},
	)

	require.ErrorIs(t, err, errs.ErrRequestFailed)
	assert.LessOrEqual(t, len(fake.recorded()), 50)
}
