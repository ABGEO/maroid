package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"resty.dev/v3"

	"github.com/abgeo/maroid/plugins/gwp/config"
	"github.com/abgeo/maroid/plugins/gwp/dto"
	"github.com/abgeo/maroid/plugins/gwp/errs"
)

// maxFailureBody bounds the part of the body of a failed answer that an error names.
const maxFailureBody = 512

// APIClientService defines the interface for interacting with the water service.
type APIClientService interface {
	Authenticate(ctx context.Context, username string, password string) (string, error)
	GetCustomers(ctx context.Context, token string) ([]dto.Customer, error)
	GetReadings(ctx context.Context, token string) ([]dto.ReadingResponse, error)
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
		SetTimeout(cfg.RequestTimeout)

	return &APIClient{
		client: client,
	}
}

// Authenticate signs in to the water account and answers its token. A sign in that
// the water service refuses is errs.ErrCredentialsRejected.
func (s *APIClient) Authenticate(
	ctx context.Context,
	username string,
	password string,
) (string, error) {
	var response dto.AuthResponse

	resp, err := s.request(ctx).
		SetResult(&response).
		SetBody(map[string]string{
			"username": username,
			"password": password,
		}).
		Put("/customer")
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

	if !response.Success {
		return "", errs.ErrCredentialsRejected
	}

	if response.Token == "" {
		return "", fmt.Errorf("%w: the sign in answered no token", errs.ErrRequestFailed)
	}

	return response.Token, nil
}

// GetCustomers retrieves the customers of the account that the token names.
func (s *APIClient) GetCustomers(ctx context.Context, token string) ([]dto.Customer, error) {
	var response dto.ListResponse[dto.Customer]

	if err := s.getWithToken(ctx, token, "/customer/ListCustomers", &response); err != nil {
		return nil, fmt.Errorf("requesting the customers: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf(
			"%w: reading the customers: %s",
			errs.ErrRequestFailed,
			response.Message,
		)
	}

	return response.Items, nil
}

// GetReadings retrieves the readings of every customer of the account that the token names.
func (s *APIClient) GetReadings(ctx context.Context, token string) ([]dto.ReadingResponse, error) {
	var response []dto.ReadingResponse

	if err := s.getWithToken(ctx, token, "/RecordDisplay", &response); err != nil {
		return nil, fmt.Errorf("requesting the readings: %w", err)
	}

	return response, nil
}

// getWithToken sends a GET that carries the token. A 401 is the only answer that
// names a token the water service no longer accepts, so it alone is
// errs.ErrNotConnected.
func (s *APIClient) getWithToken(ctx context.Context, token string, path string, result any) error {
	resp, err := s.request(ctx).
		SetAuthToken(token).
		SetResult(result).
		Get(path)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrRequestFailed, err)
	}

	if resp.StatusCode() == http.StatusUnauthorized {
		return errs.ErrNotConnected
	}

	return statusFailure(resp)
}

// request starts a request that decodes the answer as JSON, so the decode does not
// depend on the content type that the water service names.
func (s *APIClient) request(ctx context.Context) *resty.Request {
	return s.client.R().
		SetContext(ctx).
		SetForceResponseContentType("application/json")
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
// what the water service said. The answer of a failure carries no token.
func failureBody(resp *resty.Response) string {
	body := strings.TrimSpace(resp.String())
	if len(body) <= maxFailureBody {
		return body
	}

	cut := maxFailureBody
	for cut > 0 && !utf8.RuneStart(body[cut]) {
		cut--
	}

	return body[:cut]
}
