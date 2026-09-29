package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/hellofresh/health-go/v5"
	"github.com/jmoiron/sqlx"
	"github.com/openbao/openbao/api/v2"

	"github.com/abgeo/maroid/apps/hub/internal/config"
)

// The paths of the two probes. The handler mounts them, and the access log skips
// a probe that succeeds.
const (
	LivenessPath  = "/livez"
	ReadinessPath = "/readyz"
)

const (
	component  = "maroid-hub"
	version    = "0.1.0"
	checkLimit = 2 * time.Second
)

// Checker measures the liveness and the readiness of the hub.
type Checker interface {
	Liveness(ctx context.Context) health.Check
	Readiness(ctx context.Context) health.Check
	Draining() bool
}

// Service measures the liveness and the readiness of the hub.
type Service struct {
	liveness  *health.Health
	readiness *health.Health
	draining  atomic.Bool
}

var _ Checker = (*Service)(nil)

// New creates a Service whose readiness checks the database, the IdP, and the
// secret store.
func New(
	cfg *config.Config,
	db *sqlx.DB,
	secretStore *api.Client,
	httpClient *http.Client,
) (*Service, error) {
	withComponent := health.WithComponent(health.Component{Name: component, Version: version})

	liveness, err := health.New(withComponent)
	if err != nil {
		return nil, fmt.Errorf("creating the liveness checker: %w", err)
	}

	checks := []health.Config{
		databaseCheck(db),
		idpCheck(httpClient, cfg.OIDC.Issuer),
		secretStoreCheck(secretStore),
	}

	readiness, err := health.New(
		withComponent,
		health.WithMaxConcurrent(len(checks)),
		health.WithChecks(checks...),
	)
	if err != nil {
		return nil, fmt.Errorf("creating the readiness checker: %w", err)
	}

	return &Service{liveness: liveness, readiness: readiness}, nil
}

// Liveness measures whether the process runs.
func (s *Service) Liveness(ctx context.Context) health.Check {
	return s.liveness.Measure(ctx)
}

// Readiness measures whether the hub can serve a request.
func (s *Service) Readiness(ctx context.Context) health.Check {
	return s.readiness.Measure(ctx)
}

// Drain marks the start of a shutdown.
func (s *Service) Drain() {
	s.draining.Store(true)
}

// Draining reports whether a shutdown has started.
func (s *Service) Draining() bool {
	return s.draining.Load()
}

// limited runs the check under the time limit of every check. The library stops
// its wait at the limit and does not cancel the check, so the check cancels its own
// call.
func limited(name string, check health.CheckFunc) health.Config {
	return health.Config{
		Name:    name,
		Timeout: checkLimit,
		Check: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, checkLimit)
			defer cancel()

			return check(ctx)
		},
	}
}
