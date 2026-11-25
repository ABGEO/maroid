package service

import (
	"context"
	"crypto/md5" //nolint:gosec // G501: the car log service demands the MD5 of the password.
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"resty.dev/v3"

	"github.com/abgeo/maroid/plugins/drivvo/config"
	"github.com/abgeo/maroid/plugins/drivvo/dto"
	"github.com/abgeo/maroid/plugins/drivvo/errs"
	"github.com/abgeo/maroid/plugins/drivvo/model"
)

const (
	// maxFailureBody bounds the part of the body of a failed answer that an error names.
	maxFailureBody = 512

	// maxResponseBody bounds an answer of the service. The refuels of a vehicle with some
	// hundreds of entries are some hundreds of KiB.
	maxResponseBody = 20 << 20

	headerToken = "X-Token"

	pathLogIn         = "/autenticacao/login"
	pathVehicles      = "/veiculo/web"
	pathSettings      = "/configuracao"
	pathRefuels       = "/veiculo/%d/abastecimento/web"
	pathMaintenances  = "/veiculo/%d/servico/completo"
	pathExpenses      = "/veiculo/%d/despesa/completo"
	pathIncomes       = "/veiculo/%d/receita/completo"
	pathOdometers     = "/veiculo/%d/leitura/web"
	pathReminders     = "/veiculo/%d/lembrete/web"
	platformWeb       = "Web"
	mediaJSON         = "application/json"
	headerPlatform    = "App-Platform"
	headerVersion     = "App-Version"
	headerVersionName = "App-Version-Name"
)

// APIClient calls the car log service as its web site does.
type APIClient struct {
	client *resty.Client
	now    func() time.Time
}

// NewAPIClient creates a new APIClient.
func NewAPIClient(cfg *config.Config, now func() time.Time) *APIClient {
	client := resty.New().
		SetBaseURL(cfg.BaseURL).
		SetHeader("Accept", mediaJSON).
		SetHeader(headerPlatform, platformWeb).
		SetHeader(headerVersion, cfg.AppVersion).
		SetHeader(headerVersionName, cfg.AppVersionName).
		SetTimeout(cfg.RequestTimeout).
		SetResponseBodyLimit(maxResponseBody)

	return &APIClient{client: client, now: now}
}

// SignIn signs in with the email address and the password, and answers the session.
func (c *APIClient) SignIn(ctx context.Context, email, password string) (model.Session, error) {
	digest := md5.Sum([]byte(strings.TrimSpace(password))) //nolint:gosec // G401: see the import.

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", mediaJSON).
		SetBody(dto.LogInRequest{
			Email:  email,
			Senha:  hex.EncodeToString(digest[:]),
			Idioma: dto.LanguageEnglish,
		}).
		Post(pathLogIn)
	if err != nil {
		return model.Session{}, fmt.Errorf(
			"%w: sending the sign in: %w",
			errs.ErrRequestFailed,
			err,
		)
	}

	if err = logInFailure(resp); err != nil {
		return model.Session{}, err
	}

	var answer dto.LogInAnswer
	if err = json.Unmarshal(resp.Bytes(), &answer); err != nil {
		return model.Session{}, fmt.Errorf(
			"%w: reading the answer of the sign in: %w",
			errs.ErrRequestFailed,
			err,
		)
	}

	if answer.AlterarSenha {
		return model.Session{}, errs.ErrNewPasswordRequired
	}

	if answer.Token == "" {
		return model.Session{}, fmt.Errorf(
			"%w: the sign in answered no token",
			errs.ErrRequestFailed,
		)
	}

	return model.Session{Token: answer.Token, ExpiresAt: answer.Expiry(c.now()).UTC()}, nil
}

// Vehicles answers every vehicle of the account.
func (c *APIClient) Vehicles(ctx context.Context, token string) ([]model.Vehicle, error) {
	var answer []dto.Vehicle
	if err := c.read(ctx, token, pathVehicles, &answer); err != nil {
		return nil, err
	}

	vehicles := make([]model.Vehicle, 0, len(answer))

	for _, item := range answer {
		vehicle, err := item.Read()
		if err != nil {
			return nil, fmt.Errorf("reading the vehicles: %w", err)
		}

		vehicles = append(vehicles, vehicle)
	}

	return vehicles, nil
}

// Settings answers the car log settings, the first item of the answer.
func (c *APIClient) Settings(ctx context.Context, token string) (model.CarLogSettings, error) {
	var answer []dto.Settings
	if err := c.read(ctx, token, pathSettings, &answer); err != nil {
		return model.CarLogSettings{}, err
	}

	if len(answer) == 0 {
		return model.CarLogSettings{}, fmt.Errorf(
			"%w: %s answered no item",
			errs.ErrRequestFailed,
			pathSettings,
		)
	}

	settings, err := answer[0].Read()
	if err != nil {
		return model.CarLogSettings{}, fmt.Errorf("reading the settings: %w", err)
	}

	return settings, nil
}

// ReferenceItems answers the items of every list of reference items.
func (c *APIClient) ReferenceItems(
	ctx context.Context,
	token string,
) ([]model.ReferenceItem, error) {
	var items []model.ReferenceItem

	for _, list := range dto.ReferenceLists() {
		var raw json.RawMessage
		if err := c.read(ctx, token, list.Path, &raw); err != nil {
			return nil, err
		}

		read, err := dto.ReadReferenceItems(raw, list)
		if err != nil {
			return nil, fmt.Errorf("reading the reference items: %w", err)
		}

		items = append(items, read...)
	}

	return items, nil
}

