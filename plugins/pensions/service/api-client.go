package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"resty.dev/v3"

	"github.com/abgeo/maroid/plugins/pensions/config"
	"github.com/abgeo/maroid/plugins/pensions/dto"
	"github.com/abgeo/maroid/plugins/pensions/errs"
	"github.com/abgeo/maroid/plugins/pensions/model"
)

const (
	// maxFailureBody bounds the part of the body of a failed answer that an error names.
	maxFailureBody = 512

	// maxResponseBody bounds an answer of the pension service. The history of the unit
	// prices of one portfolio is about 400 KiB.
	maxResponseBody = 10 << 20

	// pageSize is the count of contributions that one request asks for.
	pageSize = 100

	// maxPages bounds the pages of one month. A month holds a few contributions, so a
	// count above it is a broken answer.
	maxPages = 50

	languageCode = "ka-GE"
	pageSuccess  = "success"

	pathSignIn        = "/v1/auth/participant-auth"
	pathParticipant   = "/v2/contributions/participant/get"
	pathAssets        = "/v2/contributions/participant/assets"
	pathChart         = "/v1/contributions/participant/chart-data"
	pathUnitPrices    = "/v2/contributions/nav-asset-value-portfolios"
	pathContributions = "/v1/fbo/contributions"
)

// APIClientService defines the interface for interacting with the pension service.
type APIClientService interface {
	SignIn(ctx context.Context, username, password string) (model.Session, error)
	Participant(ctx context.Context, session model.Session) (model.Participant, error)
	PortfolioShares(ctx context.Context, session model.Session) ([]model.PortfolioShare, error)
	AssetValues(
		ctx context.Context,
		session model.Session,
		month model.Month,
	) ([]model.AssetValue, error)
	PortfolioUnitPrices(
		ctx context.Context,
		session model.Session,
		portfolioIDs []int64,
	) ([]model.PortfolioUnitPrice, error)
	Contributions(
		ctx context.Context,
		session model.Session,
		month model.Month,
	) ([]model.Contribution, error)
}

// StatusError is an answer of the pension service with a status that is not a success.
// It wraps errs.ErrRequestFailed.
type StatusError struct {
	Status int
	// Body is at most maxFailureBody bytes of the answer.
	Body string
}

// Error names the status and the start of the body.
func (e *StatusError) Error() string {
	return fmt.Sprintf("%s: returned [%d] %s", errs.ErrRequestFailed, e.Status, e.Body)
}

// Unwrap answers errs.ErrRequestFailed.
func (e *StatusError) Unwrap() error {
	return errs.ErrRequestFailed
}

// APIClient implements APIClientService. It holds no session, because one instance
// serves the account of every person.
type APIClient struct {
	client *resty.Client
}

var _ APIClientService = (*APIClient)(nil)

// NewAPIClient creates a new APIClient.
func NewAPIClient(cfg *config.Config) *APIClient {
	client := resty.New().
		SetBaseURL(cfg.BaseURL).
		SetHeader("Accept", "application/json").
		SetTimeout(cfg.RequestTimeout).
		SetResponseBodyLimit(maxResponseBody).
		SetCookieJar(nil)

	return &APIClient{client: client}
}

// SignIn signs in with the password, and answers the access token. The pension service
// answers a rejection with a status of the client or with no token.
func (s *APIClient) SignIn(
	ctx context.Context,
	username string,
	password string,
) (model.Session, error) {
	resp, err := s.client.R().
		SetContext(ctx).
		SetBody(dto.SignInRequest{
			Username:     username,
			PasswordOne:  password,
			PasswordTwo:  "",
			LanguageCode: languageCode,
		}).
		Post(pathSignIn)
	if err != nil {
		return model.Session{}, fmt.Errorf(
			"%w: sending the sign in: %w",
			errs.ErrRequestFailed,
			err,
		)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden:
		return model.Session{}, errs.ErrCredentialsRejected
	default:
		return model.Session{}, statusFailure(resp)
	}

	var answer dto.SignInResponse
	if err = decode(resp, &answer); err != nil {
		return model.Session{}, err
	}

	if answer.AccessToken == "" {
		return model.Session{}, errs.ErrCredentialsRejected
	}

	return model.Session{Token: answer.AccessToken}, nil
}

// Participant reads the holder and the summary of the account.
func (s *APIClient) Participant(
	ctx context.Context,
	session model.Session,
) (model.Participant, error) {
	var answer dto.ParticipantResponse
	if err := s.call(ctx, session, http.MethodGet, pathParticipant, nil, nil, &answer); err != nil {
		return model.Participant{}, fmt.Errorf("reading the participant: %w", err)
	}

	return answer.MapToModel(), nil
}

// PortfolioShares reads the share of the account in each portfolio.
func (s *APIClient) PortfolioShares(
	ctx context.Context,
	session model.Session,
) ([]model.PortfolioShare, error) {
	var answer []dto.AssetResponse
	if err := s.call(ctx, session, http.MethodGet, pathAssets, nil, nil, &answer); err != nil {
		return nil, fmt.Errorf("reading the portfolio shares: %w", err)
	}

	shares := make([]model.PortfolioShare, 0, len(answer))
	for i := range answer {
		shares = append(shares, answer[i].MapToModel())
	}

	return shares, nil
}

