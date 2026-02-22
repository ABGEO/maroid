// Package service provides API client implementations for the parking plugin.
package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"resty.dev/v3"

	"github.com/abgeo/maroid/plugins/parking/config"
	"github.com/abgeo/maroid/plugins/parking/dto"
)

var (
	// ErrRequestFailed is returned when an HTTP or API request fails.
	ErrRequestFailed = errors.New("request failed")
	// ErrParkingPlaceNotFound is returned when a parking place does not exist.
	ErrParkingPlaceNotFound = errors.New("parking place not found")
)

// APIClientService defines the interface for interacting with the Parking API.
type APIClientService interface {
	GetParkingLots(
		ctx context.Context,
		left, right, top, bottom float64,
	) ([]dto.ParkingLot, error)
	GetParkingPlace(
		ctx context.Context,
		zone, number string,
	) (*dto.ParkingPlace, error)
	StartParking(
		ctx context.Context,
		placeNo, parkingType string,
	) (*dto.ParkingSession, error)
	GetPerson(ctx context.Context) (*dto.Person, error)
	GetActiveSession(ctx context.Context) (*dto.ActiveSession, error)
	StopParking(ctx context.Context, id int) (*dto.ParkingSession, error)
}

// APIClient implements APIClientService.
type APIClient struct {
	cfg    *config.Config
	client *resty.Client
}

var _ APIClientService = (*APIClient)(nil)

// NewAPIClient creates a new APIClient.
func NewAPIClient(cfg *config.Config) *APIClient {
	client := resty.New().
		SetBaseURL(cfg.BaseURL).
		SetAuthToken(cfg.AuthToken).
		SetError(map[string]any{}).
		SetHeaders(map[string]string{
			"Accept":       "*/*",
			"Content-Type": "application/json",
			"User-Agent":   "ttc-park/1.0",
			"ttl":          "2592000000",
		})

	return &APIClient{
		cfg:    cfg,
		client: client,
	}
}

// GetParkingLots retrieves parking lots within the given bounding box.
func (s *APIClient) GetParkingLots(
	ctx context.Context,
	left, right, top, bottom float64,
) ([]dto.ParkingLot, error) {
	var response dto.APIResponse[[]dto.ParkingLot]

	resp, err := s.client.R().
		SetContext(ctx).
		SetResult(&response).
		SetQueryParams(map[string]string{
			"left":   fmt.Sprintf("%.14f", left),
			"right":  fmt.Sprintf("%.14f", right),
			"top":    fmt.Sprintf("%.14f", top),
			"bottom": fmt.Sprintf("%.14f", bottom),
		}).
		Get("/business-parking/parking/place/for-screen")
	if err != nil {
		return nil, fmt.Errorf(
			"failed to send parking lots request: %w", err,
		)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf(
			"%w: returned [%d] %v",
			ErrRequestFailed,
			resp.StatusCode(),
			resp.Error(),
		)
	}

	return response.Result.Data, nil
}

// GetParkingPlace retrieves a specific parking place by zone and number.
func (s *APIClient) GetParkingPlace(
	ctx context.Context,
	zone, number string,
) (*dto.ParkingPlace, error) {
	var response dto.APIResponse[dto.ParkingPlace]

	resp, err := s.client.R().
		SetContext(ctx).
		SetResult(&response).
		SetPathParams(map[string]string{
			"zone":   zone,
			"number": number,
		}).
		Get("/parking/place/zone/{zone}/place/{number}")
	if err != nil {
		return nil, fmt.Errorf(
			"failed to send parking place request: %w", err,
		)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, ErrParkingPlaceNotFound
	}

	return &response.Result.Data, nil
}

// StartParking starts a parking session for the configured vehicle.
func (s *APIClient) StartParking(
	ctx context.Context,
	placeNo, parkingType string,
) (*dto.ParkingSession, error) {
	var response dto.APIResponse[dto.ParkingSession]

	body := dto.APIRequest[dto.StartParkingData]{
		Data: dto.StartParkingData{
			PlaceNo:   placeNo,
			VehicleID: s.cfg.VehicleID,
			Type:      parkingType,
		},
	}

	resp, err := s.client.R().
		SetContext(ctx).
		SetResult(&response).
		SetBody(body).
		Post("/parking")
	if err != nil {
		return nil, fmt.Errorf("failed to send start parking request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf(
			"%w: returned [%d] %v",
			ErrRequestFailed,
			resp.StatusCode(),
			resp.Error(),
		)
	}

	return &response.Result.Data, nil
}

// GetActiveSession retrieves the current active parking session, if any.
// Returns nil when no session is active.
func (s *APIClient) GetActiveSession(
	ctx context.Context,
) (*dto.ActiveSession, error) {
	var response dto.APIResponse[*dto.ActiveSession]

	resp, err := s.client.R().
		SetContext(ctx).
		SetResult(&response).
		Get("/parking")
	if err != nil {
		return nil, fmt.Errorf(
			"failed to send active session request: %w", err,
		)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf(
			"%w: returned [%d] %v",
			ErrRequestFailed,
			resp.StatusCode(),
			resp.Error(),
		)
	}

	return response.Result.Data, nil
}

// StopParking stops an active parking session by ID.
func (s *APIClient) StopParking(
	ctx context.Context,
	id int,
) (*dto.ParkingSession, error) {
	var response dto.APIResponse[dto.ParkingSession]

	resp, err := s.client.R().
		SetContext(ctx).
		SetResult(&response).
		SetPathParam("id", strconv.Itoa(id)).
		Delete("/parking/{id}")
	if err != nil {
		return nil, fmt.Errorf(
			"failed to send stop parking request: %w", err,
		)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf(
			"%w: returned [%d] %v",
			ErrRequestFailed,
			resp.StatusCode(),
			resp.Error(),
		)
	}

	return &response.Result.Data, nil
}

// GetPerson retrieves the authenticated person's profile.
func (s *APIClient) GetPerson(
	ctx context.Context,
) (*dto.Person, error) {
	var response dto.APIResponse[dto.Person]

	resp, err := s.client.R().
		SetContext(ctx).
		SetResult(&response).
		Get("/parking/person/check")
	if err != nil {
		return nil, fmt.Errorf("failed to send person request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf(
			"%w: returned [%d] %v",
			ErrRequestFailed,
			resp.StatusCode(),
			resp.Error(),
		)
	}

	return &response.Result.Data, nil
}
