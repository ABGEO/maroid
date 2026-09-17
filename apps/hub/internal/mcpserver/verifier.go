package mcpserver

import (
	"context"
	"net/http"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/model"
)

// The members of TokenInfo.Extra that the verifier fills.
const (
	claimsExtraKey = "claims"
	userExtraKey   = "user"
)

// rejection is the one answer that every refusal gives. A failed verification and
// an identity with no active user record read the same, so a caller learns nothing
// about which account exists.
type rejection struct{}

// Error names the problem. The caller reads this text, so it holds no chain.
func (rejection) Error() string {
	return "mcp: the request carries no usable token"
}

// Unwrap makes the bearer token middleware of the SDK answer 401.
func (rejection) Unwrap() error {
	return mcpauth.ErrInvalidToken
}

var errRejected = rejection{}

// NewTokenVerifier builds a TokenVerifier that checks a bearer token against the
// key set of IdP, with clientID as the audience, then resolves the acting user of
// the identity it names.
func NewTokenVerifier(
	oidcSvc *auth.OIDCService,
	resolver auth.IdentityResolver,
	clientID string,
) mcpauth.TokenVerifier {
	verifier := oidcSvc.VerifierForClient(clientID)

	return func(
		ctx context.Context,
		rawToken string,
		_ *http.Request,
	) (*mcpauth.TokenInfo, error) {
		token, err := verifier.Verify(ctx, rawToken)
		if err != nil {
			return nil, errRejected
		}

		var claims auth.Claims
		if err = token.Claims(&claims); err != nil {
			return nil, errRejected
		}

		if claims.Federated.ConnectorID == "" || claims.Federated.UserID == "" {
			return nil, errRejected
		}

		user, err := resolver.ResolveByProvider(
			ctx,
			claims.Federated.ConnectorID,
			claims.Federated.UserID,
		)
		if err != nil {
			return nil, errRejected
		}

		return &mcpauth.TokenInfo{
			UserID:     user.ID,
			Expiration: token.Expiry,
			Extra: map[string]any{
				claimsExtraKey: &claims,
				userExtraKey:   user,
			},
		}, nil
	}
}

// ClaimsFromTokenInfo reads the claims of the token that the verifier checked.
func ClaimsFromTokenInfo(info *mcpauth.TokenInfo) *auth.Claims {
	if info == nil {
		return nil
	}

	claims, _ := info.Extra[claimsExtraKey].(*auth.Claims)

	return claims
}

// UserFromTokenInfo reads the acting user that the verifier resolved.
func UserFromTokenInfo(info *mcpauth.TokenInfo) *model.User {
	if info == nil {
		return nil
	}

	user, _ := info.Extra[userExtraKey].(*model.User)

	return user
}
