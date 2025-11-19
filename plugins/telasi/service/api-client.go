package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"resty.dev/v3"

	"github.com/abgeo/maroid/plugins/telasi/config"
	"github.com/abgeo/maroid/plugins/telasi/dto"
)

// ErrRequestFailed is returned when an HTTP or API request fails.
var ErrRequestFailed = errors.New("request failed")

// APIClientService defines the interface for interacting with the Telasi API.
type APIClientService interface {
	SetAuthToken(token string)
	Authenticate(ctx context.Context, email string, password string) (string, error)
	GetCustomers(ctx context.Context) ([]dto.CustomerResponse, error)
	GetBillingItems(ctx context.Context, body dto.BillingItemsRequest) ([]dto.BillingItem, error)
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
	email string,
	password string,
) (string, error) {
	var response dto.AuthResponse

	resp, err := s.client.R().
		SetContext(ctx).
		SetResult(&response).
		SetBody(map[string]string{
			"email":    email,
			"password": password,
		}).
		Post("/telasiCustomers/login")
	if err != nil {
		return "", fmt.Errorf("failed to send authentication request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("%w: returned %v", ErrRequestFailed, resp.Error())
	}

	return response.Token, nil
}

// GetCustomers retrieves the list of customers associated with the authenticated user.
func (s *APIClient) GetCustomers(ctx context.Context) ([]dto.CustomerResponse, error) {
	var response []dto.CustomerResponse

	resp, err := s.client.R().
		SetContext(ctx).
		SetResult(&response).
		Post("/telasiCustomers/info/getCustomers")
	if err != nil {
		return nil, fmt.Errorf("failed to send customers request request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("%w: returned %v", ErrRequestFailed, resp.Error())
	}

	return response, nil
}

// GetBillingItems retrieves billing items based on the provided request body.
func (s *APIClient) GetBillingItems(
	ctx context.Context,
	body dto.BillingItemsRequest,
) ([]dto.BillingItem, error) {
	var response dto.ListResponse[dto.BillingItem]

	resp, err := s.client.R().
		SetContext(ctx).
		SetResult(&response).
		SetBody(body).
		Post("/telasiCustomers/info/getBillingItems")
	if err != nil {
		return nil, fmt.Errorf("failed to send billing items request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("%w: returned %v", ErrRequestFailed, resp.Error())
	}

	return response.List.Items, nil
}
