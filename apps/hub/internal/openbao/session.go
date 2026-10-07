package openbao

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/openbao/openbao/api/auth/approle/v2"
	"github.com/openbao/openbao/api/v2"

	"github.com/abgeo/maroid/apps/hub/internal/config"
)

const (
	firstRetryDelay = time.Second
	maxRetryDelay   = 30 * time.Second
	loginTimeout    = 10 * time.Second
	retryGrowth     = 2
)

// Session holds the client that every use of OpenBao shares, and keeps its token alive.
type Session struct {
	client *api.Client
	login  *approle.AppRoleAuth
	logger *slog.Logger
	secret *api.Secret

	started  atomic.Bool
	stopOnce sync.Once
	stop     chan struct{}
	done     chan struct{}
}

// New builds the client and logs in with AppRole. A failed login returns an error,
// so the process does not start.
func New(ctx context.Context, cfg *config.OpenBao, logger *slog.Logger) (*Session, error) {
	clientConfig := api.DefaultConfig()
	clientConfig.Address = cfg.Address

	client, err := api.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("building the OpenBao client: %w", err)
	}

	login, err := approle.NewAppRoleAuth(
		cfg.RoleID,
		&approle.SecretID{FromString: cfg.SecretID},
	)
	if err != nil {
		return nil, fmt.Errorf("building the AppRole login: %w", err)
	}

	session := &Session{
		client: client,
		login:  login,
		logger: logger.With(
			slog.String("component", "secret-store"),
			slog.String("address", cfg.Address),
		),
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}

	session.secret, err = session.logIn(ctx)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// Client returns the shared client. A new login changes its token in place.
func (s *Session) Client() *api.Client {
	return s.client
}

// Run keeps the token alive until Stop. It ignores the cancellation of ctx, so the
// shutdown order of the command decides when the token stops being kept.
// It returns nil after Stop.
func (s *Session) Run(ctx context.Context) error {
	s.started.Store(true)
	defer close(s.done)

	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	defer cancel()

	go func() {
		select {
		case <-s.stop:
			cancel()
		case <-ctx.Done():
		}
	}()

	secret := s.secret

	for s.watch(ctx, secret) {
		var ok bool

		secret, ok = s.logInAgain(ctx)
		if !ok {
			break
		}
	}

	return nil
}

// Stop ends Run and waits for it, or returns the error of ctx at its deadline.
// It returns nil when Run never started.
func (s *Session) Stop(ctx context.Context) error {
	s.stopOnce.Do(func() { close(s.stop) })

	if !s.started.Load() {
		return nil
	}

	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("stopping the OpenBao session: %w", ctx.Err())
	}
}

// watch renews the token of secret until the token can be renewed no more. It returns
// false when ctx ends.
func (s *Session) watch(ctx context.Context, secret *api.Secret) bool {
	watcher, err := s.client.NewLifetimeWatcher(&api.LifetimeWatcherInput{
		Secret:        secret,
		RenewBehavior: api.RenewBehaviorIgnoreErrors,
	})
	if err != nil {
		s.logger.WarnContext(ctx, "watching the openbao token failed", slog.Any("error", err))

		return ctx.Err() == nil
	}

	go watcher.Start()
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case err := <-watcher.DoneCh():
			if err != nil {
				s.logger.WarnContext(
					ctx,
					"renewing the openbao token failed",
					slog.Any("error", err),
				)
			}

			return true
		case <-watcher.RenewCh():
			s.logger.DebugContext(ctx, "renewed the openbao token")
		}
	}
}

// logInAgain logs in until the store grants a token. It returns false when ctx ends.
func (s *Session) logInAgain(ctx context.Context) (*api.Secret, bool) {
	delay := firstRetryDelay

	for {
		secret, err := s.logIn(ctx)
		if err == nil {
			s.logger.InfoContext(ctx, "logged in to openbao")

			return secret, true
		}

		if ctx.Err() != nil {
			return nil, false
		}

		s.logger.ErrorContext(
			ctx,
			"logging in to openbao failed",
			slog.Any("error", err),
			slog.Duration("retry_in", delay),
		)

		timer := time.NewTimer(delay)

		select {
		case <-ctx.Done():
			timer.Stop()

			return nil, false
		case <-timer.C:
		}

		delay = min(delay*retryGrowth, maxRetryDelay)
	}
}

func (s *Session) logIn(ctx context.Context) (*api.Secret, error) {
	ctx, cancel := context.WithTimeout(ctx, loginTimeout)
	defer cancel()

	secret, err := s.client.Auth().Login(ctx, s.login)
	if err != nil {
		return nil, fmt.Errorf("logging in to OpenBao: %w", err)
	}

	return secret, nil
}
