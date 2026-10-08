package provider_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/provider"
)

// dexOIDCKeys answers every key of the OIDC config that Dex v2.46.0 answered for a
// connector of its file on 2026-10-08.
func dexOIDCKeys() []string {
	return []string{
		"issuer", "issuerAlias", "clientID", "clientSecret", "redirectURI",
		"providerDiscoveryOverrides", "basicAuthUnsupported", "scopes", "hostedDomains",
		"rootCAs", "insecureSkipEmailVerified", "insecureEnableGroups", "allowedGroups",
		"acrValues", "insecureSkipVerify", "getUserInfo", "userIDKey", "userNameKey",
		"promptType", "pkceChallenge", "overrideClaimMapping", "claimMapping",
		"claimModifications",
	}
}

// IDPROV-SC-008: The options and the fixed fields together cover every key of the
// OIDC config of Dex, and no key that Dex lacks.
func TestTheOptionsMirrorTheConfigOfDex(t *testing.T) {
	t.Parallel()

	covered := append(provider.OptionKeys(), provider.FixedKeys()...)
	slices.Sort(covered)

	expected := dexOIDCKeys()
	expected = append(expected, provider.MarkerKey)
	slices.Sort(expected)

	assert.Equal(t, expected, covered)
}

func optionsOf(t *testing.T, text string) map[string]json.RawMessage {
	t.Helper()

	var options map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(text), &options))

	return options
}

func requireField(t *testing.T, err error, pointer string) {
	t.Helper()

	var failure *provider.FieldError
	require.ErrorAs(t, err, &failure)
	assert.Equal(t, pointer, failure.Pointer)
}

// IDPROV-SC-008: An option that Dex knows passes. A key that the preset fixes, a
// key that Dex lacks, and a value of the wrong type each name their key.
func TestTheOptionsAreChecked(t *testing.T) {
	t.Parallel()

	require.NoError(t, provider.ValidateOptions(optionsOf(t, `{"getUserInfo": true}`)))
	require.NoError(t, provider.ValidateOptions(optionsOf(t,
		`{"claimMapping": {"email": "mail"}, "hostedDomains": ["abgeo.dev"]}`)))

	requireField(t, provider.ValidateOptions(optionsOf(t, `{"userIDKey": "email"}`)),
		"/options/userIDKey")
	requireField(t, provider.ValidateOptions(optionsOf(t, `{"maroidPreset": "local"}`)),
		"/options/maroidPreset")
	requireField(t, provider.ValidateOptions(optionsOf(t, `{"getUserinfo": true}`)),
		"/options/getUserinfo")
	requireField(t, provider.ValidateOptions(optionsOf(t, `{"getUserInfo": "yes"}`)),
		"/options/getUserInfo")
	requireField(t, provider.ValidateOptions(optionsOf(t, `{"claimMapping": {"email": 1}}`)),
		"/options/claimMapping/email")
	requireField(t, provider.ValidateOptions(optionsOf(t, `{"claimMapping": {"mail": "x"}}`)),
		"/options/claimMapping")
}
