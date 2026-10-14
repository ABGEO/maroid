package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"resty.dev/v3"

	"github.com/abgeo/maroid/plugins/telmico/config"
	"github.com/abgeo/maroid/plugins/telmico/dto"
	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/model"
)

const (
	// maxFailureBody bounds the part of the body of a failed answer that an error names.
	maxFailureBody = 512

	// maxResponseBody bounds an answer of the portal. A bill is about 75 KiB.
	maxResponseBody = 10 << 20

	// pageSize is the count of items that one request of a list asks for.
	pageSize = 100

	// maxPages bounds the pages of one list. A month holds a few items, so a count above it
	// is a broken answer.
	maxPages = 50

	cookieName = "_tpcabinet_app_u_"
	csrfHeader = "X-CSRF-TOKEN"
	csrfMeta   = "_csrf"

	// userAgent names a browser, because the portal serves its pages to one.
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/155.0.0.0 Safari/537.36"

	pathStart     = "/"
	pathLogin     = "/login"
	pathPrivate   = "/private"
	pathAccounts  = "/private/accounts"
	pathAccount   = "/private/account/"
	pathPayments  = "/allpaymentsapi"
	pathReceipts  = "/receiptsapi"
	pathReceipt   = "/private/receipt/"
	signInRefusal = "error.bad_credentials"

	mediaHTML = "text/html"
	mediaPDF  = "application/pdf"

	// pdfSignature starts every PDF document.
	pdfSignature = "%PDF"
)

// APIClientService defines the interface for interacting with the portal of the
// electricity service.
type APIClientService interface {
	SignIn(ctx context.Context, username string, password string) (model.Session, error)
	Accounts(ctx context.Context, session model.Session) ([]dto.Account, error)
	AccountDetail(
		ctx context.Context,
		session model.Session,
		account string,
	) (dto.AccountDetail, error)
	Payments(
		ctx context.Context,
		session model.Session,
		account string,
		month model.Month,
	) ([]dto.Payment, error)
	Receipts(
		ctx context.Context,
		session model.Session,
		account string,
		month model.Month,
	) ([]dto.Receipt, error)
	ReceiptDocument(ctx context.Context, session model.Session, billID uuid.UUID) ([]byte, error)
}

// APIClient implements APIClientService. It holds no session and no cookie jar,
// because one instance serves the account of every workspace.
type APIClient struct {
	client *resty.Client
}

var _ APIClientService = (*APIClient)(nil)

// NewAPIClient creates a new APIClient. The portal redirects to http://, so the client
// follows no redirect and reads each one.
func NewAPIClient(cfg *config.Config) *APIClient {
	client := resty.New().
		SetBaseURL(cfg.BaseURL).
		SetHeader("Accept", "application/json").
		SetHeader("User-Agent", userAgent).
		SetHeader("Origin", cfg.BaseURL).
		SetHeader("Referer", cfg.BaseURL+"/").
		SetTimeout(cfg.RequestTimeout).
		SetResponseBodyLimit(maxResponseBody).
		SetRedirectPolicy(resty.NoRedirectPolicy()).
		SetCookieJar(nil)

	return &APIClient{client: client}
}

// SignIn signs in with the form of the portal and answers the new session. The portal
// binds the CSRF token to the session and replaces the session at the sign in, so the
// token of the new session comes from a page that the new session reads.
func (s *APIClient) SignIn(
	ctx context.Context,
	username string,
	password string,
) (model.Session, error) {
	anonymous, err := s.readPage(ctx, pathStart, "")
	if err != nil {
		return model.Session{}, fmt.Errorf("reading the sign in page: %w", err)
	}

	resp, err := s.client.R().
		SetContext(ctx).
		SetHeader("Accept", mediaHTML).
		SetCookie(sessionCookie(anonymous.Cookie)).
		SetFormData(map[string]string{
			"username": username,
			"password": password,
			csrfMeta:   anonymous.CSRFToken,
		}).
		Post(pathLogin)
	if err != nil {
		return model.Session{}, fmt.Errorf(
			"%w: sending the sign in: %w",
			errs.ErrRequestFailed,
			err,
		)
	}

	cookie, err := signedInCookie(resp)
	if err != nil {
		return model.Session{}, err
	}

	signedIn, err := s.readPage(ctx, pathPrivate, cookie)
	if err != nil {
		return model.Session{}, fmt.Errorf("reading the page of the new session: %w", err)
	}

	return signedIn, nil
}

