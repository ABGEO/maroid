package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"resty.dev/v3"

	"github.com/abgeo/maroid/plugins/magti/config"
	"github.com/abgeo/maroid/plugins/magti/dto"
	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/model"
)

const (
	// maxFailureBody bounds the part of the body of a failed answer that an error names.
	maxFailureBody = 512

	// maxResponseBody bounds an answer of the portal. A page is about 200 KiB.
	maxResponseBody = 10 << 20

	// maxRedirects bounds the redirects that a read of a page follows.
	maxRedirects = 5

	// userAgent names a browser, because the portal serves its pages to one.
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/155.0.0.0 Safari/537.36"

	pathHome            = "/ka/mymagti/"
	pathSelect          = "/mymagti"
	pathLogIn           = "/request/log-in.php"
	pathAccounts        = "/request/accounts.php"
	pathCocktailPackage = "/request/cocktail-package.php"
	pathAccountsISP     = "/request/accounts-isp.php"
	pathAccountData     = "/request/account-data.php"
	pathStatisticsPage  = "/ka/mymagti/statistics"
	pathAcceptTerms     = "/request/accept-terms.php"
	pathStatistics      = "/request/statistics.php"

	//nolint:gosec // G101: the name of a field of the form, not a credential.
	fieldLogInToken      = "csrf-token-login"
	fieldStatisticsToken = "csrf-token"
	querySubscriber      = "subscriber-number"
	statisticsTerms      = "statistics-terms"
	statisticsFlow       = "statistics"
	answerSuccess        = "success"

	// sessionRejectedBody is the answer of a data call to a session that the portal ended.
	sessionRejectedBody = "false"

	// otherSubscriberPrefix starts the number of a subscriber that is not a phone line, a
	// domain for example.
	otherSubscriberPrefix = "d_"

	mediaHTML = "text/html"
)

// ChargeType names the side of the money series of the statistics.
type ChargeType string

// The sides of the money series.
const (
	ChargeExpense ChargeType = "expense"
	ChargeIncome  ChargeType = "income"
)

// APIClientService defines the interface for interacting with the portal of the telecom
// service.
type APIClientService interface {
	SignIn(ctx context.Context, username string, password string) (model.Session, error)
	Subscribers(ctx context.Context, session model.Session) ([]model.Subscriber, error)
	Select(ctx context.Context, session model.Session, number string) (dto.SubscriberPage, error)
	MobileState(ctx context.Context, session model.Session, number string) (model.State, error)
	HomeLineState(ctx context.Context, session model.Session, number string) (model.State, error)
	OpenStatistics(ctx context.Context, session model.Session, number string) (string, error)
	Usage(
		ctx context.Context,
		session model.Session,
		statistics StatisticsQuery,
		series dto.UsageSeries,
	) ([]dto.UsageRow, error)
	Money(
		ctx context.Context,
		session model.Session,
		statistics StatisticsQuery,
		charge ChargeType,
	) ([]dto.MoneyRow, error)
}

// StatisticsQuery names the subscriber, the token, and the period of a read of the
// statistics.
type StatisticsQuery struct {
	Number string
	Token  string
	Period model.Period
}

// APIClient implements APIClientService. It holds no session and no cookie jar,
// because one instance serves the account of every workspace.
type APIClient struct {
	client *resty.Client
}

var _ APIClientService = (*APIClient)(nil)

// NewAPIClient creates a new APIClient. It follows no redirect by itself, because a
// redirect of a data call means an ended session.
func NewAPIClient(cfg *config.Config) *APIClient {
	client := resty.New().
		SetBaseURL(cfg.BaseURL).
		SetHeader("Accept", "application/json").
		SetHeader("User-Agent", userAgent).
		SetHeader("Origin", cfg.BaseURL).
		SetHeader("Referer", cfg.BaseURL+pathHome).
		SetTimeout(cfg.RequestTimeout).
		SetResponseBodyLimit(maxResponseBody).
		SetRedirectPolicy(resty.NoRedirectPolicy()).
		SetCookieJar(nil)

	return &APIClient{client: client}
}

