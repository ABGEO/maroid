package dex

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dexidp/dex/api/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
)

// Client reads and writes the connectors and the passwords that Dex holds.
type Client interface {
	ListConnectors(ctx context.Context) ([]Connector, error)
	CreateConnector(ctx context.Context, connector Connector) error
	UpdateConnector(ctx context.Context, connector Connector) error
	DeleteConnector(ctx context.Context, id string) error
	ListPasswords(ctx context.Context) ([]Password, error)
	CreatePassword(ctx context.Context, password Password) error
	UpdatePassword(ctx context.Context, email string, hash []byte) error
	DeletePassword(ctx context.Context, email string) error
	VerifyPassword(ctx context.Context, email string, password []byte) (bool, error)
}

// Connector is one connector of Dex, with its config as Dex stores it.
type Connector struct {
	ID     string
	Type   string
	Name   string
	Config json.RawMessage
}

// Password is one local account of Dex. Dex never answers the hash.
type Password struct {
	Email  string
	Hash   []byte
	Name   string
	UserID string
}

// GRPC is the Client that reaches Dex over gRPC with mutual TLS.
type GRPC struct {
	conn    *grpc.ClientConn
	api     api.DexClient
	timeout time.Duration
}

var _ Client = (*GRPC)(nil)

// New creates a client that presents the certificate of the configuration and
// trusts its authority alone. It opens no connection until the first call.
func New(cfg *config.Dex) (*GRPC, error) {
	mutual, err := mutualTLS(cfg)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.NewClient(
		cfg.Address,
		grpc.WithTransportCredentials(credentials.NewTLS(mutual)),
	)
	if err != nil {
		return nil, fmt.Errorf("creating the gRPC client of Dex: %w", err)
	}

	return &GRPC{
		conn:    conn,
		api:     api.NewDexClient(conn),
		timeout: cfg.Timeout,
	}, nil
}

func mutualTLS(cfg *config.Dex) (*tls.Config, error) {
	authority, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("reading the authority of Dex: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(authority) {
		return nil, fmt.Errorf("reading %s: %w", cfg.CAFile, errs.ErrDexNoAuthority)
	}

	pair, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("reading the client certificate of the hub: %w", err)
	}

	return &tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{pair},
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// Close closes the connection to Dex.
func (c *GRPC) Close() error {
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("closing the gRPC client of Dex: %w", err)
	}

	return nil
}

// ListConnectors answers every connector, the ones of the configuration file of
// Dex included.
func (c *GRPC) ListConnectors(ctx context.Context) ([]Connector, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	answer, err := c.api.ListConnectors(ctx, &api.ListConnectorReq{})
	if err != nil {
		return nil, failure("listing the connectors", err)
	}

	connectors := make([]Connector, 0, len(answer.GetConnectors()))
	for _, one := range answer.GetConnectors() {
		connectors = append(connectors, Connector{
			ID:     one.GetId(),
			Type:   one.GetType(),
			Name:   one.GetName(),
			Config: one.GetConfig(),
		})
	}

	return connectors, nil
}

// CreateConnector stores a new connector.
func (c *GRPC) CreateConnector(ctx context.Context, connector Connector) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	answer, err := c.api.CreateConnector(ctx, &api.CreateConnectorReq{Connector: &api.Connector{
		Id:     connector.ID,
		Type:   connector.Type,
		Name:   connector.Name,
		Config: connector.Config,
	}})
	if err != nil {
		return failure("creating a connector", err)
	}

	if answer.GetAlreadyExists() {
		return fmt.Errorf("creating the connector %s: %w", connector.ID, errs.ErrProviderExists)
	}

	return nil
}

// UpdateConnector replaces the type, the name, and the config of a connector.
func (c *GRPC) UpdateConnector(ctx context.Context, connector Connector) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	answer, err := c.api.UpdateConnector(ctx, &api.UpdateConnectorReq{
		Id:        connector.ID,
		NewType:   connector.Type,
		NewName:   connector.Name,
		NewConfig: connector.Config,
	})
	if err != nil {
		return failure("updating a connector", err)
	}

	if answer.GetNotFound() {
		return fmt.Errorf("updating the connector %s: %w", connector.ID, errs.ErrProviderNotFound)
	}

	return nil
}

// DeleteConnector removes a connector.
func (c *GRPC) DeleteConnector(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	answer, err := c.api.DeleteConnector(ctx, &api.DeleteConnectorReq{Id: id})
	if err != nil {
		return failure("deleting a connector", err)
	}

	if answer.GetNotFound() {
		return fmt.Errorf("deleting the connector %s: %w", id, errs.ErrProviderNotFound)
	}

	return nil
}

// ListPasswords answers every local account, with no hash.
func (c *GRPC) ListPasswords(ctx context.Context) ([]Password, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	answer, err := c.api.ListPasswords(ctx, &api.ListPasswordReq{})
	if err != nil {
		return nil, failure("listing the passwords", err)
	}

	passwords := make([]Password, 0, len(answer.GetPasswords()))
	for _, one := range answer.GetPasswords() {
		passwords = append(passwords, Password{
			Email:  one.GetEmail(),
			Name:   one.GetUsername(),
			UserID: one.GetUserId(),
		})
	}

	return passwords, nil
}

// CreatePassword stores a new local account.
func (c *GRPC) CreatePassword(ctx context.Context, password Password) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	answer, err := c.api.CreatePassword(ctx, &api.CreatePasswordReq{Password: &api.Password{
		Email:    password.Email,
		Hash:     password.Hash,
		Username: password.Name,
		UserId:   password.UserID,
	}})
	if err != nil {
		return failure("creating a password", err)
	}

	if answer.GetAlreadyExists() {
		return fmt.Errorf("creating a password: %w", errs.ErrLocalAccountExists)
	}

	return nil
}

// UpdatePassword replaces the hash of a local account and keeps its name.
func (c *GRPC) UpdatePassword(ctx context.Context, email string, hash []byte) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	answer, err := c.api.UpdatePassword(ctx, &api.UpdatePasswordReq{Email: email, NewHash: hash})
	if err != nil {
		return failure("updating a password", err)
	}

	if answer.GetNotFound() {
		return fmt.Errorf("updating a password: %w", errs.ErrLocalAccountNotFound)
	}

	return nil
}

// DeletePassword removes a local account.
func (c *GRPC) DeletePassword(ctx context.Context, email string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	answer, err := c.api.DeletePassword(ctx, &api.DeletePasswordReq{Email: email})
	if err != nil {
		return failure("deleting a password", err)
	}

	if answer.GetNotFound() {
		return fmt.Errorf("deleting a password: %w", errs.ErrLocalAccountNotFound)
	}

	return nil
}

// VerifyPassword answers whether the password matches the local account of the address.
func (c *GRPC) VerifyPassword(ctx context.Context, email string, password []byte) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	answer, err := c.api.VerifyPassword(ctx, &api.VerifyPasswordReq{
		Email:    email,
		Password: string(password),
	})
	if err != nil {
		return false, failure("verifying a password", err)
	}

	if answer.GetNotFound() {
		return false, fmt.Errorf("verifying a password: %w", errs.ErrLocalAccountNotFound)
	}

	return answer.GetVerified(), nil
}

// failure marks a call that never reached an answer of Dex, so the caller can
// report the IdP as unavailable and not as a fault of the request.
func failure(doing string, err error) error {
	if code := status.Code(err); code == codes.DeadlineExceeded || code == codes.Unavailable {
		return fmt.Errorf("%s: %w: %w", doing, errs.ErrIDPUnavailable, err)
	}

	return fmt.Errorf("%s: %w", doing, err)
}
