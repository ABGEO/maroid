// Package service provides API client implementations for the parking plugin.
package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"resty.dev/v3"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
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
	settings *pluginapi.PluginSettings
	client   *resty.Client
}

var _ APIClientService = (*APIClient)(nil)

// NewAPIClient creates a new APIClient.
func NewAPIClient(cfg *config.Config, settings *pluginapi.PluginSettings) *APIClient {
	client := resty.New().
		SetBaseURL(cfg.BaseURL).
		SetError(map[string]any{}).
		SetHeaders(map[string]string{
			"Accept":       "*/*",
			"Content-Type": "application/json",
			"User-Agent":   "ttc-park/1.0",
			"ttl":          "2592000000",
		})

	return &APIClient{
		settings: settings,
		client:   client,
	}
}

// GetParkingLots retrieves parking lots within the given bounding box.
func (s *APIClient) GetParkingLots(
	ctx context.Context,
	left, right, top, bottom float64,
) ([]dto.ParkingLot, error) {
	var response dto.APIResponse[[]dto.ParkingLot]

	req, _, err := s.authorized(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := req.
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
			"sending parking lots request: %w", err,
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

	req, _, err := s.authorized(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := req.
		SetResult(&response).
		SetPathParams(map[string]string{
			"zone":   zone,
			"number": number,
		}).
		Get("/parking/place/zone/{zone}/place/{number}")
	if err != nil {
		return nil, fmt.Errorf(
			"sending parking place request: %w", err,
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

	req, userSettings, err := s.authorized(ctx)
	if err != nil {
		return nil, err
	}

	vehicleID, err := strconv.Atoi(userSettings.VehicleID)
	if err != nil {
		return nil, fmt.Errorf("reading the vehicle identifier: %w", err)
	}

	body := dto.APIRequest[dto.StartParkingData]{
		Data: dto.StartParkingData{
			PlaceNo:   placeNo,
			VehicleID: vehicleID,
			Type:      parkingType,
		},
	}

	resp, err := req.
		SetResult(&response).
		SetBody(body).
		Post("/parking")
	if err != nil {
		return nil, fmt.Errorf("sending start parking request: %w", err)
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

	req, _, err := s.authorized(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := req.
		SetResult(&response).
		Get("/parking")
	if err != nil {
		return nil, fmt.Errorf(
			"sending active session request: %w", err,
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

	req, _, err := s.authorized(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := req.
		SetResult(&response).
		SetPathParam("id", strconv.Itoa(id)).
		Delete("/parking/{id}")
	if err != nil {
		return nil, fmt.Errorf(
			"sending stop parking request: %w", err,
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

	req, _, err := s.authorized(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := req.
		SetResult(&response).
		Get("/parking/person/check")
	if err != nil {
		return nil, fmt.Errorf("sending person request: %w", err)
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

// authorized builds a request that carries the credential of the acting user, and
// returns the settings that produced it. See PSET-FR-012.
// It fails with pluginapi.ErrSettingsAbsent when the acting user stored no credential.
func (s *APIClient) authorized(
	ctx context.Context,
) (*resty.Request, *config.UserSettings, error) {
	values, err := s.settings.Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("reading the settings of the acting user: %w", err)
	}

	userSettings := new(config.UserSettings)
	if err = pluginconfig.DecodeAndValidateSettings(values, userSettings); err != nil {
		return nil, nil, fmt.Errorf("decoding the settings of the acting user: %w", err)
	}

	return s.client.R().SetContext(ctx).SetAuthToken(userSettings.AuthToken), userSettings, nil
}
