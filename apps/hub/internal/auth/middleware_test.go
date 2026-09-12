package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/logger"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const recordID = "01998aa0-1111-7000-8000-000000000001"

// fakeUserRepo answers with the record that the test gives, or with the error.
// The methods that the test does not reach report a missing record.
type fakeUserRepo struct {
	user *model.User
	err  error
}

var _ repository.UserRepository = (*fakeUserRepo)(nil)

func (f fakeUserRepo) GetActiveByID(context.Context, string) (*model.User, error) {
	return f.user, f.err
}

func (f fakeUserRepo) GetActiveByTelegramID(context.Context, int64) (*model.User, error) {
	return f.user, f.err
}

func (f fakeUserRepo) ListActive(context.Context) ([]model.User, error) {
	return nil, errs.ErrUserNotFound
}

func (f fakeUserRepo) SyncProfileByTelegramID(
	context.Context, int64, model.Profile,
) (*model.User, error) {
	return f.user, f.err
}

func newJWTService(t *testing.T) *auth.JWTService {
	t.Helper()

	const bits = 2048

	key, err := rsa.GenerateKey(rand.Reader, bits)
	require.NoError(t, err)

	dir := t.TempDir()
	privatePath := filepath.Join(dir, "private.pem")
	publicPath := filepath.Join(dir, "public.pem")

	private := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	require.NoError(t, os.WriteFile(privatePath, private, 0o600))

	publicBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)

	public := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicBytes})
	require.NoError(t, os.WriteFile(publicPath, public, 0o600))

	service, err := auth.NewJWTService(&config.Config{
		JWT: config.JWT{
			Issuer:      "https://hub.maroid.test",
			PrivateKey:  privatePath,
			PublicKey:   publicPath,
			TokenExpiry: time.Hour,
		},
	})
	require.NoError(t, err)

	return service
}

func newLogger(t *testing.T) *slog.Logger {
	t.Helper()

	instance, err := logger.New(&config.Config{
		Logger: config.Logger{Level: "error", Format: "json"},
	})
	require.NoError(t, err)

	return instance
}

// IDENT-SC-006: A request whose subject names no active record gets status 401,
// and the next handler does not run.
func TestMiddlewareRefusesAPersonWithNoActiveRecord(t *testing.T) {
	t.Parallel()

	service := newJWTService(t)

	token, err := service.Sign(auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: recordID},
	})
	require.NoError(t, err)

	cases := map[string]struct {
		token    string
		userRepo fakeUserRepo
	}{
		"the request carries no token": {
			token:    "",
			userRepo: fakeUserRepo{user: activeRecord(), err: nil},
		},
		"no active record holds the subject": {
			token:    token,
			userRepo: fakeUserRepo{user: nil, err: errs.ErrUserNotFound},
		},
		"the subject claim is not a UUID": {
			token:    signWithSubject(t, service, "722183546"),
			userRepo: fakeUserRepo{user: activeRecord(), err: nil},
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			called := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })

			recorder := serve(t, service, testCase.userRepo, testCase.token, next)

			require.Equal(t, http.StatusUnauthorized, recorder.Code)
			require.False(t, called, "the next handler must not run")
		})
	}
}

// IDENT-FR-003: The request that an active record carries reaches the data with
// that record as the acting user.
func TestMiddlewarePutsTheActingUserInTheContext(t *testing.T) {
	t.Parallel()

	service := newJWTService(t)

	token, err := service.Sign(auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: recordID},
	})
	require.NoError(t, err)

	var acting string

	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		acting = pluginapi.ActingUserFromContext(r.Context())
	})

	recorder := serve(t, service, fakeUserRepo{user: activeRecord(), err: nil}, token, next)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, recordID, acting)
}

func activeRecord() *model.User {
	return &model.User{
		ID:         recordID,
		TelegramID: 722183546,
		Status:     model.StatusActive,
	}
}

func signWithSubject(t *testing.T, service *auth.JWTService, subject string) string {
	t.Helper()

	token, err := service.Sign(auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: subject},
	})
	require.NoError(t, err)

	return token
}

func serve(
	t *testing.T,
	service *auth.JWTService,
	userRepo fakeUserRepo,
	token string,
	next http.Handler,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/plugins", nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()

	auth.Middleware(newLogger(t), service, userRepo)(next).ServeHTTP(recorder, request)

	return recorder
}
