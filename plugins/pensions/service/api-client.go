package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"resty.dev/v3"

	"github.com/abgeo/maroid/plugins/pensions/config"
	"github.com/abgeo/maroid/plugins/pensions/dto"
)

// ErrRequestFailed is returned when an HTTP or API request fails.
var ErrRequestFailed = errors.New("request failed")

// APIClientService defines the interface for interacting with the Pensions API.
type APIClientService interface {
	SetAuthToken(token string)
	Authenticate(ctx context.Context, username string, password string) (string, error)
	GetParticipantInfo(ctx context.Context) (*dto.ParticipantInfoResponse, error)
	GetContributions(
		ctx context.Context,
		query dto.ContributionsRequest,
	) ([]dto.Contribution, error)
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
			"username":     username,
			"passwordOne":  password,
			"passwordTwo":  "",
			"languageCode": "ka-GE",
		}).
		Post("/v1/auth/participant-auth")
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

	if response.AccessToken == "" {
		return "", fmt.Errorf("%w: %s", ErrRequestFailed, response.Message)
	}

	return response.AccessToken, nil
}

// GetParticipantInfo retrieves participant information from the API.
func (s *APIClient) GetParticipantInfo(
	ctx context.Context,
) (*dto.ParticipantInfoResponse, error) {
	var response *dto.ParticipantInfoResponse

	resp, err := s.client.R().
		SetContext(ctx).
		SetResult(&response).
		Get("/v2/contributions/participant/get")
	if err != nil {
		return nil, fmt.Errorf("sending participant info request: %w", err)
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

// GetContributions retrieves contributions from the API based on the provided query.
func (s *APIClient) GetContributions(
	ctx context.Context,
	query dto.ContributionsRequest,
) ([]dto.Contribution, error) {
	var response *dto.PaginatedResponse[dto.Contribution]

	resp, err := s.client.R().
		SetContext(ctx).
		SetResult(&response).
		SetQueryParams(query.ToQueryParams()).
		Get("/v1/fbo/contributions")
	if err != nil {
		return nil, fmt.Errorf("sending contributions request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf(
			"%w: returned [%d] %v",
			ErrRequestFailed,
			resp.StatusCode(),
			resp.Error(),
		)
	}

	if response.Status != "success" || response.Code != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrRequestFailed, response.Message)
	}

	return response.Data.Result, nil
}