// SignIn signs in with the form of the portal and answers the session: every cookie
// that the page of the form and the answer of the sign in set.
func (s *APIClient) SignIn(
	ctx context.Context,
	username string,
	password string,
) (model.Session, error) {
	session := model.Session{Cookies: map[string]string{}}

	page, err := s.readPage(ctx, &session, pathHome)
	if err != nil {
		return model.Session{}, fmt.Errorf("reading the sign in page: %w", err)
	}

	token := inputValue(page, fieldLogInToken)
	if token == "" {
		return model.Session{}, fmt.Errorf(
			"%w: the sign in page holds no CSRF token",
			errs.ErrRequestFailed,
		)
	}

	resp, err := s.request(ctx, session).
		SetFormData(map[string]string{
			"subscriber":    username,
			"password":      password,
			"action":        "",
			fieldLogInToken: token,
		}).
		Post(pathLogIn)
	if err != nil {
		return model.Session{}, fmt.Errorf(
			"%w: sending the sign in: %w",
			errs.ErrRequestFailed,
			err,
		)
	}

	if resp.StatusCode() != http.StatusOK {
		return model.Session{}, statusFailure(resp)
	}

	var answer dto.LogInAnswer
	if err = json.Unmarshal(resp.Bytes(), &answer); err != nil {
		return model.Session{}, fmt.Errorf(
			"%w: reading the answer of the sign in: %w",
			errs.ErrRequestFailed,
			err,
		)
	}

	if err = logInFailure(answer.Message); err != nil {
		return model.Session{}, err
	}

	keepCookies(&session, resp.Cookies())

	return session, nil
}

// Subscribers answers each subscriber of the account with its kind. The menu of a page
// holds no link to the selected subscriber, so a second page, of another subscriber,
// names the one that the first page left out.
func (s *APIClient) Subscribers(
	ctx context.Context,
	session model.Session,
) ([]model.Subscriber, error) {
	first, err := s.page(ctx, session, pathHome)
	if err != nil {
		return nil, fmt.Errorf("reading the subscribers: %w", err)
	}

	if len(first.Links) == 0 {
		return nil, fmt.Errorf("%w: the page names no subscriber", errs.ErrRequestFailed)
	}

	pivot := pivotLink(first.Links)

	second, err := s.Select(ctx, session, pivot.Number)
	if err != nil {
		return nil, fmt.Errorf("reading the subscribers: %w", err)
	}

	kinds := map[string]model.SubscriberKind{pivot.Number: second.Kind}
	subscribers := make([]model.Subscriber, 0, len(first.Links)+1)

	for _, link := range second.Links {
		if !holdsLink(first.Links, link.Number) {
			kinds[link.Number] = first.Kind
			subscribers = append(
				subscribers,
				model.Subscriber{Number: link.Number, Label: link.Label},
			)
		}
	}

	for _, link := range first.Links {
		subscribers = append(subscribers, model.Subscriber{Number: link.Number, Label: link.Label})
	}

	for index := range subscribers {
		kind, err := s.subscriberKind(ctx, session, subscribers[index].Number, kinds)
		if err != nil {
			return nil, err
		}

		subscribers[index].Kind = kind
	}

	return subscribers, nil
}

// Select selects the subscriber in the session, and answers its page. The portal answers
// every later call of the session for that subscriber.
func (s *APIClient) Select(
	ctx context.Context,
	session model.Session,
	number string,
) (dto.SubscriberPage, error) {
	page, err := s.page(ctx, session, pathSelect+"?"+url.Values{querySubscriber: {number}}.Encode())
	if err != nil {
		return dto.SubscriberPage{}, fmt.Errorf("selecting a subscriber: %w", err)
	}

	return page, nil
}

// MobileState answers the state of the selected mobile subscriber.
func (s *APIClient) MobileState(
	ctx context.Context,
	session model.Session,
	number string,
) (model.State, error) {
	var accounts dto.Accounts
	if err := s.get(ctx, session, pathAccounts, nil, &accounts); err != nil {
		return model.State{}, fmt.Errorf("reading the balances: %w", err)
	}

	var plan *dto.CocktailPackage

	if accounts.HasPlan() {
		plan = new(dto.CocktailPackage)

		query := url.Values{
			"package-id":  {accounts.ActiveCocktail.CocktailID.String()},
			"auto-top-up": {"1"},
			"validityId":  {accounts.ActiveCocktail.ValidityID.String()},
		}
		if err := s.get(ctx, session, pathCocktailPackage, query, plan); err != nil {
			return model.State{}, fmt.Errorf("reading the plan: %w", err)
		}
	}

	state, err := dto.MobileState(number, accounts, plan)
	if err != nil {
		return model.State{}, fmt.Errorf("reading the state: %w", err)
	}

	return state, nil
}

