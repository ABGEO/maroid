package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/parking/config"
	"github.com/abgeo/maroid/plugins/parking/errs"
	"github.com/abgeo/maroid/plugins/parking/model"
)

const (
	// SecretPending names the plugin secret that holds the pending connection.
	SecretPending = "pending"

	pendingLifetime = 5 * time.Minute
	// defaultCodeDelay applies when the parking service gives a wait that is not a number.
	defaultCodeDelay = 60 * time.Second
)

// CodeWaitError reports the time left before the parking service sends a new code.
// It matches errs.ErrCodeWait.
type CodeWaitError struct {
	Left time.Duration
}

// Error names the time left.
func (e *CodeWaitError) Error() string {
	return fmt.Sprintf("%s: %s left", errs.ErrCodeWait, e.Left.Round(time.Second))
}

// Is reports whether target is errs.ErrCodeWait.
func (e *CodeWaitError) Is(target error) bool {
	return target == errs.ErrCodeWait
}

// ConnectionService connects the parking account of the acting user.
type ConnectionService interface {
	Status(ctx context.Context) (*model.Connection, error)
	Start(ctx context.Context) (*model.PendingConnection, error)
	ResendCode(ctx context.Context) (*model.PendingConnection, error)
	Verify(ctx context.Context, code string) (*model.Connection, error)
	Disconnect(ctx context.Context) error
}

// Connections implements ConnectionService over the API client, the settings, and
// the plugin secrets.
type Connections struct {
	client   APIClientService
	settings pluginapi.PluginSettings
	secrets  pluginapi.PluginSecrets
}

var _ ConnectionService = (*Connections)(nil)

