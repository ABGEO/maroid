package handler

import (
	"context"
	"crypto/sha256"
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
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
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
	// errorKey names the reason in a JSON failure. See API-006.
	errorKey = "error"
)

// The reasons that the hub reports at the target of a flow. Section 4.5 of the
// specification gives each one, and the deck renders a message for each.
const (
	reasonAuthFailed        = "auth_failed"
	reasonNoIdentity        = "no_identity"
	reasonAccessDenied      = "access_denied"
	reasonIdentityTaken     = "identity_taken"
	reasonInvitationInvalid = "invitation_invalid"
)

var (
	errInvalidQueryParameter  = errors.New("invalid query parameter")
	errMissingFederatedClaims = errors.New("the token carries no federated claims")
	errRecordNotActive        = errors.New("the user record is not active")
)

// AuthHandler represents the Auth handler interface.
type AuthHandler interface {
	Handler

	Initiate(w http.ResponseWriter, r *http.Request) error
	Callback(w http.ResponseWriter, r *http.Request) error
	Link(w http.ResponseWriter, r *http.Request) error
	Identities(w http.ResponseWriter, r *http.Request) error
	Detach(w http.ResponseWriter, r *http.Request) error
	Invite(w http.ResponseWriter, r *http.Request) error
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
	invitationRepo   repository.InvitationRepository
	authSvc          *auth.Service
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
	invitationRepo repository.InvitationRepository,
	authSvc *auth.Service,
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
		invitationRepo:   invitationRepo,
		authSvc:          authSvc,
	}
}

// Register registers the auth routes.
func (h *Auth) Register(router chi.Router) {
	h.logger.Debug("registering routes")

	router.Route("/auth", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Get("/", Wrap(h.logger, h.Initiate))
			r.Get("/callback", Wrap(h.logger, h.Callback))
			r.Get("/invite", Wrap(h.logger, h.Invite))
		})

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(h.logger, h.verifier, h.identityResolver))
			r.Get("/me", Wrap(h.logger, h.Me))
			r.Get("/link", Wrap(h.logger, h.Link))
			r.Get("/identities", Wrap(h.logger, h.Identities))
			r.Delete("/identities/{provider}", Wrap(h.logger, h.Detach))
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

	switch flow.Intent {
	case model.IntentAttach:
		return h.finishAttach(w, r, flow, claims)
	case model.IntentRedeem:
		return h.finishRedeem(w, r, flow, claims, rawToken)
	case model.IntentSignIn:
	}

	if _, err = h.resolveAndSync(r.Context(), claims); err != nil {
		redirectWithReason(w, r, redirect, signInReason(err))

		return err
	}

	setAuthCookie(w, rawToken, h.cfg.Auth.SessionTTL)
	//nolint:gosec // G710: validateRedirect already matched the target against the allowed list.
	http.Redirect(w, r, redirect, http.StatusFound)

	return nil
}

// Link starts an attach of a further external account to the acting user.
func (h *Auth) Link(w http.ResponseWriter, r *http.Request) error {
	redirect := r.URL.Query().Get("redirect")
	if !validateRedirect(redirect, h.cfg.Auth.AllowedRedirects) {
		sendBadRequest(w, r, "missing or invalid redirect parameter")

		return fmt.Errorf("%w: missing or invalid redirect parameter", errInvalidQueryParameter)
	}

	provider := r.URL.Query().Get("provider")
	if provider == "" {
		sendBadRequest(w, r, "missing provider parameter")

		return fmt.Errorf("%w: missing provider parameter", errInvalidQueryParameter)
	}

	userID := auth.UserIDFromContext(r.Context())

	authURL, binding, err := h.oidcFlow.Initiate(r.Context(), model.AuthFlow{
		Intent:   model.IntentAttach,
		UserID:   &userID,
		Provider: &provider,
		Redirect: redirect,
	})
	if err != nil {
		redirectWithError(w, r, redirect)

		return fmt.Errorf("initiating the attach: %w", err)
	}

	setBindingCookie(w, binding)
	http.Redirect(w, r, authURL, http.StatusFound)

	return nil
}

