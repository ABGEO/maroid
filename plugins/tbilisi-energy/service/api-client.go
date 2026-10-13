package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"resty.dev/v3"

	"github.com/abgeo/maroid/plugins/tbilisi-energy/config"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/dto"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/errs"
)

const (
	// maxFailureBody bounds the part of the body of a failed answer that an error names.
	maxFailureBody = 512

	// maxResponseBody bounds an answer of the gas service. A bill is about 600 KiB.
	maxResponseBody = 10 << 20

	headerToken = "token"
)

// APIClientService defines the interface for interacting with the gas service.
type APIClientService interface {
	Authenticate(ctx context.Context, username string, password string) (string, error)
	GetCustomers(ctx context.Context, token string) ([]dto.Customer, error)
	GetTransactions(
		ctx context.Context,
		token string,
		request dto.TransactionsRequest,
	) ([]dto.Transaction, error)
	DownloadFile(ctx context.Context, token string, path string) ([]byte, error)
}

// APIClient implements APIClientService. It holds no credential, because one instance
// serves the account of every workspace.
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
		SetResponseBodyLimit(maxResponseBody)

	return &APIClient{
		client: client,
	}
}

// Authenticate signs in to the gas account and answers its token. A sign in that the
// gas service refuses is errs.ErrCredentialsRejected, and one that it holds for a
// human check is errs.ErrSignInBlocked.
func (s *APIClient) Authenticate(
	ctx context.Context,
	username string,
	password string,
) (string, error) {
	var response dto.AuthResponse

	resp, err := s.request(ctx).
		SetResult(&response).
		SetBody(map[string]string{
			"userName":      username,
			"password":      password,
			"recaptchaCode": "",
			"platform":      "web",
			"app":           "0.1.0",
		}).
		Post("/api/Users/Authenticate")
	if err != nil {
		return "", fmt.Errorf("%w: sending the sign in: %w", errs.ErrRequestFailed, err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
	case http.StatusBadRequest, http.StatusUnauthorized:
		return "", fmt.Errorf(
			"%w: the sign in returned [%d] %s",
			errs.ErrCredentialsRejected,
			resp.StatusCode(),
			failureBody(resp),
		)
	default:
		return "", statusFailure(resp)
	}

	switch {
	case response.RecaptchaRequested:
		return "", errs.ErrSignInBlocked
	case response.Code != 0:
		return "", fmt.Errorf(
			"%w: the sign in returned code %d",
			errs.ErrCredentialsRejected,
			response.Code,
		)
	case response.Token == "":
		return "", fmt.Errorf("%w: the sign in answered no token", errs.ErrRequestFailed)
	}

	return response.Token, nil
}

// GetCustomers retrieves the customers of the account that the token names.
func (s *APIClient) GetCustomers(ctx context.Context, token string) ([]dto.Customer, error) {
	var response dto.HomeResponse

	if err := s.postWithToken(ctx, token, "/api/Home", struct{}{}, &response); err != nil {
		return nil, fmt.Errorf("requesting the customers: %w", err)
	}

	if err := answerFailure(response.BaseResponse); err != nil {
		return nil, fmt.Errorf("reading the customers: %w", err)
	}

	return response.Customers, nil
}

// GetTransactions retrieves the transactions of one customer of the account in the
// period of the request.
func (s *APIClient) GetTransactions(
	ctx context.Context,
	token string,
	request dto.TransactionsRequest,
) ([]dto.Transaction, error) {
	var response dto.TransactionsResponse

	err := s.postWithToken(ctx, token, "/api/Customer/GetTransactions", request, &response)
	if err != nil {
		return nil, fmt.Errorf("requesting the transactions: %w", err)
	}

	if err = answerFailure(response.BaseResponse); err != nil {
		return nil, fmt.Errorf("reading the transactions: %w", err)
	}

	return response.Transactions, nil
}

// DownloadFile downloads a bill or a meter photo of the account that the token names.
// The path is the one that a transaction carries, relative to the origin of the gas
// service.
func (s *APIClient) DownloadFile(ctx context.Context, token string, path string) ([]byte, error) {
	resp, err := s.client.R().
		SetContext(ctx).
		SetHeader(headerToken, token).
		SetDoNotParseResponse(true).
		Get("/" + path)
	if err != nil {
		return nil, fmt.Errorf("%w: downloading a file: %w", errs.ErrRequestFailed, err)
	}

	content, err := readBounded(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("downloading a file: %w", err)
	}

	if isRejectedToken(resp) {
		return nil, errs.ErrNotConnected
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf(
			"%w: downloading a file returned [%d] %s",
			errs.ErrRequestFailed,
			resp.StatusCode(),
			cutFailure(string(content)),
		)
	}

	return content, nil
}

// readBounded reads the body and closes it. A body over maxResponseBody fails, so a
// broken answer cannot fill the memory of the hub.
func readBounded(body io.ReadCloser) ([]byte, error) {
	defer func() {
		_ = body.Close()
	}()

	content, err := io.ReadAll(io.LimitReader(body, maxResponseBody+1))
	if err != nil {
		return nil, fmt.Errorf("%w: reading the body: %w", errs.ErrRequestFailed, err)
	}

	if len(content) > maxResponseBody {
		return nil, fmt.Errorf("%w: the body passes 10 MiB", errs.ErrRequestFailed)
	}

	return content, nil
}

// postWithToken sends a POST that carries the token in the header of the gas service.
func (s *APIClient) postWithToken(
	ctx context.Context,
	token string,
	path string,
	body any,
	result any,
) error {
	resp, err := s.request(ctx).
		SetHeader(headerToken, token).
		SetBody(body).
		SetResult(result).
		Post(path)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrRequestFailed, err)
	}

	if isRejectedToken(resp) {
		return errs.ErrNotConnected
	}

	return statusFailure(resp)
}

// isRejectedToken reports an answer that names a token the gas service no longer accepts.
func isRejectedToken(resp *resty.Response) bool {
	return resp.StatusCode() == http.StatusUnauthorized
}

// request starts a request that decodes the answer as JSON, so the decode does not
// depend on the content type that the gas service names.
func (s *APIClient) request(ctx context.Context) *resty.Request {
	return s.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json;charset=UTF-8").
		SetForceResponseContentType("application/json")
}

// answerFailure answers the failure that a successful status carries in its body.
func answerFailure(response dto.BaseResponse) error {
	if response.Code == 0 {
		return nil
	}

	return fmt.Errorf(
		"%w: returned code %d %s",
		errs.ErrRequestFailed,
		response.Code,
		response.Message,
	)
}

func statusFailure(resp *resty.Response) error {
	if resp.StatusCode() == http.StatusOK {
		return nil
	}

	return fmt.Errorf(
		"%w: returned [%d] %s",
		errs.ErrRequestFailed,
		resp.StatusCode(),
		failureBody(resp),
	)
}

// failureBody answers the start of the body of a failed answer, so an error names
// what the gas service said. The answer of a failure carries no token.
func failureBody(resp *resty.Response) string {
	return cutFailure(resp.String())
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
