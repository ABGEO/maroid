// Package authtest serves a fake OpenID Connect provider. It signs a token the
// way a connector of Dex does, so a test needs no identity provider.
package authtest

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"maps"
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

	mu         sync.Mutex
	codes      map[string]jwt.MapClaims
	atHashSalt string
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

// KeyRequests reports how often the hub read the key set.
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

// BreakAccessTokenBinding makes the identity token carry an at_hash of another
// value, so a test can prove that the hub refuses the exchange.
func (d *Provider) BreakAccessTokenBinding() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.atHashSalt = "another-value"
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

// accessTokenHash renders the at_hash claim of a token that RS256 signed: the
// left half of the SHA-256 digest, in URL safe base64.
func accessTokenHash(accessToken string) string {
	sum := sha256.Sum256([]byte(accessToken))

	return base64.RawURLEncoding.EncodeToString(sum[:len(sum)/2])
}

// copyClaims returns a claim set that a caller changes without touching the
// registered one.
func copyClaims(claims jwt.MapClaims) jwt.MapClaims {
	copied := make(jwt.MapClaims, len(claims))
	maps.Copy(copied, claims)

	return copied
}

// tokenHandler exchanges a registered code for the tokens that the test named.
//
// The access token is a signed token of the same shape as the identity token,
// the way Dex mints it, and the identity token carries the at_hash that binds the
// two. The hub reads that claim.
func (d *Provider) tokenHandler(key *rsa.PrivateKey) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		d.mu.Lock()
		claims, ok := d.codes[r.FormValue("code")]
		salt := d.atHashSalt
		d.mu.Unlock()

		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":"invalid_grant"}`)

			return
		}

		// The nonce belongs to the identity token. It binds that token to one
		// flow, and the access token carries no such binding.
		accessClaims := copyClaims(claims)
		delete(accessClaims, "nonce")
		accessToken := d.signWith(key, accessClaims)

		idClaims := copyClaims(claims)
		idClaims["at_hash"] = accessTokenHash(accessToken + salt)

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, fmt.Sprintf(
			`{"access_token":%q,"token_type":"bearer","expires_in":3600,"id_token":%q}`,
			accessToken,
			d.signWith(key, idClaims),
		))
	}
}
