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
	"github.com/abgeo/maroid/apps/hub/internal/domain/problems"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/rest"
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
	Logout(w http.ResponseWriter, r *http.Request) error
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
			r.Post("/logout", Wrap(h.logger, h.Logout))
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

	auth.SetBindingCookie(w, binding)
	http.Redirect(w, r, authURL, http.StatusFound)

	return nil
}

// Callback completes the OIDC flow, verifies the ID token, and finishes the
// sign in, the attach, or the redemption that the flow row names.
func (h *Auth) Callback(w http.ResponseWriter, r *http.Request) error {
	state := r.URL.Query().Get("state")

	binding := auth.TakeBindingCookie(w, r)

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

	session, err := h.processOIDCCallback(r, flow)
	if err != nil {
		redirectWithError(w, r, redirect)

		return err
	}

	switch flow.Intent {
	case model.IntentAttach:
		return h.finishAttach(w, r, flow, session.Claims)
	case model.IntentRedeem:
		return h.finishRedeem(w, r, flow, session)
	case model.IntentSignIn:
	}

	if _, err = h.resolveAndSync(r.Context(), session.Claims); err != nil {
		redirectWithReason(w, r, redirect, signInReason(err))

		return err
	}

	auth.SetSessionCookie(w, session.AccessToken, session.Expiry)
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

	auth.SetBindingCookie(w, binding)
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
			// encoding/json writes a time.Time in the zone that it carries, and
			// the database answers the zone of the session.
			attachedAt := identity.CreatedAt.UTC()
			state.AttachedAt = &attachedAt
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
		rest.Write(w, r, problems.NewIdentityLast())
	case errors.Is(err, errs.ErrIdentityNotFound):
		rest.Write(w, r, rest.NewNotFound())
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

	auth.SetBindingCookie(w, binding)
	http.Redirect(w, r, authURL, http.StatusFound)

	return nil
}

// logoutResponse is the body of POST /auth/logout.
type logoutResponse struct {
	Redirect string `json:"redirect"`
}

// Logout ends the session of the person at Maroid.
//
// The route runs behind no access check, so a second call answers
// as the first one did. The session at the IdP survives, so a sign in that follows
// needs no question from that service.
func (h *Auth) Logout(w http.ResponseWriter, r *http.Request) error {
	target := r.URL.Query().Get("redirect")
	if target == "" {
		target = h.cfg.Auth.DeckURL
	}

	if !validateRedirect(target, h.cfg.Auth.AllowedRedirects) {
		sendBadRequest(w, r, "missing or invalid redirect parameter")

		return fmt.Errorf("%w: missing or invalid redirect parameter", errInvalidQueryParameter)
	}

	auth.ClearSessionCookie(w)
	render.JSON(w, r, logoutResponse{Redirect: target})

	return nil
}

// meResponse is the body of GET /auth/me.
type meResponse struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Picture   string  `json:"picture"`
	Provider  string  `json:"provider"`
}

// Me returns the name of the acting user, the picture of the session, and the
// provider that authenticated it.
func (h *Auth) Me(w http.ResponseWriter, r *http.Request) error {
	claims := auth.ClaimsFromContext(r.Context())

	user, err := h.userRepo.GetActiveByID(r.Context(), auth.UserIDFromContext(r.Context()))
	if err != nil {
		return fmt.Errorf("reading the acting user: %w", err)
	}

	render.JSON(w, r, meResponse{
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Picture:   claims.Picture,
		Provider:  claims.Federated.ConnectorID,
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
	session *auth.Session,
) error {
	claims := session.Claims
	federated := claims.Federated

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

	auth.SetSessionCookie(w, session.AccessToken, session.Expiry)
	//nolint:gosec // G710: validateRedirect already matched the target against the allowed list.
	http.Redirect(w, r, flow.Redirect, http.StatusFound)

	return nil
}

// resolveAndSync reads the user record that the external account names, then
// writes the profile that the provider gave onto that identity.
func (h *Auth) resolveAndSync(ctx context.Context, claims *auth.Claims) (*model.User, error) {
	federated := claims.Federated

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

// processOIDCCallback exchanges the code, verifies the ID token, and confirms
// that the claims carry the federated account that every intent needs.
func (h *Auth) processOIDCCallback(
	r *http.Request,
	flow *model.AuthFlow,
) (*auth.Session, error) {
	code := r.URL.Query().Get("code")
	if code == "" {
		return nil, fmt.Errorf("%w: missing code query parameter", errInvalidQueryParameter)
	}

	session, err := h.oidcFlow.Verify(r.Context(), code, flow.Nonce, flow.Verifier)
	if err != nil {
		return nil, fmt.Errorf("verifying OIDC flow: %w", err)
	}

	if session.Claims.Federated.ConnectorID == "" || session.Claims.Federated.UserID == "" {
		return nil, errMissingFederatedClaims
	}

	return session, nil
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

// sendBadRequest answers with the problem of a request that a route cannot read.
// The detail names the parameter and nothing of the request.
func sendBadRequest(w http.ResponseWriter, r *http.Request, detail string) {
	rest.Write(w, r, rest.NewRequestInvalid().WithDetail(detail))
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
