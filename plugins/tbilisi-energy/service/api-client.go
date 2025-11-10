package service

import (
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"resty.dev/v3"

	"github.com/abgeo/maroid/plugins/tbilisi-energy/config"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/dto"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/errs"
)

// APIClientService defines the interface for interacting with the Tbilisi Energy API.
type APIClientService interface {
	SetAuthToken(token string)
	Authenticate(username string, password string) (string, error)
	GetTransactions(body dto.TransactionsRequest) ([]dto.Transaction, error)
}

// APIClient implements APIClientService.
type APIClient struct {
	client *resty.Client
}

var _ APIClientService = (*APIClient)(nil)

// NewAPIClient creates a new APIClient.
func NewAPIClient(cfg *config.Config) *APIClient {
	client := resty.New().
		SetBaseURL(cfg.BaseURL).
		SetAuthScheme("").
		SetHeaderAuthorizationKey("token").
		SetError(map[string]any{}).
		SetHeader("Accept", "application/json")

	return &APIClient{
		client: client,
	}
}

// SetAuthToken sets the authentication token for subsequent API requests.
func (s *APIClient) SetAuthToken(token string) {
	s.client.SetAuthToken(token)
}

// Authenticate authenticates the user and returns an authentication token.
func (s *APIClient) Authenticate(username, password string) (string, error) {
	var response dto.AuthResponse

	resp, err := s.client.R().
		SetResult(&response).
		SetBody(map[string]string{
			"userName":      username,
			"password":      password,
			"recaptchaCode": "",
			"platform":      "web",
			"app":           "0.1.0",
		}).
		Post("/Users/Authenticate")
	if err != nil {
		return "", fmt.Errorf("failed to send authentication request: %w", err)
	}

	if err = extractHTTPError(resp); err != nil {
		return "", err
	}

	if response.Code != 0 {
		return "", fmt.Errorf("%w: [%d] %s", errs.ErrRequestFailed, response.Code, response.Message)
	}

	return response.Token, nil
}

// GetTransactions retrieves transactions based on the provided request parameters.
func (s *APIClient) GetTransactions(body dto.TransactionsRequest) ([]dto.Transaction, error) {
	var response dto.TransactionsResponse

	resp, err := s.client.R().
		SetResult(&response).
		SetBody(body).
		Post("/Customer/GetTransactions")
	if err != nil {
		return nil, fmt.Errorf("failed to send transactions request: %w", err)
	}

	if err = extractHTTPError(resp); err != nil {
		return nil, err
	}

	if response.Code != 0 {
		return nil, fmt.Errorf(
			"%w: [%d] %s",
			errs.ErrRequestFailed,
			response.Code,
			response.Message,
		)
	}

	return response.Transactions, nil
}

func extractHTTPError(response *resty.Response) error {
	if response.StatusCode() < http.StatusBadRequest {
		return nil
	}

	errMap, ok := response.Error().(*map[string]any)
	if !ok {
		return fmt.Errorf("%w: unknown error format %v", errs.ErrRequestFailed, response.Error())
	}

	if err := tryParseErrorResponse(errMap); err != nil {
		return err
	}

	if err := tryParseBaseResponse(errMap); err != nil {
		return err
	}

	return fmt.Errorf("%w: unknown error format %v", errs.ErrRequestFailed, errMap)
}

func tryParseErrorResponse(errMap *map[string]any) error {
	var errResp dto.ErrorResponse

	if err := mapstructure.Decode(errMap, &errResp); err != nil {
		return nil //nolint:nilerr
	}

	if errResp.Title == "" {
		return nil
	}

	return fmt.Errorf(
		"%w: [%d] %s %v",
		errs.ErrRequestFailed,
		errResp.Status,
		errResp.Title,
		errResp.Errors,
	)
}

func tryParseBaseResponse(errMap *map[string]any) error {
	var baseResp dto.BaseResponse

	if err := mapstructure.Decode(errMap, &baseResp); err != nil {
		return nil //nolint:nilerr
	}

	if baseResp.Message == "" {
		return nil
	}

	return fmt.Errorf("%w: [%d] %s", errs.ErrRequestFailed, baseResp.Code, baseResp.Message)
}