// AssetValues reads the asset value of the account on each date of the month that the
// pension service reports.
func (s *APIClient) AssetValues(
	ctx context.Context,
	session model.Session,
	month model.Month,
) ([]model.AssetValue, error) {
	body := dto.ChartRequest{
		DateFrom: month.First().Format(time.DateOnly),
		DateTo:   month.Last().Format(time.DateOnly),
	}

	var answer dto.ChartResponse
	if err := s.call(ctx, session, http.MethodPost, pathChart, body, nil, &answer); err != nil {
		return nil, fmt.Errorf("reading the asset values: %w", err)
	}

	if answer.Failed() {
		return nil, fmt.Errorf(
			"%w: the chart answered errors %s",
			errs.ErrRequestFailed,
			cutFailure(string(answer.Errors)),
		)
	}

	values := make([]model.AssetValue, 0, len(answer.Result))

	for i := range answer.Result {
		value, err := answer.Result[i].MapToModel()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errs.ErrRequestFailed, err)
		}

		values = append(values, value)
	}

	return values, nil
}

// PortfolioUnitPrices reads the whole history of the unit price of each portfolio. No
// portfolio sends no call.
func (s *APIClient) PortfolioUnitPrices(
	ctx context.Context,
	session model.Session,
	portfolioIDs []int64,
) ([]model.PortfolioUnitPrice, error) {
	if len(portfolioIDs) == 0 {
		return nil, nil
	}

	body := dto.NavRequest{PortfolioIDs: portfolioIDs}

	var answer []dto.NavPortfolio
	if err := s.call(
		ctx,
		session,
		http.MethodPost,
		pathUnitPrices,
		body,
		nil,
		&answer,
	); err != nil {
		return nil, fmt.Errorf("reading the unit prices: %w", err)
	}

	var prices []model.PortfolioUnitPrice

	for _, portfolio := range answer {
		for i := range portfolio.NetAssetValue {
			price, err := portfolio.NetAssetValue[i].MapToModel()
			if err != nil {
				return nil, fmt.Errorf("%w: %w", errs.ErrRequestFailed, err)
			}

			if price.PortfolioID == 0 {
				price.PortfolioID = portfolio.PortfolioID
			}

			prices = append(prices, price)
		}
	}

	return prices, nil
}

// Contributions reads every page of the contributions of the month.
func (s *APIClient) Contributions(
	ctx context.Context,
	session model.Session,
	month model.Month,
) ([]model.Contribution, error) {
	var contributions []model.Contribution

	for page := 1; ; page++ {
		answer, err := s.contributionsPage(ctx, session, month, page)
		if err != nil {
			return nil, fmt.Errorf("reading page %d of the contributions: %w", page, err)
		}

		for i := range answer.Data.Result {
			contribution, err := answer.Data.Result[i].MapToModel()
			if err != nil {
				return nil, fmt.Errorf("%w: %w", errs.ErrRequestFailed, err)
			}

			contributions = append(contributions, contribution)
		}

		if page >= answer.Data.PageCount {
			return contributions, nil
		}

		if answer.Data.PageCount > maxPages {
			return nil, fmt.Errorf(
				"%w: the contributions name %d pages, more than %d",
				errs.ErrRequestFailed,
				answer.Data.PageCount,
				maxPages,
			)
		}
	}
}

func (s *APIClient) contributionsPage(
	ctx context.Context,
	session model.Session,
	month model.Month,
	page int,
) (dto.ContributionsPage, error) {
	query := map[string]string{
		"pageNumber": strconv.Itoa(page),
		"pageSize":   strconv.Itoa(pageSize),
		"startDate":  month.First().Format(time.DateOnly),
		"endDate":    month.Last().Format(time.DateOnly),
	}

	var answer dto.ContributionsPage
	if err := s.call(
		ctx,
		session,
		http.MethodGet,
		pathContributions,
		nil,
		query,
		&answer,
	); err != nil {
		return dto.ContributionsPage{}, err
	}

	if answer.Status != pageSuccess || answer.Code != http.StatusOK {
		return dto.ContributionsPage{}, fmt.Errorf(
			"%w: the page answered status %q and code %d",
			errs.ErrRequestFailed,
			answer.Status,
			answer.Code,
		)
	}

	return answer, nil
}

// call sends one call with the session and decodes the answer. An ended session answers
// 401. A nil body sends none.
func (s *APIClient) call(
	ctx context.Context,
	session model.Session,
	method string,
	path string,
	body any,
	query map[string]string,
	result any,
) error {
	request := s.client.R().
		SetContext(ctx).
		SetAuthToken(session.Token).
		SetQueryParams(query)

	if body != nil {
		request.SetBody(body)
	}

	resp, err := request.Execute(method, path)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrRequestFailed, err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return decode(resp, result)
	case http.StatusUnauthorized:
		return errs.ErrSessionRejected
	default:
		return statusFailure(resp)
	}
}

func decode(resp *resty.Response, result any) error {
	if err := json.Unmarshal(resp.Bytes(), result); err != nil {
		return fmt.Errorf("%w: decoding the answer: %w", errs.ErrRequestFailed, err)
	}

	return nil
}

func statusFailure(resp *resty.Response) error {
	return &StatusError{Status: resp.StatusCode(), Body: cutFailure(resp.String())}
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
