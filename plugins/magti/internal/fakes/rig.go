package fakes

import (
	"testing"
	"time"

	"github.com/abgeo/maroid/plugins/magti/config"
	"github.com/abgeo/maroid/plugins/magti/internal/fakeportal"
	"github.com/abgeo/maroid/plugins/magti/service"
)

const (
	requestTimeout = 5 * time.Second
	stateTimeout   = 4 * time.Second
)

// Rig runs the services of the plugin against the fake portal and the fakes.
type Rig struct {
	Portal   *fakeportal.Portal
	Secrets  *Secrets
	Settings *Settings
	Store    *Store
	Config   *config.Config
	Client   *service.APIClient
	Account  *service.Account
}

// NewRig starts a fake portal, and answers the services over it. An empty cookie stores
// no session.
func NewRig(t *testing.T, cookie string, store *Store) *Rig {
	t.Helper()

	portal := fakeportal.New(t)
	cfg := &config.Config{
		BaseURL:        portal.URL(),
		RequestTimeout: requestTimeout,
		StateTimeout:   stateTimeout,
		CronSchedule:   config.CronSchedule{TrafficCollector: "0 9 * * *"},
	}
	client := service.NewAPIClient(cfg)
	secrets := NewSecrets(cookie)
	settings := Credentials()

	return &Rig{
		Portal:   portal,
		Secrets:  secrets,
		Settings: settings,
		Store:    store,
		Config:   cfg,
		Client:   client,
		Account:  service.NewAccount(client, settings, secrets),
	}
}

// Collector answers the collector of the rig.
func (r *Rig) Collector() *service.Collector {
	return service.NewCollector(r.Account, r.Client, r.Store)
}
