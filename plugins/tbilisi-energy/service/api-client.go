package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"resty.dev/v3"

	"github.com/abgeo/maroid/plugins/tbilisi-energy/config"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/dto"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/errs"
)

// APIClientService defines the interface for interacting with the Tbilisi Energy API.
type APIClientService interface {
	SetAuthToken(token string)
	Authenticate(ctx context.Context, username string, password string) (string, error)
	GetTransactions(ctx context.Context, body dto.TransactionsRequest) ([]dto.Transaction, error)
	DownloadFile(ctx context.Context, fileURL string) ([]byte, error)
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
func (s *APIClient) Authenticate(ctx context.Context, username, password string) (string, error) {
	var response dto.AuthResponse

	resp, err := s.client.R().
		SetContext(ctx).
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
		return "", fmt.Errorf("sending authentication request: %w", err)
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
func (s *APIClient) GetTransactions(
	ctx context.Context,
	body dto.TransactionsRequest,
) ([]dto.Transaction, error) {
	var response dto.TransactionsResponse

	resp, err := s.client.R().
		SetContext(ctx).
		SetResult(&response).
		SetBody(body).
		Post("/Customer/GetTransactions")
	if err != nil {
		return nil, fmt.Errorf("sending transactions request: %w", err)
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

// DownloadFile downloads a file from the specified URL.
func (s *APIClient) DownloadFile(ctx context.Context, fileURL string) ([]byte, error) {
	fileURL = strings.TrimPrefix(fileURL, "api/")

	resp, err := s.client.R().
		SetContext(ctx).
		SetDoNotParseResponse(true).
		Get(fileURL)
	if err != nil {
		return nil, fmt.Errorf("downloading file: %w", err)
	}

	if err = extractHTTPError(resp); err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	return data, nil
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