// HomeLineState answers the state of the selected home line.
func (s *APIClient) HomeLineState(
	ctx context.Context,
	session model.Session,
	number string,
) (model.State, error) {
	var accounts dto.IspAccounts
	if err := s.get(ctx, session, pathAccountsISP, nil, &accounts); err != nil {
		return model.State{}, fmt.Errorf("reading the balances: %w", err)
	}

	var data dto.AccountData

	query := url.Values{querySubscriber: {number}}
	if err := s.get(ctx, session, pathAccountData, query, &data); err != nil {
		return model.State{}, fmt.Errorf("reading the status: %w", err)
	}

	state, err := dto.HomeLineState(number, accounts, data)
	if err != nil {
		return model.State{}, fmt.Errorf("reading the state: %w", err)
	}

	return state, nil
}

// OpenStatistics accepts the terms of the statistics, opens them for the subscriber, and
// answers the token that each read of the statistics sends.
func (s *APIClient) OpenStatistics(
	ctx context.Context,
	session model.Session,
	number string,
) (string, error) {
	page, err := s.readPage(ctx, &session, pathStatisticsPage)
	if err != nil {
		return "", fmt.Errorf("reading the statistics page: %w", err)
	}

	token := inputValue(page, fieldStatisticsToken)
	if token == "" {
		return "", fmt.Errorf("%w: the statistics page holds no CSRF token", errs.ErrRequestFailed)
	}

	if _, err = s.post(ctx, session, pathAcceptTerms, map[string]string{
		"post": statisticsTerms,
	}); err != nil {
		return "", fmt.Errorf("accepting the terms of the statistics: %w", err)
	}

	answer, err := s.post(ctx, session, pathStatistics, map[string]string{
		"step":          "1",
		"flow":          statisticsFlow,
		"terms_post":    statisticsTerms,
		"mobile-number": number,
		"csrf-token":    token,
	})
	if err != nil {
		return "", fmt.Errorf("opening the statistics: %w", err)
	}

	if strings.Trim(answer, `" `) != answerSuccess {
		return "", fmt.Errorf(
			"%w: the statistics answered %q",
			errs.ErrRequestFailed,
			cutFailure(answer),
		)
	}

	return token, nil
}

// Usage answers the rows of one usage series of the period.
func (s *APIClient) Usage(
	ctx context.Context,
	session model.Session,
	statistics StatisticsQuery,
	series dto.UsageSeries,
) ([]dto.UsageRow, error) {
	var rows []dto.UsageRow
	if err := s.statistics(
		ctx,
		session,
		statistics,
		string(series),
		ChargeExpense,
		&rows,
	); err != nil {
		return nil, fmt.Errorf("reading the usage %s: %w", series, err)
	}

	return rows, nil
}

// Money answers the rows of one side of the money series of the period.
func (s *APIClient) Money(
	ctx context.Context,
	session model.Session,
	statistics StatisticsQuery,
	charge ChargeType,
) ([]dto.MoneyRow, error) {
	var rows []dto.MoneyRow
	if err := s.statistics(ctx, session, statistics, "GEL", charge, &rows); err != nil {
		return nil, fmt.Errorf("reading the money %s: %w", charge, err)
	}

	return rows, nil
}

func (s *APIClient) statistics(
	ctx context.Context,
	session model.Session,
	statistics StatisticsQuery,
	key string,
	charge ChargeType,
	rows any,
) error {
	body, err := s.post(ctx, session, pathStatistics, map[string]string{
		"step":          "3",
		"flow":          statisticsFlow,
		"mobile-number": statistics.Number,
		"csrf-token":    statistics.Token,
		"key":           key,
		"date[from]":    statistics.Period.From.PortalForm(),
		"date[to]":      statistics.Period.To.PortalForm(),
		"charge-type":   string(charge),
	})
	if err != nil {
		return err
	}

	var answer dto.Statistics
	if err = json.Unmarshal([]byte(body), &answer); err != nil {
		return fmt.Errorf("%w: reading the statistics: %w", errs.ErrRequestFailed, err)
	}

	if len(answer.History) == 0 || string(answer.History) == "null" {
		return nil
	}

	if err = json.Unmarshal(answer.History, rows); err != nil {
		return fmt.Errorf("%w: reading the history: %w", errs.ErrRequestFailed, err)
	}

	return nil
}

func (s *APIClient) subscriberKind(
	ctx context.Context,
	session model.Session,
	number string,
	known map[string]model.SubscriberKind,
) (model.SubscriberKind, error) {
	if strings.HasPrefix(number, otherSubscriberPrefix) {
		return model.SubscriberKindOther, nil
	}

	if kind, found := known[number]; found {
		return kind, nil
	}

	page, err := s.Select(ctx, session, number)
	if err != nil {
		return "", err
	}

	return page.Kind, nil
}

