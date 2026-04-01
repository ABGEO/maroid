package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/config"
)

const (
	stateCookieName    = "maroid_oauth_state"
	nonceCookieName    = "maroid_oauth_nonce"
	verifierCookieName = "maroid_oauth_verifier"
	redirectCookieName = "maroid_oauth_redirect"
	oauthCookieMaxAge  = 5 * 60 // 5 minutes in seconds
)

var (
	errInvalidQueryParameter = errors.New("invalid query parameter")
	errInvalidRedirectCookie = errors.New("invalid redirect cookie value")
)

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
	redirect := r.URL.Query().Get("redirect")
	if !validateRedirect(redirect, h.cfg.Auth.AllowedRedirects) {
		http.Error(w, "Missing or invalid 'redirect' parameter", http.StatusBadRequest)

		return fmt.Errorf("%w: missing or invalid redirect parameter", errInvalidQueryParameter)
	}

	result, err := h.oidcFlow.Initiate()
	if err != nil {
		redirectWithError(w, r, redirect)

		return fmt.Errorf("initiating OIDC flow: %w", err)
	}

	setOAuthCookie(w, r, redirectCookieName, redirect)
	setOAuthCookie(w, r, stateCookieName, result.State)
	setOAuthCookie(w, r, nonceCookieName, result.Nonce)
	setOAuthCookie(w, r, verifierCookieName, result.Verifier)

	http.Redirect(w, r, result.AuthURL, http.StatusFound)

	return nil
}

// Callback completes the OIDC flow, verifies the ID token, and redirects with a signed JWT.
func (h *Auth) Callback(w http.ResponseWriter, r *http.Request) error {
	redirectCookie, err := r.Cookie(redirectCookieName)
	if err != nil {
		http.Error(w, "Missing or invalid 'redirect' parameter", http.StatusBadRequest)

		return fmt.Errorf("retrieving redirect cookie: %w", err)
	}

	redirect := redirectCookie.Value

	clearOAuthCookie(w, r, redirectCookieName)

	// The redirect URL is stored in an HTTP-only cookie to prevent tampering.
	// We still need to validate it against the allowed list to prevent open redirect vulnerabilities.
	if !validateRedirect(redirect, h.cfg.Auth.AllowedRedirects) {
		http.Error(w, "Missing or invalid 'redirect' parameter", http.StatusBadRequest)

		return errInvalidRedirectCookie
	}

	idClaims, err := h.processOIDCCallback(w, r)
	if err != nil {
		redirectWithError(w, r, redirect)

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
		redirectWithError(w, r, redirect)

		return fmt.Errorf("signing JWT: %w", err)
	}

	redirectWithToken(w, r, redirect, token)

	return nil
}

func (h *Auth) processOIDCCallback(
	w http.ResponseWriter,
	r *http.Request,
) (*auth.IDTokenClaims, error) {
	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil {
		return nil, fmt.Errorf("retrieving cookie: %s: %w", stateCookieName, err)
	}

	nonceCookie, err := r.Cookie(nonceCookieName)
	if err != nil {
		return nil, fmt.Errorf("retrieving cookie: %s: %w", nonceCookieName, err)
	}

	verifierCookie, err := r.Cookie(verifierCookieName)
	if err != nil {
		return nil, fmt.Errorf("retrieving cookie: %s: %w", verifierCookieName, err)
	}

	clearOAuthCookie(w, r, stateCookieName)
	clearOAuthCookie(w, r, nonceCookieName)
	clearOAuthCookie(w, r, verifierCookieName)

	state := r.URL.Query().Get("state")
	if state == "" || state != stateCookie.Value {
		return nil, fmt.Errorf(
			"%w: state mismatch: got %q, expected %q",
			errInvalidQueryParameter,
			state,
			stateCookie.Value,
		)
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		return nil, fmt.Errorf("%w: missing code query parameter", errInvalidQueryParameter)
	}

	idClaims, err := h.oidcFlow.Verify(r.Context(), code, nonceCookie.Value, verifierCookie.Value)
	if err != nil {
		return nil, fmt.Errorf("verifying OIDC flow: %w", err)
	}

	return idClaims, nil
}

func redirectWithError(w http.ResponseWriter, r *http.Request, target string) {
	redirectURL, _ := url.Parse(target)

	queryParams := redirectURL.Query()
	queryParams.Set("error", "auth_failed")
	redirectURL.RawQuery = queryParams.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

func redirectWithToken(w http.ResponseWriter, r *http.Request, target string, token string) {
	redirectURL, _ := url.Parse(target)
	redirectURL.Fragment = "token=" + token

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
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

func validateRedirect(redirect string, allowed []string) bool {
	parsed, err := url.Parse(redirect)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}

	redirectOrigin := parsed.Scheme + "://" + parsed.Host

	for _, rawAllowed := range allowed {
		parsedAllowed, err := url.Parse(rawAllowed)
		if err != nil {
			continue
		}

		allowedOrigin := parsedAllowed.Scheme + "://" + parsedAllowed.Host
		if redirectOrigin == allowedOrigin {
			return true
		}
	}

	return false
}
