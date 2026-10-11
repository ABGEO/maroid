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
	"github.com/abgeo/maroid/plugins/parking/config"
	"github.com/abgeo/maroid/plugins/parking/dto"
	"github.com/abgeo/maroid/plugins/parking/errs"
	"github.com/abgeo/maroid/plugins/parking/model"
)

const (
	// SecretSession names the plugin secret that holds the session token.
	SecretSession = "session"

	targetPhone          = "PHONE_VERIFICATION"
	statusTwoFactor      = "REQUIRES_TWO_FA"
	statusSuccess        = "SUCCESS"
	headerSessionToken   = "session-token"
	pathSignIn           = "/v3/session/mobile"
	pathCodeRequest      = "/um/v3/user/two-Fa/send"
	pathCodeSubmit       = "/um/v3/user/two-Fa/submit"
	pathVehicles         = "/parking/person/vehicles"
	pathPerson           = "/parking/person/check"
	pathParking          = "/parking"
	pathParkingByID      = "/parking/{id}"
	pathParkingPlace     = "/parking/place/zone/{zone}/place/{number}"
	pathParkingForScreen = "/business-parking/parking/place/for-screen"
)

// APIClientService defines the interface for interacting with the Parking API.
type APIClientService interface {
	SignIn(ctx context.Context, username, password string) (*dto.SignInResult, error)
	RequestCode(ctx context.Context, signInID string) (*dto.CodeRequestResult, error)
	SubmitCode(ctx context.Context, signInID, code string) (string, error)
	ListVehicles(ctx context.Context) ([]model.Vehicle, error)
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
		vehicleID int,
	) (*dto.ParkingSession, error)
	GetPerson(ctx context.Context) (*dto.Person, error)
	GetActiveSession(ctx context.Context) (*dto.ActiveSession, error)
	StopParking(ctx context.Context, id int) (*dto.ParkingSession, error)
}

// APIClient implements APIClientService.
type APIClient struct {
	secrets *pluginapi.PluginSecrets
	client  *resty.Client
}

var _ APIClientService = (*APIClient)(nil)

// NewAPIClient creates a new APIClient. It reads the session token of the acting
// user from the plugin secrets.
func NewAPIClient(cfg *config.Config, secrets *pluginapi.PluginSecrets) *APIClient {
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
		secrets: secrets,
		client:  client,
	}
}

// SignIn sends the credentials. It carries no session token.
func (s *APIClient) SignIn(
	ctx context.Context,
	username, password string,
) (*dto.SignInResult, error) {
	var response dto.APIResponse[dto.SignInResult]

	resp, err := s.client.R().
		SetContext(ctx).
		SetResult(&response).
		SetBody(dto.APIRequest[dto.SignInData]{
			Data: dto.SignInData{Username: username, Password: password},
		}).
		Post(pathSignIn)
	if err != nil {
		return nil, fmt.Errorf("%w: sending the sign in: %w", errs.ErrRequestFailed, err)
	}

	if isRejection(resp.StatusCode()) {
		return nil, fmt.Errorf("%w: returned [%d]", errs.ErrCredentialsRejected, resp.StatusCode())
	}

	if err = requireSuccess(resp); err != nil {
		return nil, err
	}

	return &response.Result.Data, nil
}

// RequestCode asks the parking service to send the code of the sign in by SMS.
func (s *APIClient) RequestCode(
	ctx context.Context,
	signInID string,
) (*dto.CodeRequestResult, error) {
	var response dto.APIResponse[dto.CodeRequestResult]

	resp, err := s.client.R().
		SetContext(ctx).
		SetResult(&response).
		SetBody(dto.APIRequest[dto.CodeRequestData]{
			Data: dto.CodeRequestData{TargetType: targetPhone, UUID: signInID},
		}).
		Post(pathCodeRequest)
	if err != nil {
		return nil, fmt.Errorf("%w: sending the code request: %w", errs.ErrRequestFailed, err)
	}

	if err = requireSuccess(resp); err != nil {
		return nil, err
	}

	return &response.Result.Data, nil
}

