package service_test

import (
	"time"

	"github.com/abgeo/maroid/plugins/drivvo/config"
)

// The members of the answers of the car log service, and the values, that the tests of
// this package repeat.
const (
	memberCode            = "code"
	memberException       = "exception"
	memberValue           = "valor"
	memberToken           = "token"
	memberDay             = "data"
	memberOdometer        = "odometro"
	memberMaintenanceType = "id_tipo_servico"
	currencyGEL           = "GEL"
)

// testConfig answers the configuration of the tests over the fake service at the origin.
func testConfig(baseURL string) *config.Config {
	return &config.Config{
		BaseURL:        baseURL,
		RequestTimeout: 5 * time.Second,
		AppVersion:     "23",
		AppVersionName: "14.2.3",
	}
}