// Accounts answers the customers of the account of the session.
func (s *APIClient) Accounts(ctx context.Context, session model.Session) ([]dto.Account, error) {
	var accounts []dto.Account

	if err := s.post(ctx, session, pathAccounts, nil, &accounts); err != nil {
		return nil, fmt.Errorf("requesting the customers: %w", err)
	}

	return accounts, nil
}

// AccountDetail answers the address of one customer of the account.
func (s *APIClient) AccountDetail(
	ctx context.Context,
	session model.Session,
	account string,
) (dto.AccountDetail, error) {
	var detail dto.AccountDetail

	if err := s.post(ctx, session, accountPath(account), nil, &detail); err != nil {
		return dto.AccountDetail{}, fmt.Errorf("requesting the detail of a customer: %w", err)
	}

	return detail, nil
}

// Payments answers the transactions of one customer in one month, every page of them.
func (s *APIClient) Payments(
	ctx context.Context,
	session model.Session,
	account string,
	month model.Month,
) ([]dto.Payment, error) {
	payments, err := readPages(func(page int) ([]dto.Payment, int, error) {
		form := monthForm(page, month, month)
		form["sort"] = "operDate,desc"

		var answer dto.PaymentsPage

		err := s.post(ctx, session, accountPath(account)+pathPayments, form, &answer)

		return answer.Items, answer.TotalPages, err
	})
	if err != nil {
		return nil, fmt.Errorf("requesting the transactions of %s: %w", month, err)
	}

	return payments, nil
}

// Receipts answers the bills of one customer in one month, every page of them. The
// receipts endpoint takes to_month as an exclusive bound, unlike the payments endpoint.
func (s *APIClient) Receipts(
	ctx context.Context,
	session model.Session,
	account string,
	month model.Month,
) ([]dto.Receipt, error) {
	receipts, err := readPages(func(page int) ([]dto.Receipt, int, error) {
		var answer dto.ReceiptsPage

		err := s.post(
			ctx,
			session,
			accountPath(account)+pathReceipts,
			monthForm(page, month, month.Next()),
			&answer,
		)

		return answer.Items, answer.TotalPages, err
	})
	if err != nil {
		return nil, fmt.Errorf("requesting the bills of %s: %w", month, err)
	}

	return receipts, nil
}

// ReceiptDocument answers the PDF document of one bill. The portal serves it only
// inside a session.
func (s *APIClient) ReceiptDocument(
	ctx context.Context,
	session model.Session,
	billID uuid.UUID,
) ([]byte, error) {
	resp, err := s.client.R().
		SetContext(ctx).
		SetHeader("Accept", mediaPDF).
		SetCookie(sessionCookie(session.Cookie)).
		Get(pathReceipt + billID.String())
	if err != nil {
		return nil, fmt.Errorf("%w: downloading a bill: %w", errs.ErrRequestFailed, err)
	}

	if err = answerFailure(resp); err != nil {
		return nil, fmt.Errorf("downloading a bill: %w", err)
	}

	document := resp.Bytes()
	if !bytes.HasPrefix(document, []byte(pdfSignature)) {
		return nil, fmt.Errorf("%w: the bill is no PDF document", errs.ErrRequestFailed)
	}

	return document, nil
}

// readPage reads a page of the portal and answers the session cookie and the CSRF
// token that it carries. The cookie of the answer replaces the one sent, when it sets one.
func (s *APIClient) readPage(
	ctx context.Context,
	path string,
	cookie string,
) (model.Session, error) {
	request := s.client.R().
		SetContext(ctx).
		SetHeader("Accept", mediaHTML)

	if cookie != "" {
		request.SetCookie(sessionCookie(cookie))
	}

	resp, err := request.Get(path)
	if err != nil {
		return model.Session{}, fmt.Errorf("%w: %w", errs.ErrRequestFailed, err)
	}

	if resp.StatusCode() != http.StatusOK {
		return model.Session{}, statusFailure(resp)
	}

	if answered := cookieOf(resp); answered != "" {
		cookie = answered
	}

	if cookie == "" {
		return model.Session{}, fmt.Errorf(
			"%w: the page set no session cookie",
			errs.ErrRequestFailed,
		)
	}

	token, err := csrfToken(resp.Bytes())
	if err != nil {
		return model.Session{}, err
	}

	return model.Session{Cookie: cookie, CSRFToken: token}, nil
}