// SubmitCode sends the code of the sign in, and returns the session token that the
// parking service gives for it.
func (s *APIClient) SubmitCode(ctx context.Context, signInID, code string) (string, error) {
	var response dto.APIResponse[dto.CodeSubmitResult]

	resp, err := s.client.R().
		SetContext(ctx).
		SetResult(&response).
		SetBody(dto.APIRequest[dto.CodeSubmitData]{
			Data: dto.CodeSubmitData{TargetType: targetPhone, UUID: signInID, Token: code},
		}).
		Post(pathCodeSubmit)
	if err != nil {
		return "", fmt.Errorf("%w: sending the code: %w", errs.ErrRequestFailed, err)
	}

	if resp.StatusCode() >= http.StatusBadRequest &&
		resp.StatusCode() < http.StatusInternalServerError {
		return "", fmt.Errorf("%w: returned [%d]", errs.ErrCodeRejected, resp.StatusCode())
	}

	if err = requireSuccess(resp); err != nil {
		return "", err
	}

	if response.Result.Data.Status != statusSuccess {
		return "", fmt.Errorf(
			"%w: the status is %q",
			errs.ErrCodeRejected,
			response.Result.Data.Status,
		)
	}

	token := resp.Header().Get(headerSessionToken)
	if token == "" {
		return "", fmt.Errorf("%w: the answer carries no session token", errs.ErrRequestFailed)
	}

	return token, nil
}

// ListVehicles retrieves the vehicles of the parking account.
func (s *APIClient) ListVehicles(ctx context.Context) ([]model.Vehicle, error) {
	var response dto.APIResponse[[]dto.Vehicle]

	err := s.send(ctx, "reading the vehicles", func(req *resty.Request) (*resty.Response, error) {
		return req.SetResult(&response).Get(pathVehicles)
	})
	if err != nil {
		return nil, err
	}

	vehicles := make([]model.Vehicle, 0, len(response.Result.Data))
	for _, vehicle := range response.Result.Data {
		vehicles = append(vehicles, vehicle.ToModel())
	}

	return vehicles, nil
}

// GetParkingLots retrieves parking lots within the given bounding box.
func (s *APIClient) GetParkingLots(
	ctx context.Context,
	left, right, top, bottom float64,
) ([]dto.ParkingLot, error) {
	var response dto.APIResponse[[]dto.ParkingLot]

	err := s.send(
		ctx,
		"reading the parking lots",
		func(req *resty.Request) (*resty.Response, error) {
			return req.
				SetResult(&response).
				SetQueryParams(map[string]string{
					"left":   fmt.Sprintf("%.14f", left),
					"right":  fmt.Sprintf("%.14f", right),
					"top":    fmt.Sprintf("%.14f", top),
					"bottom": fmt.Sprintf("%.14f", bottom),
				}).
				Get(pathParkingForScreen)
		},
	)
	if err != nil {
		return nil, err
	}

	return response.Result.Data, nil
}

// GetParkingPlace retrieves a specific parking place by zone and number.
func (s *APIClient) GetParkingPlace(
	ctx context.Context,
	zone, number string,
) (*dto.ParkingPlace, error) {
	var response dto.APIResponse[dto.ParkingPlace]

	err := s.send(
		ctx,
		"reading the parking place",
		func(req *resty.Request) (*resty.Response, error) {
			return req.
				SetResult(&response).
				SetPathParams(map[string]string{"zone": zone, "number": number}).
				Get(pathParkingPlace)
		},
	)
	if _, isStatus := errors.AsType[*StatusError](err); isStatus {
		return nil, errs.ErrParkingPlaceNotFound
	}

	if err != nil {
		return nil, err
	}

	return &response.Result.Data, nil
}

