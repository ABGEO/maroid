package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"resty.dev/v3"

	"github.com/abgeo/maroid/plugins/gwp/config"
	"github.com/abgeo/maroid/plugins/gwp/dto"
)

// ErrRequestFailed is returned when an HTTP or API request fails.
var ErrRequestFailed = errors.New("request failed")

// APIClientService defines the interface for interacting with the GWP API.
type APIClientService interface {
	SetAuthToken(token string)
	Authenticate(ctx context.Context, username string, password string) (string, error)
	GetCustomers(ctx context.Context) (*dto.ListResponse[dto.Customer], error)
	GetReadings(ctx context.Context) ([]dto.ReadingResponse, error)
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
func (s *APIClient) Authenticate(
	ctx context.Context,
	username string,
	password string,
) (string, error) {
	var response dto.AuthResponse

	resp, err := s.client.R().
		SetContext(ctx).
		SetResult(&response).
		SetBody(map[string]string{
			"username": username,
			"password": password,
		}).
		Put("/customer")
	if err != nil {
		return "", fmt.Errorf("sending authentication request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf(
			"%w: returned [%d] %v",
			ErrRequestFailed,
			resp.StatusCode(),
			resp.Error(),
		)
	}

	if !response.Success {
		return "", fmt.Errorf("%w: %s", ErrRequestFailed, response.Message)
	}

	return response.Token, nil
}

// GetCustomers retrieves the list of customers from the API.
func (s *APIClient) GetCustomers(
	ctx context.Context,
) (*dto.ListResponse[dto.Customer], error) {
	var response *dto.ListResponse[dto.Customer]

	resp, err := s.client.R().
		SetContext(ctx).
		SetResult(&response).
		Get("/customer/ListCustomers")
	if err != nil {
		return nil, fmt.Errorf("sending readings request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf(
			"%w: returned [%d] %v",
			ErrRequestFailed,
			resp.StatusCode(),
			resp.Error(),
		)
	}

	if !response.Success {
		return nil, fmt.Errorf("%w: %s", ErrRequestFailed, response.Message)
	}

	return response, nil
}

// GetReadings retrieves the readings from the API.
func (s *APIClient) GetReadings(
	ctx context.Context,
) ([]dto.ReadingResponse, error) {
	var response []dto.ReadingResponse

	resp, err := s.client.R().
		SetContext(ctx).
		SetResult(&response).
		Get("/RecordDisplay")
	if err != nil {
		return nil, fmt.Errorf("sending readings request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf(
			"%w: returned [%d] %v",
			ErrRequestFailed,
			resp.StatusCode(),
			resp.Error(),
		)
	}

	return response, nil
}
