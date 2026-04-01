package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/golang-jwt/jwt/v5"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/config"
)

const (
	stateCookieName    = "maroid_oauth_state"
	nonceCookieName    = "maroid_oauth_nonce"
	verifierCookieName = "maroid_oauth_verifier"
	oauthCookieMaxAge  = 5 * 60 // 5 minutes in seconds
)

var errInvalidQueryParameter = errors.New("invalid query parameter")

// AuthHandler represents the Auth handler interface.
type AuthHandler interface {
	Handler

	Initiate(w http.ResponseWriter, r *http.Request) error
	Callback(w http.ResponseWriter, r *http.Request) error
}

// Auth represents the authentication handler.
type Auth struct {
	cfg      *config.Config
	logger   *slog.Logger
	jwtSvc   *auth.JWTService
	oidcFlow *auth.OIDCFlow
}

var _ AuthHandler = (*Auth)(nil)

// NewAuth creates a new Auth handler.
func NewAuth(
	cfg *config.Config,
	logger *slog.Logger,
	jwtSvc *auth.JWTService,
	oidcFlow *auth.OIDCFlow,
) *Auth {
	return &Auth{
		cfg: cfg,
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "auth"),
		),
		jwtSvc:   jwtSvc,
		oidcFlow: oidcFlow,
	}
}

// Register registers the auth routes.
func (h *Auth) Register(router chi.Router) {
	h.logger.Debug("registering routes")

	router.Get("/auth", Wrap(h.logger, h.Initiate))
	router.Get("/auth/callback", Wrap(h.logger, h.Callback))
}

// Initiate starts the OIDC flow.
func (h *Auth) Initiate(w http.ResponseWriter, r *http.Request) error {
	result, err := h.oidcFlow.Initiate()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "internal server error"})

		return fmt.Errorf("initiating OIDC flow: %w", err)
	}

	setOAuthCookie(w, r, stateCookieName, result.State)
	setOAuthCookie(w, r, nonceCookieName, result.Nonce)
	setOAuthCookie(w, r, verifierCookieName, result.Verifier)

	http.Redirect(w, r, result.AuthURL, http.StatusFound)

	return nil
}

// Callback completes the OIDC flow, verifies the ID token, and returns a signed JWT.
func (h *Auth) Callback(w http.ResponseWriter, r *http.Request) error {
	idClaims, err := h.processOIDCCallback(w, r)
	if err != nil {
		return err
	}

	token, err := h.jwtSvc.Sign(auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: idClaims.ID,
		},
		Name:     idClaims.Name,
		Username: idClaims.Username,
		Picture:  idClaims.Picture,
	})
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "internal server error"})

		return fmt.Errorf("signing JWT: %w", err)
	}

	render.JSON(w, r, map[string]string{
		"type":  "Bearer",
		"token": token,
	})

	return nil
}

func (h *Auth) processOIDCCallback(
	w http.ResponseWriter,
	r *http.Request,
) (*auth.IDTokenClaims, error) {
	stateCookie, err := requireOAuthCookie(w, r, stateCookieName)
	if err != nil {
		return nil, err
	}

	nonceCookie, err := requireOAuthCookie(w, r, nonceCookieName)
	if err != nil {
		return nil, err
	}

	verifierCookie, err := requireOAuthCookie(w, r, verifierCookieName)
	if err != nil {
		return nil, err
	}

	clearOAuthCookie(w, r, stateCookieName)
	clearOAuthCookie(w, r, nonceCookieName)
	clearOAuthCookie(w, r, verifierCookieName)

	state := r.URL.Query().Get("state")
	if state == "" || state != stateCookie {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid request"})

		return nil, fmt.Errorf(
			"%w: state mismatch: got %q, expected %q",
			errInvalidQueryParameter,
			state,
			stateCookie,
		)
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid request"})

		return nil, fmt.Errorf("%w: missing code query parameter", errInvalidQueryParameter)
	}

	idClaims, err := h.oidcFlow.Verify(r.Context(), code, nonceCookie, verifierCookie)
	if err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "authentication failed"})

		return nil, fmt.Errorf("verifying OIDC flow: %w", err)
	}

	return idClaims, nil
}

func setOAuthCookie(w http.ResponseWriter, r *http.Request, name string, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/auth/callback",
		MaxAge:   oauthCookieMaxAge,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

func requireOAuthCookie(w http.ResponseWriter, r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid request"})

		return "", fmt.Errorf("retrieving cookie: %s: %w", name, err)
	}

	return cookie.Value, nil
}

func clearOAuthCookie(w http.ResponseWriter, r *http.Request, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/auth/callback",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
	})
}