// VehicleEntries answers every log entry of one vehicle: the refuels, the maintenances, the
// expenses, the incomes, and the odometer entries.
func (c *APIClient) VehicleEntries(
	ctx context.Context,
	token string,
	vehicle int32,
) ([]model.LogEntry, error) {
	var entries []model.LogEntry

	readers := []struct {
		path string
		read func(context.Context, *APIClient, string, string, int32) ([]model.LogEntry, error)
	}{
		{pathRefuels, readEntries[dto.Refuel]},
		{pathMaintenances, readEntries[dto.Maintenance]},
		{pathExpenses, readEntries[dto.Expense]},
		{pathIncomes, readEntries[dto.Income]},
		{pathOdometers, readEntries[dto.OdometerEntry]},
	}

	for _, reader := range readers {
		read, err := reader.read(ctx, c, token, reader.path, vehicle)
		if err != nil {
			return nil, err
		}

		entries = append(entries, read...)
	}

	return entries, nil
}

// Reminders answers every reminder of one vehicle.
func (c *APIClient) Reminders(
	ctx context.Context,
	token string,
	vehicle int32,
) ([]model.Reminder, error) {
	var answer []dto.Reminder
	if err := c.read(ctx, token, fmt.Sprintf(pathReminders, vehicle), &answer); err != nil {
		return nil, err
	}

	reminders := make([]model.Reminder, 0, len(answer))

	for _, item := range answer {
		reminder, err := item.Read()
		if err != nil {
			return nil, fmt.Errorf("reading the reminders of vehicle %d: %w", vehicle, err)
		}

		reminder.VehicleSource = vehicle
		reminders = append(reminders, reminder)
	}

	return reminders, nil
}

type entryReader interface {
	Read() (model.LogEntry, error)
}

func readEntries[T entryReader](
	ctx context.Context,
	client *APIClient,
	token string,
	pathFormat string,
	vehicle int32,
) ([]model.LogEntry, error) {
	var answer []T
	if err := client.read(ctx, token, fmt.Sprintf(pathFormat, vehicle), &answer); err != nil {
		return nil, err
	}

	entries := make([]model.LogEntry, 0, len(answer))

	for _, item := range answer {
		entry, err := item.Read()
		if err != nil {
			return nil, fmt.Errorf("reading the entries of vehicle %d: %w", vehicle, err)
		}

		entry.VehicleSource = vehicle
		entries = append(entries, entry)
	}

	return entries, nil
}

// read sends a data call with the token and decodes its answer. A 401 is
// errs.ErrSessionRejected.
func (c *APIClient) read(ctx context.Context, token, path string, target any) error {
	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader(headerToken, token).
		Get(path)
	if err != nil {
		return fmt.Errorf("%w: GET %s: %w", errs.ErrRequestFailed, path, err)
	}

	if resp.StatusCode() == http.StatusUnauthorized {
		return fmt.Errorf("%w: GET %s", errs.ErrSessionRejected, path)
	}

	if !resp.IsStatusSuccess() {
		return statusFailure(path, resp)
	}

	if err = json.Unmarshal(resp.Bytes(), target); err != nil {
		return fmt.Errorf("%w: GET %s: decoding the answer: %w", errs.ErrRequestFailed, path, err)
	}

	return nil
}

// logInFailure reads the signals of a rejected sign in: a 401, a 403, or the exception
// code 1908 in any answer.
func logInFailure(resp *resty.Response) error {
	var failure dto.ServiceError

	decodeErr := json.Unmarshal(resp.Bytes(), &failure)
	if decodeErr == nil && failure.Code() == dto.CodeCredentialsRejected {
		return errs.ErrCredentialsRejected
	}

	switch code := resp.StatusCode(); {
	case code == http.StatusUnauthorized, code == http.StatusForbidden:
		return errs.ErrCredentialsRejected
	case !resp.IsStatusSuccess():
		// The body of a failed sign in can repeat the email address, so the error names
		// the status alone.
		return fmt.Errorf(
			"%w: %s returned [%d]",
			errs.ErrRequestFailed,
			pathLogIn,
			resp.StatusCode(),
		)
	case decodeErr != nil:
		return fmt.Errorf(
			"%w: reading the answer of the sign in: %w",
			errs.ErrRequestFailed,
			decodeErr,
		)
	case failure.Code() != 0:
		return fmt.Errorf(
			"%w: the sign in answered the exception %d",
			errs.ErrRequestFailed,
			failure.Code(),
		)
	}

	return nil
}

func statusFailure(path string, resp *resty.Response) error {
	return fmt.Errorf(
		"%w: %s returned [%d] %s",
		errs.ErrRequestFailed,
		path,
		resp.StatusCode(),
		cutFailure(resp.String()),
	)
}

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

// IsRefusal reports a sign in that the car log service refused for the account, as against
// one that failed on the way.
func IsRefusal(err error) bool {
	return errors.Is(err, errs.ErrCredentialsRejected) ||
		errors.Is(err, errs.ErrNewPasswordRequired)
}