// providerState is one member of the list that `GET /auth/identities` returns.
//
// The four members below Attached carry the profile that the provider gave at the
// last sign in with it, and they are absent when nothing is attached.
type providerState struct {
	Provider string `json:"provider"`
	Name     string `json:"name"`
	Attached bool   `json:"attached"`

	Username    *string    `json:"username,omitempty"`
	DisplayName *string    `json:"display_name,omitempty"`
	PictureURL  *string    `json:"picture_url,omitempty"`
	AttachedAt  *time.Time `json:"attached_at,omitempty"`
}

// Identities reports every provider that Maroid offers, attached or not.
func (h *Auth) Identities(w http.ResponseWriter, r *http.Request) error {
	identities, err := h.identityRepo.ListByUser(
		r.Context(),
		auth.UserIDFromContext(r.Context()),
	)
	if err != nil {
		return fmt.Errorf("listing the identities of the acting user: %w", err)
	}

	attached := make(map[string]model.Identity, len(identities))
	for _, identity := range identities {
		attached[identity.Provider] = identity
	}

	states := make([]providerState, 0, len(h.cfg.Auth.Providers))
	for _, provider := range h.cfg.Auth.Providers {
		state := providerState{Provider: provider.ID, Name: provider.Name}

		if identity, ok := attached[provider.ID]; ok {
			state.Attached = true
			state.Username = identity.Username
			state.DisplayName = identity.DisplayName
			state.PictureURL = identity.PictureURL
			state.AttachedAt = &identity.CreatedAt
		}

		states = append(states, state)
	}

	render.JSON(w, r, states)

	return nil
}

// Detach removes an external account from the acting user.
func (h *Auth) Detach(w http.ResponseWriter, r *http.Request) error {
	provider := chi.URLParam(r, "provider")

	err := h.authSvc.Detach(r.Context(), auth.UserIDFromContext(r.Context()), provider)

	switch {
	case err == nil:
		render.NoContent(w, r)
	case errors.Is(err, errs.ErrLastIdentity):
		render.Status(r, http.StatusConflict)
		render.JSON(w, r, map[string]string{
			errorKey: "the last external account cannot be detached",
		})
	case errors.Is(err, errs.ErrIdentityNotFound):
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{errorKey: "not found"})
	default:
		return fmt.Errorf("detaching the external account: %w", err)
	}

	return nil
}

