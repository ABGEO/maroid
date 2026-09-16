package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

const (
	clientID  = "hub"
	keyID     = "the-signing-key"
	handleOfA = "abgeo"
)

// fakeDex serves the discovery document and the key set that the hub reads, and
// it signs a token the way Dex does. No test reaches a real identity provider.
type fakeDex struct {
	URL string

	key         *rsa.PrivateKey
	keyRequests atomic.Int64
}

// startFakeDex launches the provider and stops it when the test ends.
func startFakeDex(t *testing.T) *fakeDex {
	t.Helper()

	const keyBits = 2048

	key, err := rsa.GenerateKey(rand.Reader, keyBits)
	require.NoError(t, err)

	dex := &fakeDex{key: key}

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	t.Cleanup(server.Close)

	dex.URL = server.URL

	discovery := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, fmt.Sprintf(`{
			"issuer": %[1]q,
			"authorization_endpoint": "%[1]s/auth",
			"token_endpoint": "%[1]s/token",
			"jwks_uri": "%[1]s/keys",
			"id_token_signing_alg_values_supported": ["RS256"]
		}`, server.URL))
	}

	mux.HandleFunc("/.well-known/openid-configuration", discovery)

	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		dex.keyRequests.Add(1)

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, fmt.Sprintf(`{"keys":[{
			"kty": "RSA",
			"kid": %q,
			"alg": "RS256",
			"use": "sig",
			"n": %q,
			"e": %q
		}]}`,
			keyID,
			base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			base64.RawURLEncoding.EncodeToString(bigEndian(key.E)),
		))
	})

	return dex
}

// KeyRequests reports how often the hub read the key set. EXTID-NFR-001 bounds it.
func (d *fakeDex) KeyRequests() int64 {
	return d.keyRequests.Load()
}

// Claims builds the claim set that a connector of Dex produces. A test changes
// one member and signs the result.
func (d *fakeDex) Claims(connector string, account string) jwt.MapClaims {
	return jwt.MapClaims{
		"iss": d.URL,
		"aud": clientID,
		"sub": "a-subject-of-dex",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
		"federated_claims": map[string]any{
			"connector_id": connector,
			"user_id":      account,
		},
		"name":               "Temuri Takalandze",
		"preferred_username": handleOfA,
		"picture":            "https://example.com/a.jpg",
	}
}

// Sign mints a token for one external account, the way a connector of Dex does.
func (d *fakeDex) Sign(t *testing.T, connector string, account string) string {
	t.Helper()

	return d.SignClaims(t, d.Claims(connector, account))
}

// SignClaims mints a token with exactly the claims that the test gives.
func (d *fakeDex) SignClaims(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID

	signed, err := token.SignedString(d.key)
	require.NoError(t, err)

	return signed
}

func bigEndian(value int) []byte {
	var bytes []byte

	for value > 0 {
		bytes = append([]byte{byte(value & 0xff)}, bytes...)
		value >>= 8
	}

	return bytes
}
