// Package fakeportal plays the portal of Magti for the tests of the plugin. No production
// code imports it.
//
//nolint:mnd,goconst,gosec,errchkjson,prealloc // a fixture of the tests, never served.
package fakeportal

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

// The values that the fake portal answers unless a test sets others.
const (
	CookieName     = "PHPSESSID"
	SessionCookie  = "K"
	AnonCookie     = "ANON"
	LogInToken     = "LOGIN-T"
	StatisticsTok  = "STAT-T"
	Mobile         = "599000001"
	HomeLine       = "322000001"
	Domain         = "d_example.ge"
	HolderName     = "Holder Name"
	PersonalNumber = "01001010101"
	ContractToken  = "CONTRACT-TOKEN"

	formLimit = 1 << 20
)

// Subscriber is one subscriber of the fake account.
type Subscriber struct {
	Number string
	Label  string
	// Source is the data-source of the balances on its page.
	Source string
}

// Call is one request that the fake portal received.
type Call struct {
	Method string
	Path   string
	Query  url.Values
	Form   url.Values
	Cookie string
}

// Portal is the state of the fake portal. A test changes a field before the calls.
type Portal struct {
	mu sync.Mutex

	Subscribers []Subscriber
	Selected    string
	// LogInMessage is the message of the answer of the sign in.
	LogInMessage string
	// Valid holds the session cookies that the portal accepts.
	Valid map[string]bool
	// Rotate, when set, is a new session cookie that the page of the menu sets, and that
	// the portal accepts from then on.
	Rotate string
	// Override answers a path in place of the default handler. It answers false to let
	// the default handler answer.
	Override func(w http.ResponseWriter, r *http.Request, count int) bool
	// History answers the history of a series of the statistics, by key and charge type.
	History map[string]any
	// Accounts, Cocktail, IspAccounts, and AccountData are the JSON answers of the data
	// calls.
	Accounts    map[string]any
	Cocktail    map[string]any
	IspAccounts map[string]any
	AccountData map[string]any

	calls  []Call
	counts map[string]int
	server *httptest.Server
}

// New starts a fake portal that holds a mobile subscriber, a home line, and a domain, with
// the mobile subscriber selected and the session K valid.
func New(t *testing.T) *Portal {
	t.Helper()

	portal := &Portal{
		Subscribers: []Subscriber{
			{Number: Mobile, Source: "request/accounts.php"},
			{Number: HomeLine, Label: "House", Source: "request/accounts-isp.php"},
			{Number: Domain, Source: "request/accounts-domain.php"},
		},
		Selected:     Mobile,
		LogInMessage: "success",
		Valid:        map[string]bool{SessionCookie: true},
		History:      map[string]any{},
		Accounts:     MobileAccounts(),
		Cocktail: map[string]any{
			"name":  "Cocktail 17",
			"price": "17.30",
		},
		IspAccounts: map[string]any{
			"package_sum": 40,
			"balances": []any{
				map[string]any{"groupKey": "ISP_BALANCE", "balance": "27.70"},
				map[string]any{"groupKey": "ISP_INTERNET", "balance": "70 Mbps"},
			},
		},
		AccountData: map[string]any{
			"status":        "1",
			"statusName":    "Active",
			"daysRemaining": 20,
			"totalDays":     30,
			"name":          HolderName,
		},
		counts: map[string]int{},
	}

	portal.server = httptest.NewServer(portal)
	t.Cleanup(portal.server.Close)

	return portal
}

// MobileAccounts answers the default answer of accounts.php. It holds the personal data
// of the holder, which no reader of the plugin decodes.
func MobileAccounts() map[string]any {
	return map[string]any{
		"status":            "Active",
		"statusId":          1,
		"signContractToken": ContractToken,
		"holder":            map[string]any{"name": HolderName, "personalNumber": PersonalNumber},
		"balances": []any{
			balance("BALANCE", "0.03"),
			balance("INTERNET", "2589.29"),
			balance("MINUTES", "56:16"),
			balance("SMS", "100.0"),
		},
		"activeCocktail": map[string]any{
			"cocktailId": 77,
			"untilDate":  "13/12/2026 01:50:47",
			"validityId": 3,
		},
	}
}

func balance(key, value string) map[string]any {
	return map[string]any{
		"balance":                 value,
		"accountGroupConfigItems": []any{map[string]any{"accountGroupKey": key}},
	}
}

// URL answers the base address of the fake portal.
func (p *Portal) URL() string {
	return p.server.URL
}

// Calls answers each request that the portal received.
func (p *Portal) Calls() []Call {
	p.mu.Lock()
	defer p.mu.Unlock()

	return append([]Call(nil), p.calls...)
}

// CallsTo answers the requests to one path.
func (p *Portal) CallsTo(path string) []Call {
	var matched []Call

	for _, call := range p.Calls() {
		if call.Path == path {
			matched = append(matched, call)
		}
	}

	return matched
}

// Selections answers the numbers that the requests selected, in order.
func (p *Portal) Selections() []string {
	var numbers []string

	for _, call := range p.CallsTo("/mymagti") {
		numbers = append(numbers, call.Query.Get("subscriber-number"))
	}

	return numbers
}

// Set changes the state of the portal under its lock.
func (p *Portal) Set(change func(p *Portal)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	change(p)
}

