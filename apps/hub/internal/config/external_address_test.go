package config_test

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/config"
)

const (
	exampleHost = "https://hub.example.com"
	mcpPath     = "/mcp"
)

// newServer returns a server section that validates, so that a test varies the
// external address alone.
func newServer(externalURL string) config.Server {
	return config.Server{
		ExternalURL: externalURL,
		ListenAddr:  "0.0.0.0",
		Port:        "8000",
	}
}

// APIFMT-SC-014: One stored value builds every external address, and it carries
// the scheme.
func TestExternalAddressJoinsTheStoredValue(t *testing.T) {
	t.Parallel()

	for name, testCase := range map[string]struct {
		externalURL string
		path        string
		want        string
	}{
		"https": {
			externalURL: exampleHost,
			path:        "/telegram/webhook",
			want:        "https://hub.example.com/telegram/webhook",
		},
		"http": {
			externalURL: "http://hub.maroid.localhost",
			path:        mcpPath,
			want:        "http://hub.maroid.localhost" + mcpPath,
		},
		"a port": {
			externalURL: "http://localhost:8000",
			path:        mcpPath,
			want:        "http://localhost:8000" + mcpPath,
		},
		"a trailing slash never doubles": {
			externalURL: exampleHost + "/",
			path:        mcpPath,
			want:        exampleHost + mcpPath,
		},
		"no path gives the bare origin": {
			externalURL: exampleHost,
			path:        "",
			want:        exampleHost,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := newServer(testCase.externalURL)
			require.Equal(t, testCase.want, server.ExternalAddress(testCase.path))
		})
	}
}

// APIFMT-SC-014: A deployment that stores no external address, or one without a
// scheme, stops at the start.
func TestTheExternalAddressIsRequiredAndCarriesTheScheme(t *testing.T) {
	t.Parallel()

	validate := validator.New()

	for name, testCase := range map[string]struct {
		externalURL string
		valid       bool
	}{
		"an absolute address": {externalURL: exampleHost, valid: true},
		"another scheme":      {externalURL: "http://hub.maroid.localhost", valid: true},
		"no value":            {externalURL: "", valid: false},
		"a bare host":         {externalURL: "hub.example.com", valid: false},
		"a path with no host": {externalURL: "/mcp", valid: false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := validate.Struct(newServer(testCase.externalURL))

			if testCase.valid {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
		})
	}
}
