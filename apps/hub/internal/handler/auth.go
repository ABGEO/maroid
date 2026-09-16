package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
)

const (
	//nolint:gosec // G101: this names the cookie, it holds no credential.
	authTokenCookieName = "maroid_token"
	// bindingCookieName holds the secret that binds one authorization flow to one
	// browser. It grants nothing on its own: the row holds every authority, and
	// this value only proves that the browser that finishes the flow started it.
	bindingCookieName = "maroid_auth_binding"
	bindingMaxAge     = 10 * 60 // 10 minutes in seconds, the lifetime of a flow
)

var (
	errInvalidQueryParameter  = errors.New("invalid query parameter")
	errMissingFederatedClaims = errors.New("the token carries no federated claims")
)

// AuthHandler represents the Auth handler interface.
type AuthHandler interface {
	Handler

	Initiate(w http.ResponseWriter, r *http.Request) error
	Callback(w http.ResponseWriter, r *http.Request) error
}

// Auth represents the authentication handler.
type Auth struct {
	cfg              *config.Config
	logger           *slog.Logger
	verifier         auth.TokenVerifier
	oidcFlow         *auth.OIDCFlow
	userRepo         repository.UserRepository
	identityRepo     repository.IdentityRepository
	identityResolver auth.IdentityResolver
}

var _ AuthHandler = (*Auth)(nil)

// NewAuth creates a new Auth handler.
func NewAuth(
	cfg *config.Config,
	logger *slog.Logger,
	verifier auth.TokenVerifier,
	oidcFlow *auth.OIDCFlow,
	userRepo repository.UserRepository,
	identityRepo repository.IdentityRepository,
	identityResolver auth.IdentityResolver,
) *Auth {
	return &Auth{
		cfg: cfg,
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "auth"),
		),
		verifier:         verifier,
		oidcFlow:         oidcFlow,
		userRepo:         userRepo,
		identityRepo:     identityRepo,
		identityResolver: identityResolver,
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
			r.Use(auth.Middleware(h.logger, h.verifier, h.identityResolver))
			r.Get("/me", Wrap(h.logger, h.Me))
		})
	})
}

// Initiate starts the OIDC flow.
func (h *Auth) Initiate(w http.ResponseWriter, r *http.Request) error {
	redirect := r.URL.Query().Get("redirect")
	if !validateRedirect(redirect, h.cfg.Auth.AllowedRedirects) {
		sendBadRequest(w, r, "missing or invalid redirect parameter")

		return fmt.Errorf("%w: missing or invalid redirect parameter", errInvalidQueryParameter)
	}

	// A sign in that names no provider reaches the connector list of Dex, which is
	// the page that a person picks from.
	flow := model.AuthFlow{Intent: model.IntentSignIn, Redirect: redirect}
	if provider := r.URL.Query().Get("provider"); provider != "" {
		flow.Provider = &provider
	}

	authURL, binding, err := h.oidcFlow.Initiate(r.Context(), flow)
	if err != nil {
		redirectWithError(w, r, redirect)

		return fmt.Errorf("initiating OIDC flow: %w", err)
	}

	setBindingCookie(w, binding)
	http.Redirect(w, r, authURL, http.StatusFound)

	return nil
}

// Callback completes the OIDC flow, verifies the ID token, and redirects with a signed JWT.
func (h *Auth) Callback(w http.ResponseWriter, r *http.Request) error {
	state := r.URL.Query().Get("state")

	binding := takeBindingCookie(w, r)

	flow, err := h.oidcFlow.Consume(r.Context(), state, binding)
	if err != nil && flow == nil {
		sendBadRequest(w, r, "invalid state")

		return fmt.Errorf("consuming the authorization flow: %w", err)
	}

	redirect := flow.Redirect

	// The row holds the target, and a row cannot come from the browser. The list
	// still applies, because the row outlives a change to the configuration.
	if !validateRedirect(redirect, h.cfg.Auth.AllowedRedirects) {
		sendBadRequest(w, r, "missing or invalid redirect parameter")

		return errInvalidQueryParameter
	}

	if err != nil {
		redirectWithError(w, r, redirect)

		return fmt.Errorf("consuming the authorization flow: %w", err)
	}

	rawToken, claims, err := h.processOIDCCallback(r, flow)
	if err != nil {
		redirectWithError(w, r, redirect)

		return err
	}

	if _, err = h.resolveAndSync(r.Context(), claims); err != nil {
		redirectWithError(w, r, redirect)

		return err
	}

	setAuthCookie(w, rawToken, h.cfg.Auth.SessionTTL)
	//nolint:gosec // G710: validateRedirect already matched the target against the allowed list.
	http.Redirect(w, r, redirect, http.StatusFound)

	return nil
}