// post sends a call of the API, which the portal takes as a POST with the session
// cookie and the CSRF token. A nil form sends no body.
func (s *APIClient) post(
	ctx context.Context,
	session model.Session,
	path string,
	form map[string]string,
	result any,
) error {
	request := s.client.R().
		SetContext(ctx).
		SetCookie(sessionCookie(session.Cookie)).
		SetHeader(csrfHeader, session.CSRFToken).
		SetForceResponseContentType("application/json").
		SetResult(result)

	if form != nil {
		request.SetMultipartFormData(form)
	}

	resp, err := request.Post(path)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrRequestFailed, err)
	}

	return answerFailure(resp)
}

// signedInCookie reads the answer of the sign in. The portal redirects to /private with
// a new session cookie, or back to the sign in page with the reason in the query.
func signedInCookie(resp *resty.Response) (string, error) {
	if resp.StatusCode() != http.StatusFound {
		return "", statusFailure(resp)
	}

	location, err := url.Parse(resp.Header().Get("Location"))
	if err != nil {
		return "", fmt.Errorf(
			"%w: the sign in redirected to no address: %w",
			errs.ErrRequestFailed,
			err,
		)
	}

	switch {
	case location.Query().Get("error") == signInRefusal:
		return "", errs.ErrCredentialsRejected
	case location.Path != pathPrivate:
		return "", fmt.Errorf(
			"%w: the sign in redirected to %q",
			errs.ErrRequestFailed,
			location.Path,
		)
	}

	cookie := cookieOf(resp)
	if cookie == "" {
		return "", fmt.Errorf("%w: the sign in set no session cookie", errs.ErrRequestFailed)
	}

	return cookie, nil
}

// answerFailure answers the failure of a call. An ended session answers 401, a token
// that does not match answers 403, and a page of the browser answers a redirect.
func answerFailure(resp *resty.Response) error {
	code := resp.StatusCode()

	switch {
	case code == http.StatusUnauthorized, code == http.StatusForbidden:
		return errs.ErrSessionRejected
	case code >= http.StatusMultipleChoices && code < http.StatusBadRequest:
		return errs.ErrSessionRejected
	case code != http.StatusOK:
		return statusFailure(resp)
	}

	return nil
}

// readPages reads each page of a list. The portal counts the pages from zero in a
// request.
func readPages[T any](read func(page int) ([]T, int, error)) ([]T, error) {
	var items []T

	for page := 0; ; page++ {
		answered, totalPages, err := read(page)
		if err != nil {
			return nil, err
		}

		items = append(items, answered...)

		if page+1 >= totalPages {
			return items, nil
		}

		if page+1 >= maxPages {
			return nil, fmt.Errorf(
				"%w: the list names %d pages, more than %d",
				errs.ErrRequestFailed,
				totalPages,
				maxPages,
			)
		}
	}
}

func monthForm(page int, from model.Month, to model.Month) map[string]string {
	return map[string]string{
		"page":       strconv.Itoa(page),
		"size":       strconv.Itoa(pageSize),
		"from_month": from.PortalForm(),
		"to_month":   to.PortalForm(),
	}
}

func accountPath(account string) string {
	return pathAccount + url.PathEscape(account)
}

func sessionCookie(value string) *http.Cookie {
	//nolint:gosec // G124: a request carries a name and a value, and no attribute.
	return &http.Cookie{Name: cookieName, Value: value}
}

func cookieOf(resp *resty.Response) string {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == cookieName {
			return cookie.Value
		}
	}

	return ""
}

// csrfToken answers the content of <meta name="_csrf"> of a page.
func csrfToken(page []byte) (string, error) {
	document, err := html.Parse(bytes.NewReader(page))
	if err != nil {
		return "", fmt.Errorf("%w: parsing a page: %w", errs.ErrRequestFailed, err)
	}

	for node := range document.Descendants() {
		if node.Type != html.ElementNode || node.DataAtom != atom.Meta {
			continue
		}

		if attribute(node, "name") == csrfMeta {
			if token := attribute(node, "content"); token != "" {
				return token, nil
			}
		}
	}

	return "", fmt.Errorf("%w: the page holds no CSRF token", errs.ErrRequestFailed)
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

// cutFailure answers at most maxFailureBody bytes of the body, cut at a rune. The answer
// of a failure carries no session.
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