// page reads a page of the session, and answers what its menu tells about the
// subscribers.
func (s *APIClient) page(
	ctx context.Context,
	session model.Session,
	path string,
) (dto.SubscriberPage, error) {
	document, err := s.readPage(ctx, &session, path)
	if err != nil {
		return dto.SubscriberPage{}, err
	}

	if inputValue(document, fieldLogInToken) != "" {
		return dto.SubscriberPage{}, errs.ErrSessionRejected
	}

	return dto.SubscriberPage{Kind: kindOf(document), Links: subscriberLinks(document)}, nil
}

// readPage reads a page, and follows its redirects with the cookies of the session. The
// cookies that an answer sets join the session.
func (s *APIClient) readPage(
	ctx context.Context,
	session *model.Session,
	path string,
) (*html.Node, error) {
	for range maxRedirects {
		resp, err := s.request(ctx, *session).SetHeader("Accept", mediaHTML).Get(path)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errs.ErrRequestFailed, err)
		}

		keepCookies(session, resp.Cookies())

		if isRedirect(resp.StatusCode()) {
			path, err = s.redirectPath(resp)
			if err != nil {
				return nil, err
			}

			continue
		}

		if resp.StatusCode() != http.StatusOK {
			return nil, statusFailure(resp)
		}

		document, err := html.Parse(bytes.NewReader(resp.Bytes()))
		if err != nil {
			return nil, fmt.Errorf("%w: parsing a page: %w", errs.ErrRequestFailed, err)
		}

		return document, nil
	}

	return nil, fmt.Errorf("%w: more than %d redirects", errs.ErrRequestFailed, maxRedirects)
}

// redirectPath answers the path of a redirect. A redirect to another host is a failure,
// because the cookies of the session belong to the portal.
func (s *APIClient) redirectPath(resp *resty.Response) (string, error) {
	location, err := url.Parse(resp.Header().Get("Location"))
	if err != nil {
		return "", fmt.Errorf("%w: a redirect to no address: %w", errs.ErrRequestFailed, err)
	}

	base, err := url.Parse(s.client.BaseURL())
	if err != nil {
		return "", fmt.Errorf("%w: reading the base address: %w", errs.ErrRequestFailed, err)
	}

	target := base.ResolveReference(location)
	if target.Host != base.Host {
		return "", fmt.Errorf("%w: a redirect to another host", errs.ErrRequestFailed)
	}

	return target.RequestURI(), nil
}

// get sends a data call and decodes its JSON answer.
func (s *APIClient) get(
	ctx context.Context,
	session model.Session,
	path string,
	query url.Values,
	result any,
) error {
	request := s.request(ctx, session)
	if query != nil {
		request.SetQueryParamsFromValues(query)
	}

	resp, err := request.Get(path)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrRequestFailed, err)
	}

	if err = answerFailure(resp); err != nil {
		return err
	}

	if err = json.Unmarshal(resp.Bytes(), result); err != nil {
		return fmt.Errorf("%w: reading the answer: %w", errs.ErrRequestFailed, err)
	}

	return nil
}

// post sends a form of a data call and answers the body.
func (s *APIClient) post(
	ctx context.Context,
	session model.Session,
	path string,
	form map[string]string,
) (string, error) {
	resp, err := s.request(ctx, session).
		SetHeader("X-Requested-With", "XMLHttpRequest").
		SetFormData(form).
		Post(path)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errs.ErrRequestFailed, err)
	}

	if err = answerFailure(resp); err != nil {
		return "", err
	}

	return resp.String(), nil
}

func (s *APIClient) request(ctx context.Context, session model.Session) *resty.Request {
	request := s.client.R().SetContext(ctx)

	for name, value := range session.Cookies {
		//nolint:gosec // G124: a request carries a name and a value, and no attribute.
		request.SetCookie(&http.Cookie{Name: name, Value: value})
	}

	return request
}

// answerFailure answers the failure of a data call. An ended session answers a redirect
// to the sign in page, 401, 403, or the body false.
func answerFailure(resp *resty.Response) error {
	code := resp.StatusCode()

	switch {
	case code == http.StatusUnauthorized, code == http.StatusForbidden, isRedirect(code):
		return errs.ErrSessionRejected
	case code != http.StatusOK:
		return statusFailure(resp)
	case strings.TrimSpace(resp.String()) == sessionRejectedBody:
		return errs.ErrSessionRejected
	}

	return nil
}