// Me returns the authenticated user's information.
func (h *Auth) Me(w http.ResponseWriter, r *http.Request) error {
	claims := auth.ClaimsFromContext(r.Context())

	// @todo: create a structure for user data.
	render.JSON(w, r, map[string]any{
		"name":    claims.Name,
		"picture": claims.Picture,
	})

	return nil
}

// resolveAndSync reads the user record that the external account names, then
// writes the profile that the provider gave onto that identity.
func (h *Auth) resolveAndSync(ctx context.Context, claims *auth.Claims) (*model.User, error) {
	federated := claims.Federated
	if federated.ConnectorID == "" || federated.UserID == "" {
		return nil, errMissingFederatedClaims
	}

	user, err := h.identityResolver.ResolveByProvider(ctx, federated.ConnectorID, federated.UserID)
	if err != nil {
		return nil, fmt.Errorf("resolving the user record of the external account: %w", err)
	}

	err = h.identityRepo.SyncProfile(ctx, federated.ConnectorID, federated.UserID, model.Profile{
		Username:    claims.Username,
		DisplayName: claims.Name,
		PictureURL:  claims.Picture,
	})
	if err != nil {
		return nil, fmt.Errorf("syncing the profile of the identity: %w", err)
	}

	return user, nil
}

func (h *Auth) processOIDCCallback(
	r *http.Request,
	flow *model.AuthFlow,
) (string, *auth.Claims, error) {
	code := r.URL.Query().Get("code")
	if code == "" {
		return "", nil, fmt.Errorf("%w: missing code query parameter", errInvalidQueryParameter)
	}

	rawToken, claims, err := h.oidcFlow.Verify(r.Context(), code, flow.Nonce, flow.Verifier)
	if err != nil {
		return "", nil, fmt.Errorf("verifying OIDC flow: %w", err)
	}

	return rawToken, claims, nil
}

// redirectWithError sends the caller back to the target with an error marker.
// Every caller passes a target that validateRedirect already accepted.
func redirectWithError(w http.ResponseWriter, r *http.Request, target string) {
	redirectURL, _ := url.Parse(target)

	queryParams := redirectURL.Query()
	queryParams.Set("error", "auth_failed")
	redirectURL.RawQuery = queryParams.Encode()

	//nolint:gosec // G710: validateRedirect already matched the target against the allowed list.
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// takeBindingCookie reads the binding of this browser and clears it.
//
// The binding proves that this browser started the flow. A callback that carries
// a valid state without it is a forged one, and finishing it would sign the holder
// of this browser in as somebody else.
func takeBindingCookie(w http.ResponseWriter, r *http.Request) string {
	var binding string

	if cookie, err := r.Cookie(bindingCookieName); err == nil {
		binding = cookie.Value
	}

	clearBindingCookie(w)

	return binding
}

func setBindingCookie(w http.ResponseWriter, binding string) {
	http.SetCookie(w, &http.Cookie{
		Name:     bindingCookieName,
		Value:    binding,
		Path:     "/",
		MaxAge:   bindingMaxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearBindingCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     bindingCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Unix(0, 0),
		SameSite: http.SameSiteLaxMode,
	})
}

// sendBadRequest answers with the JSON body that API-006 gives.
func sendBadRequest(w http.ResponseWriter, r *http.Request, reason string) {
	render.Status(r, http.StatusBadRequest)
	render.JSON(w, r, map[string]string{"error": reason})
}

func setAuthCookie(w http.ResponseWriter, token string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     authTokenCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   true,
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