// StartParking starts a parking session for the vehicle.
func (s *APIClient) StartParking(
	ctx context.Context,
	placeNo, parkingType string,
	vehicleID int,
) (*dto.ParkingSession, error) {
	var response dto.APIResponse[dto.ParkingSession]

	body := dto.APIRequest[dto.StartParkingData]{
		Data: dto.StartParkingData{
			PlaceNo:   placeNo,
			VehicleID: vehicleID,
			Type:      parkingType,
		},
	}

	err := s.send(
		ctx,
		"starting the parking session",
		func(req *resty.Request) (*resty.Response, error) {
			return req.SetResult(&response).SetBody(body).Post(pathParking)
		},
	)
	if err != nil {
		return nil, err
	}

	return &response.Result.Data, nil
}

// GetActiveSession retrieves the current active parking session, if any.
// Returns nil when no session is active.
func (s *APIClient) GetActiveSession(
	ctx context.Context,
) (*dto.ActiveSession, error) {
	var response dto.APIResponse[*dto.ActiveSession]

	err := s.send(
		ctx,
		"reading the active session",
		func(req *resty.Request) (*resty.Response, error) {
			return req.SetResult(&response).Get(pathParking)
		},
	)
	if err != nil {
		return nil, err
	}

	return response.Result.Data, nil
}

// StopParking stops an active parking session by ID.
func (s *APIClient) StopParking(
	ctx context.Context,
	id int,
) (*dto.ParkingSession, error) {
	var response dto.APIResponse[dto.ParkingSession]

	err := s.send(
		ctx,
		"stopping the parking session",
		func(req *resty.Request) (*resty.Response, error) {
			return req.
				SetResult(&response).
				SetPathParam("id", strconv.Itoa(id)).
				Delete(pathParkingByID)
		},
	)
	if err != nil {
		return nil, err
	}

	return &response.Result.Data, nil
}

// GetPerson retrieves the authenticated person's profile.
func (s *APIClient) GetPerson(
	ctx context.Context,
) (*dto.Person, error) {
	var response dto.APIResponse[dto.Person]

	err := s.send(ctx, "reading the person", func(req *resty.Request) (*resty.Response, error) {
		return req.SetResult(&response).Get(pathPerson)
	})
	if err != nil {
		return nil, err
	}

	return &response.Result.Data, nil
}

// send runs one call with the session token of the acting user. A user with no token
// gets errs.ErrNotConnected with no call. A 401 is the only sign that the parking service
// ended the token, so send removes the token and answers errs.ErrNotConnected.
func (s *APIClient) send(
	ctx context.Context,
	action string,
	call func(*resty.Request) (*resty.Response, error),
) error {
	token, err := s.secrets.Get(ctx, pluginapi.SecretScopeUser, SecretSession)
	if errors.Is(err, pluginapi.ErrSecretAbsent) {
		return errs.ErrNotConnected
	}

	if err != nil {
		return fmt.Errorf("reading the session token: %w", err)
	}

	resp, err := call(s.client.R().SetContext(ctx).SetAuthToken(token))
	if err != nil {
		return fmt.Errorf("%w: %s: %w", errs.ErrRequestFailed, action, err)
	}

	if resp.StatusCode() == http.StatusUnauthorized {
		if err = s.secrets.Delete(ctx, pluginapi.SecretScopeUser, SecretSession); err != nil {
			return fmt.Errorf("removing the rejected session token: %w", err)
		}

		return errs.ErrNotConnected
	}

	if err = requireSuccess(resp); err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	return nil
}

func isRejection(status int) bool {
	switch status {
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		return true
	default:
		return false
	}
}

// StatusError reports an answer of the parking service whose status is not 200. It
// matches errs.ErrRequestFailed.
type StatusError struct {
	Status int
}

// Error names the status.
func (e *StatusError) Error() string {
	return fmt.Sprintf("%s: returned [%d]", errs.ErrRequestFailed, e.Status)
}

// Is reports whether target is errs.ErrRequestFailed.
func (e *StatusError) Is(target error) bool {
	return target == errs.ErrRequestFailed
}

func requireSuccess(resp *resty.Response) error {
	if resp.StatusCode() != http.StatusOK {
		return &StatusError{Status: resp.StatusCode()}
	}

	return nil
}