// Invite starts the redemption of an invitation.
func (h *Auth) Invite(w http.ResponseWriter, r *http.Request) error {
	redirect := r.URL.Query().Get("redirect")
	if !validateRedirect(redirect, h.cfg.Auth.AllowedRedirects) {
		sendBadRequest(w, r, "missing or invalid redirect parameter")

		return fmt.Errorf("%w: missing or invalid redirect parameter", errInvalidQueryParameter)
	}

	digest := sha256.Sum256([]byte(r.URL.Query().Get("token")))

	invitation, err := h.invitationRepo.GetValidByTokenHash(r.Context(), digest[:])
	if err != nil {
		redirectWithReason(w, r, redirect, reasonInvitationInvalid)

		return fmt.Errorf("reading the invitation: %w", err)
	}

	authURL, binding, err := h.oidcFlow.Initiate(r.Context(), model.AuthFlow{
		Intent:       model.IntentRedeem,
		InvitationID: &invitation.ID,
		Redirect:     redirect,
	})
	if err != nil {
		redirectWithError(w, r, redirect)

		return fmt.Errorf("initiating the redemption: %w", err)
	}

	setBindingCookie(w, binding)
	http.Redirect(w, r, authURL, http.StatusFound)

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

// finishAttach binds the external account to the record that the flow row names.
func (h *Auth) finishAttach(
	w http.ResponseWriter,
	r *http.Request,
	flow *model.AuthFlow,
	claims *auth.Claims,
) error {
	federated := claims.Federated
	if federated.ConnectorID == "" || federated.UserID == "" {
		redirectWithError(w, r, flow.Redirect)

		return errMissingFederatedClaims
	}

	err := h.authSvc.Attach(
		r.Context(),
		*flow.UserID,
		federated.ConnectorID,
		federated.UserID,
		model.Profile{
			Username:    claims.Username,
			DisplayName: claims.Name,
			PictureURL:  claims.Picture,
		},
	)
	if errors.Is(err, errs.ErrIdentityTaken) {
		redirectWithReason(w, r, flow.Redirect, reasonIdentityTaken)

		return nil
	}

	if err != nil {
		redirectWithError(w, r, flow.Redirect)

		return fmt.Errorf("attaching the external account: %w", err)
	}

	//nolint:gosec // G710: validateRedirect already matched the target against the allowed list.
	http.Redirect(w, r, flow.Redirect, http.StatusFound)

	return nil
}

// finishRedeem spends the invitation that the flow row names and signs the person
// in with the identity that the redemption just wrote.
func (h *Auth) finishRedeem(
	w http.ResponseWriter,
	r *http.Request,
	flow *model.AuthFlow,
	claims *auth.Claims,
	rawToken string,
) error {
	federated := claims.Federated
	if federated.ConnectorID == "" || federated.UserID == "" {
		redirectWithError(w, r, flow.Redirect)

		return errMissingFederatedClaims
	}

	_, err := h.authSvc.Redeem(
		r.Context(),
		*flow.InvitationID,
		federated.ConnectorID,
		federated.UserID,
		model.Profile{
			Username:    claims.Username,
			DisplayName: claims.Name,
			PictureURL:  claims.Picture,
		},
	)
	if errors.Is(err, errs.ErrIdentityTaken) {
		redirectWithReason(w, r, flow.Redirect, reasonIdentityTaken)

		return nil
	}

	if err != nil {
		redirectWithError(w, r, flow.Redirect)

		return fmt.Errorf("redeeming the invitation: %w", err)
	}

	setAuthCookie(w, rawToken, h.cfg.Auth.SessionTTL)
	//nolint:gosec // G710: validateRedirect already matched the target against the allowed list.
	http.Redirect(w, r, flow.Redirect, http.StatusFound)

	return nil
}

// resolveAndSync reads the user record that the external account names, then
// writes the profile that the provider gave onto that identity.
func (h *Auth) resolveAndSync(ctx context.Context, claims *auth.Claims) (*model.User, error) {
	federated := claims.Federated
	if federated.ConnectorID == "" || federated.UserID == "" {
		return nil, errMissingFederatedClaims
	}

	user, err := h.identityRepo.GetUserByProvider(ctx, federated.ConnectorID, federated.UserID)
	if err != nil {
		return nil, fmt.Errorf("resolving the user record of the external account: %w", err)
	}

	if user.Status != model.StatusActive {
		return nil, fmt.Errorf("%w: %s", errRecordNotActive, user.ID)
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

// signInReason names the failure of a sign in at the target of the flow.
func signInReason(err error) string {
	switch {
	case errors.Is(err, errs.ErrUserNotFound):
		return reasonNoIdentity
	case errors.Is(err, errRecordNotActive):
		return reasonAccessDenied
	default:
		return reasonAuthFailed
	}
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
func redirectWithError(w http.ResponseWriter, r *http.Request, target string) {
	redirectWithReason(w, r, target, reasonAuthFailed)
}

// redirectWithReason sends the caller back to the target with a named failure.
// Every caller passes a target that validateRedirect already accepted.
func redirectWithReason(w http.ResponseWriter, r *http.Request, target string, reason string) {
	redirectURL, _ := url.Parse(target)

	queryParams := redirectURL.Query()
	queryParams.Set("error", reason)
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
	render.JSON(w, r, map[string]string{errorKey: reason})
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
