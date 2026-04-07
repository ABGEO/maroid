package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/golang-jwt/jwt/v5"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/config"
)

const (
	stateCookieName     = "maroid_oauth_state"
	nonceCookieName     = "maroid_oauth_nonce"
	verifierCookieName  = "maroid_oauth_verifier"
	redirectCookieName  = "maroid_oauth_redirect"
	authTokenCookieName = "maroid_token"
	oauthCookieMaxAge   = 5 * 60           // 5 minutes in seconds
	authTokenMaxAge     = 7 * 24 * 60 * 60 // 7 days in seconds
)

var (
	errInvalidQueryParameter = errors.New("invalid query parameter")
	errInvalidRedirectCookie = errors.New("invalid redirect cookie value")
	errUserIsNotAllowed      = errors.New("user is not allowed")
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

	router.Route("/auth", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Get("/", Wrap(h.logger, h.Initiate))
			r.Get("/callback", Wrap(h.logger, h.Callback))
		})

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(h.logger, h.jwtSvc, h.cfg.Telegram.AllowedUsers))
			r.Get("/me", Wrap(h.logger, h.Me))
		})
	})
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

	setOAuthCookie(w, redirectCookieName, redirect)
	setOAuthCookie(w, stateCookieName, result.State)
	setOAuthCookie(w, nonceCookieName, result.Nonce)
	setOAuthCookie(w, verifierCookieName, result.Verifier)

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

	clearOAuthCookie(w, redirectCookieName)

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

	userID, err := strconv.ParseInt(idClaims.ID, 10, 64)
	if err != nil {
		redirectWithError(w, r, redirect)

		return fmt.Errorf("invalid user id claims: %w", err)
	}

	if !slices.Contains(h.cfg.Telegram.AllowedUsers, userID) {
		redirectWithError(w, r, redirect)

		return fmt.Errorf("user %d is not allowed: %w", userID, errUserIsNotAllowed)
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

	setAuthCookie(w, token)
	http.Redirect(w, r, redirect, http.StatusFound)

	return nil
}

func (h *Auth) Me(w http.ResponseWriter, r *http.Request) error {
	claims := auth.ClaimsFromContext(r.Context())

	render.JSON(w, r, map[string]any{
		"name":    claims.Name,
		"picture": claims.Picture,
	})

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

	clearOAuthCookie(w, stateCookieName)
	clearOAuthCookie(w, nonceCookieName)
	clearOAuthCookie(w, verifierCookieName)

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

func setAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     authTokenCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   authTokenMaxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func setOAuthCookie(w http.ResponseWriter, name string, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   oauthCookieMaxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearOAuthCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Unix(0, 0),
		SameSite: http.SameSiteLaxMode,
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
