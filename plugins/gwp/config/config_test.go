package config_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/gwp/config"
)

const (
	keyEnabled = "enabled"
	keyScheme  = "scheme"
	keyHost    = "host"
	keyPort    = "port"
	proxyHost  = "proxy.example"
	proxyPort  = 7777
)

func TestTheProxyIsValidatedWhenEnabled(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		proxy map[string]any
		valid bool
	}{
		"no proxy": {proxy: nil, valid: true},
		"disabled, partial": {
			proxy: map[string]any{keyEnabled: false, keyHost: proxyHost},
			valid: true,
		},
		"enabled": {
			proxy: map[string]any{keyEnabled: true, keyHost: proxyHost, keyPort: proxyPort},
			valid: true,
		},
		"enabled, an IP": {
			proxy: map[string]any{keyEnabled: true, keyHost: "10.0.0.1", keyPort: proxyPort},
			valid: true,
		},
		"enabled, no host": {
			proxy: map[string]any{keyEnabled: true, keyPort: proxyPort},
			valid: false,
		},
		"enabled, no port": {
			proxy: map[string]any{keyEnabled: true, keyHost: proxyHost},
			valid: false,
		},
		"enabled, a URL": {
			proxy: map[string]any{
				keyEnabled: true,
				keyHost:    "https://" + proxyHost,
				keyPort:    proxyPort,
			},
		},
		"unknown scheme": {
			proxy: map[string]any{
				keyEnabled: true,
				keyScheme:  "ftp",
				keyHost:    proxyHost,
				keyPort:    proxyPort,
			},
		},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := decodeProxy(setup.proxy)

			if setup.valid {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
		})
	}
}

func TestTheProxySchemeDefaultsToHTTPS(t *testing.T) {
	t.Parallel()

	decoded := new(config.Config)

	require.NoError(t, pluginconfig.DecodeAndValidateConfig(map[string]any{}, decoded))
	require.Equal(t, "https", decoded.Proxy.Scheme)
}

func decodeProxy(proxy map[string]any) error {
	values := map[string]any{}
	if proxy != nil {
		values["proxy"] = proxy
	}

	if err := pluginconfig.DecodeAndValidateConfig(values, new(config.Config)); err != nil {
		return fmt.Errorf("decoding the proxy: %w", err)
	}

	return nil
}
