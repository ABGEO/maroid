// Package authtest serves a fake OpenID Connect provider. It signs a token the
// way a connector of Dex does, so a test needs no identity provider.
package authtest

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

const (
	// ClientID is the audience that every token of this provider names.
	ClientID = "hub"

	keyID = "the-signing-key"
	// HandleOfA is the account handle that the signed claims carry.
	HandleOfA = "abgeo"
)

// Provider serves the discovery document and the key set that the hub reads, and
// it signs a token the way Dex does. No test reaches a real identity provider.
type Provider struct {
	URL string

	key         *rsa.PrivateKey
	keyRequests atomic.Int64

	mu    sync.Mutex
	codes map[string]jwt.MapClaims
}

// StartProvider launches the provider and stops it when the test ends.
func StartProvider(t *testing.T) *Provider {
	t.Helper()

	const keyBits = 2048

	key, err := rsa.GenerateKey(rand.Reader, keyBits)
	require.NoError(t, err)

	dex := &Provider{key: key, codes: map[string]jwt.MapClaims{}}

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

	mux.HandleFunc("/token", dex.tokenHandler(key))

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
func (d *Provider) KeyRequests() int64 {
	return d.keyRequests.Load()
}

// Claims builds the claim set that a connector of Dex produces. A test changes
// one member and signs the result.
func (d *Provider) Claims(connector string, account string) jwt.MapClaims {
	return jwt.MapClaims{
		"iss": d.URL,
		"aud": ClientID,
		"sub": "a-subject-of-dex",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
		"federated_claims": map[string]any{
			"connector_id": connector,
			"user_id":      account,
		},
		"name":               "Temuri Takalandze",
		"preferred_username": HandleOfA,
		"picture":            "https://example.com/a.jpg",
	}
}

// Sign mints a token for one external account, the way a connector of Dex does.
func (d *Provider) Sign(t *testing.T, connector string, account string) string {
	t.Helper()

	return d.SignClaims(t, d.Claims(connector, account))
}

// SignClaims mints a token with exactly the claims that the test gives.
func (d *Provider) SignClaims(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID

	signed, err := token.SignedString(d.key)
	require.NoError(t, err)

	return signed
}

// bigEndian renders the public exponent the way a JSON Web Key carries it.
func bigEndian(value int) []byte {
	const (
		lowByte  = 0xff
		byteBits = 8
	)

	var bytes []byte

	for value > 0 {
		bytes = append([]byte{byte(value & lowByte)}, bytes...)
		value >>= byteBits
	}

	return bytes
}

// IssueCode registers an authorization code and the token that the exchange of
// that code returns. The nonce must match the flow row, because the hub checks it.
func (d *Provider) IssueCode(code string, claims jwt.MapClaims) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.codes[code] = claims
}

func (d *Provider) signWith(key *rsa.PrivateKey, claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID

	signed, err := token.SignedString(key)
	if err != nil {
		return ""
	}

	return signed
}

// tokenHandler exchanges a registered code for the token that the test named.
func (d *Provider) tokenHandler(key *rsa.PrivateKey) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		d.mu.Lock()
		claims, ok := d.codes[r.FormValue("code")]
		d.mu.Unlock()

		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":"invalid_grant"}`)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, fmt.Sprintf(
			`{"access_token":"an-access-token","token_type":"bearer","id_token":%q}`,
			d.signWith(key, claims),
		))
	}
}