func logInFailure(message string) error {
	switch message {
	case dto.LogInSuccess:
		return nil
	case dto.LogInInvalidCredentials:
		return errs.ErrCredentialsRejected
	case dto.LogInCodeRequired:
		return errs.ErrCodeRequired
	case dto.LogInLocked:
		return errs.ErrAccountLocked
	}

	return fmt.Errorf("%w: the sign in answered %q", errs.ErrRequestFailed, cutFailure(message))
}

func isRedirect(code int) bool {
	return code >= http.StatusMultipleChoices && code < http.StatusBadRequest
}

func keepCookies(session *model.Session, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		if cookie.MaxAge < 0 || cookie.Value == "" {
			delete(session.Cookies, cookie.Name)

			continue
		}

		session.Cookies[cookie.Name] = cookie.Value
	}
}

// pivotLink answers the link that Subscribers selects. A phone line answers its kind on
// its page, so it comes before a subscriber of another kind.
func pivotLink(links []dto.SubscriberLink) dto.SubscriberLink {
	for _, link := range links {
		if !strings.HasPrefix(link.Number, otherSubscriberPrefix) {
			return link
		}
	}

	return links[0]
}

func holdsLink(links []dto.SubscriberLink, number string) bool {
	for _, link := range links {
		if link.Number == number {
			return true
		}
	}

	return false
}

// subscriberLinks answers each link of the page whose query names a subscriber, once.
func subscriberLinks(document *html.Node) []dto.SubscriberLink {
	var links []dto.SubscriberLink

	for node := range document.Descendants() {
		if node.Type != html.ElementNode || node.DataAtom != atom.A {
			continue
		}

		target, err := url.Parse(attribute(node, "href"))
		if err != nil {
			continue
		}

		number := strings.TrimSpace(target.Query().Get(querySubscriber))
		if number == "" || holdsLink(links, number) {
			continue
		}

		links = append(links, dto.SubscriberLink{Number: number, Label: linkLabel(node, number)})
	}

	return links
}

// linkLabel answers the text of a link without the number. The portal writes a domain
// without the prefix d_.
func linkLabel(node *html.Node, number string) string {
	var text strings.Builder

	for child := range node.Descendants() {
		if child.Type == html.TextNode {
			text.WriteString(child.Data)
			text.WriteString(" ")
		}
	}

	label := strings.ReplaceAll(text.String(), number, "")
	label = strings.ReplaceAll(label, strings.TrimPrefix(number, otherSubscriberPrefix), "")

	return strings.Join(strings.Fields(label), " ")
}

// kindOf answers the kind of the selected subscriber from the source of the balances
// of its page.
func kindOf(document *html.Node) model.SubscriberKind {
	for node := range document.Descendants() {
		if node.Type != html.ElementNode || !hasClass(node, "accounts") {
			continue
		}

		source := attribute(node, "data-source")

		switch {
		case strings.HasSuffix(source, strings.TrimPrefix(pathAccounts, "/")):
			return model.SubscriberKindMobile
		case strings.HasSuffix(source, strings.TrimPrefix(pathAccountsISP, "/")):
			return model.SubscriberKindHomeLine
		}
	}

	return model.SubscriberKindOther
}

func inputValue(document *html.Node, name string) string {
	for node := range document.Descendants() {
		if node.Type == html.ElementNode && node.DataAtom == atom.Input &&
			attribute(node, "name") == name {
			return attribute(node, "value")
		}
	}

	return ""
}

func hasClass(node *html.Node, class string) bool {
	return strings.Contains(" "+attribute(node, "class")+" ", " "+class+" ")
}

func attribute(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}

	return ""
}

// statusFailure names the status and the start of the body of a failed answer. A page of
// the portal carries the CSRF token of a session, so the body of an HTML answer stays out.
func statusFailure(resp *resty.Response) error {
	if strings.HasPrefix(resp.Header().Get("Content-Type"), mediaHTML) {
		return fmt.Errorf("%w: returned [%d] a page", errs.ErrRequestFailed, resp.StatusCode())
	}

	return fmt.Errorf(
		"%w: returned [%d] %s",
		errs.ErrRequestFailed,
		resp.StatusCode(),
		cutFailure(resp.String()),
	)
}

// cutFailure answers at most maxFailureBody bytes of the body, cut at a rune.
func cutFailure(raw string) string {
	body := strings.TrimSpace(raw)
	if len(body) <= maxFailureBody {
		return body
	}

	cut := maxFailureBody
	for cut > 0 && !utf8.RuneStart(body[cut]) {
		cut--
	}

	return body[:cut]
}