// ServeHTTP records the call and answers it.
func (p *Portal) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	call := Call{Method: r.Method, Path: r.URL.Path, Query: r.URL.Query()}

	if cookie, err := r.Cookie(CookieName); err == nil {
		call.Cookie = cookie.Value
	}

	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, formLimit)
		_ = r.ParseForm()
		call.Form = r.PostForm
	}

	p.mu.Lock()
	p.calls = append(p.calls, call)
	p.counts[call.Path]++
	count := p.counts[call.Path]
	override := p.Override
	p.mu.Unlock()

	if override != nil && override(w, r, count) {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.answer(w, r, call)
}

func (p *Portal) answer(w http.ResponseWriter, r *http.Request, call Call) {
	valid := p.Valid[call.Cookie]

	switch call.Path {
	case "/ka/mymagti/":
		p.rotate(w, call, valid)
		p.home(w, valid)
	case "/mymagti":
		if valid {
			p.Selected = call.Query.Get("subscriber-number")
		}

		http.Redirect(w, r, "/ka/mymagti/", http.StatusFound)
	case "/request/log-in.php":
		p.logIn(w, call)
	case "/ka/mymagti/statistics":
		if !valid {
			p.home(w, false)

			return
		}

		writeHTML(
			w,
			`<form><input type="hidden" name="csrf-token" value="`+StatisticsTok+`"></form>`,
		)
	default:
		p.data(w, call, valid)
	}
}

func (p *Portal) rotate(w http.ResponseWriter, call Call, valid bool) {
	if valid && p.Rotate != "" && call.Cookie != p.Rotate {
		http.SetCookie(w, &http.Cookie{Name: CookieName, Value: p.Rotate})
		p.Valid[p.Rotate] = true
	}
}

func (p *Portal) home(w http.ResponseWriter, valid bool) {
	if !valid {
		http.SetCookie(w, &http.Cookie{Name: CookieName, Value: AnonCookie})
		writeHTML(
			w,
			`<form><input type="hidden" name="csrf-token-login" value="`+LogInToken+`"></form>`,
		)

		return
	}

	var page strings.Builder

	page.WriteString(`<nav><a href="#">` + html.EscapeString(p.Selected) + `</a><ul>`)

	source := ""

	for _, subscriber := range p.Subscribers {
		if subscriber.Number == p.Selected {
			source = subscriber.Source

			continue
		}

		fmt.Fprintf(
			&page,
			`<li><a href="https://www.magticom.ge/mymagti?subscriber-number=%s"><span>%s</span> <b>%s</b></a></li>`,
			url.QueryEscape(subscriber.Number),
			html.EscapeString(subscriber.Label),
			html.EscapeString(strings.TrimPrefix(subscriber.Number, "d_")),
		)
	}

	page.WriteString(`</ul></nav><div class="box accounts" data-source="` + source + `"></div>`)
	writeHTML(w, page.String())
}

func (p *Portal) logIn(w http.ResponseWriter, call Call) {
	if call.Form.Get("csrf-token-login") != LogInToken || call.Cookie != AnonCookie {
		writeJSON(w, map[string]any{"message": "invalid-token"})

		return
	}

	if p.LogInMessage == "success" {
		http.SetCookie(w, &http.Cookie{Name: CookieName, Value: SessionCookie})
		p.Valid[SessionCookie] = true
	}

	writeJSON(w, map[string]any{"message": p.LogInMessage})
}

func (p *Portal) data(w http.ResponseWriter, call Call, valid bool) {
	if !valid {
		_, _ = w.Write([]byte("false"))

		return
	}

	switch call.Path {
	case "/request/accounts.php":
		writeJSON(w, p.Accounts)
	case "/request/cocktail-package.php":
		writeJSON(w, p.Cocktail)
	case "/request/accounts-isp.php":
		writeJSON(w, p.IspAccounts)
	case "/request/account-data.php":
		writeJSON(w, p.AccountData)
	case "/request/accept-terms.php":
		_, _ = w.Write([]byte("success"))
	case "/request/statistics.php":
		p.statistics(w, call)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (p *Portal) statistics(w http.ResponseWriter, call Call) {
	if call.Form.Get("csrf-token") != StatisticsTok {
		_, _ = w.Write([]byte("error"))

		return
	}

	if call.Form.Get("step") == "1" {
		_, _ = w.Write([]byte("success"))

		return
	}

	key := call.Form.Get("key") + "/" + call.Form.Get("charge-type")
	history, found := p.History[key]

	if !found {
		history = []any{}
	}

	writeJSON(w, map[string]any{"name": key, "key": call.Form.Get("key"), "history": history})
}

// UsageRow answers a row of a usage series: the out and the in of the day of the moment
// in milliseconds.
func UsageRow(day int64, out, in float64) map[string]any {
	return map[string]any{"day": day, "chartData": out, "chartDataIn": in}
}

// TotalRow answers the row that sums a series.
func TotalRow(out float64) map[string]any {
	return map[string]any{"sum": true, "chartData": out, "entryAmount": out}
}

// MoneyRow answers a row of the money series.
func MoneyRow(moment, amount, kind string) map[string]any {
	return map[string]any{"transactionDate": moment, "entryAmount": amount, "actionType": kind}
}

// HistoryKey names a series of the statistics in History.
func HistoryKey(key, charge string) string {
	return key + "/" + charge
}

func writeHTML(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte("<!DOCTYPE html><html><body>" + body + "</body></html>"))
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = json.NewEncoder(w).Encode(body)
}