// pendingRecord is the stored form of a pending connection.
type pendingRecord struct {
	SignInID  string    `json:"sign_in_id"`
	Phone     string    `json:"phone"`
	ResendAt  time.Time `json:"resend_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NewConnections creates a new Connections instance.
func NewConnections(
	client APIClientService,
	settings pluginapi.PluginSettings,
	secrets pluginapi.PluginSecrets,
) *Connections {
	return &Connections{
		client:   client,
		settings: settings,
		secrets:  secrets,
	}
}

// Status reads the connection of the acting user. A connected account reads the
// parking service, which also proves that the token still works.
func (c *Connections) Status(ctx context.Context) (*model.Connection, error) {
	credentials, err := c.credentials(ctx)
	if err != nil && !errors.Is(err, errs.ErrCredentialsAbsent) {
		return nil, err
	}

	pending, err := c.pending(ctx)
	if err != nil && !errors.Is(err, errs.ErrPendingAbsent) {
		return nil, err
	}

	connection := &model.Connection{
		State:             model.ConnectionStateNotConnected,
		CredentialsStored: credentials != nil,
		Pending:           pending,
	}

	person, err := c.client.GetPerson(ctx)

	switch {
	case errors.Is(err, errs.ErrNotConnected):
		return connection, nil
	case err != nil:
		return nil, fmt.Errorf("reading the parking account: %w", err)
	}

	connection.State = model.ConnectionStateConnected
	connection.Account = &model.Account{
		FirstName:        person.FirstName,
		LastName:         person.LastName,
		Balance:          person.BalanceAmount,
		FreeParkingsLeft: person.DailyFreeParkingLeft,
	}

	return connection, nil
}

// Start signs in with the stored credentials, and asks the parking service to send
// a code by SMS. It replaces the pending connection that exists.
func (c *Connections) Start(ctx context.Context) (*model.PendingConnection, error) {
	credentials, err := c.credentials(ctx)
	if err != nil {
		return nil, err
	}

	result, err := c.client.SignIn(ctx, credentials.Username, credentials.Password)
	if err != nil {
		return nil, fmt.Errorf("signing in: %w", err)
	}

	if result.Status != statusTwoFactor || !result.TwoFA.PhoneVerification {
		return nil, fmt.Errorf(
			"%w: the status is %q",
			errs.ErrVerificationUnsupported,
			result.Status,
		)
	}

	return c.requestCode(ctx, result.UUID)
}

// ResendCode asks the parking service for a new code of the pending connection, once
// the wait that the parking service gave ends.
func (c *Connections) ResendCode(ctx context.Context) (*model.PendingConnection, error) {
	pending, err := c.pending(ctx)
	if err != nil {
		return nil, err
	}

	if left := time.Until(pending.ResendAt); left > 0 {
		return nil, &CodeWaitError{Left: left}
	}

	return c.requestCode(ctx, pending.SignInID)
}

// Verify sends the code of the pending connection, and stores the session token that
// the parking service gives for it. The new token replaces the old one only here, so
// the old one serves every command until the code succeeds.
func (c *Connections) Verify(ctx context.Context, code string) (*model.Connection, error) {
	pending, err := c.pending(ctx)
	if err != nil {
		return nil, err
	}

	token, err := c.client.SubmitCode(ctx, pending.SignInID, code)
	if err != nil {
		return nil, fmt.Errorf("verifying the code: %w", err)
	}

	if err = c.secrets.Put(ctx, pluginapi.SecretScopeUser, SecretSession, token); err != nil {
		return nil, fmt.Errorf("storing the session token: %w", err)
	}

	if err = c.secrets.Delete(ctx, pluginapi.SecretScopeUser, SecretPending); err != nil {
		return nil, fmt.Errorf("removing the pending connection: %w", err)
	}

	connection, err := c.Status(ctx)
	if err != nil {
		// The token is stored, so the account is connected. Only its read failed, and an
		// error would send the member back to a code that no pending connection holds.
		return &model.Connection{ //nolint:nilerr // see the reason above.
			State:             model.ConnectionStateConnected,
			CredentialsStored: true,
		}, nil
	}

	return connection, nil
}

// Disconnect removes the session token of the acting user. The settings stay.
func (c *Connections) Disconnect(ctx context.Context) error {
	if err := c.secrets.Delete(ctx, pluginapi.SecretScopeUser, SecretSession); err != nil {
		return fmt.Errorf("removing the session token: %w", err)
	}

	return nil
}

func (c *Connections) requestCode(
	ctx context.Context,
	signInID string,
) (*model.PendingConnection, error) {
	answer, err := c.client.RequestCode(ctx, signInID)
	if err != nil {
		return nil, fmt.Errorf("requesting a code: %w", err)
	}

	now := time.Now().UTC()
	record := pendingRecord{
		SignInID:  signInID,
		Phone:     answer.Phone,
		ResendAt:  now.Add(codeDelayOf(answer.Delay.String())),
		ExpiresAt: now.Add(pendingLifetime),
	}

	encoded, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("encoding the pending connection: %w", err)
	}

	err = c.secrets.PutUntil(
		ctx,
		pluginapi.SecretScopeUser,
		SecretPending,
		string(encoded),
		record.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("storing the pending connection: %w", err)
	}

	return record.toModel(), nil
}

func (c *Connections) credentials(ctx context.Context) (*config.UserSettings, error) {
	values, err := c.settings.Get(ctx)
	if errors.Is(err, pluginapi.ErrSettingsAbsent) {
		return nil, errs.ErrCredentialsAbsent
	}

	if err != nil {
		return nil, fmt.Errorf("reading the settings of the acting user: %w", err)
	}

	credentials := new(config.UserSettings)
	if err = pluginconfig.DecodeAndValidateSettings(values, credentials); err != nil {
		return nil, fmt.Errorf("decoding the settings of the acting user: %w", err)
	}

	if credentials.Username == "" || credentials.Password == "" {
		return nil, errs.ErrCredentialsAbsent
	}

	return credentials, nil
}

func (c *Connections) pending(ctx context.Context) (*model.PendingConnection, error) {
	encoded, err := c.secrets.Get(ctx, pluginapi.SecretScopeUser, SecretPending)
	if errors.Is(err, pluginapi.ErrSecretAbsent) {
		return nil, errs.ErrPendingAbsent
	}

	if err != nil {
		return nil, fmt.Errorf("reading the pending connection: %w", err)
	}

	var record pendingRecord
	if err = json.Unmarshal([]byte(encoded), &record); err != nil {
		return nil, fmt.Errorf("decoding the pending connection: %w", err)
	}

	return record.toModel(), nil
}

func (r pendingRecord) toModel() *model.PendingConnection {
	return &model.PendingConnection{
		SignInID:  r.SignInID,
		Phone:     r.Phone,
		ResendAt:  r.ResendAt,
		ExpiresAt: r.ExpiresAt,
	}
}

func codeDelayOf(seconds string) time.Duration {
	delay, err := time.ParseDuration(seconds + "s")
	if err != nil || delay <= 0 {
		return defaultCodeDelay
	}

	return delay
}
