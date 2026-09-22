package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/problem"
)

var (
	errMissingToken          = errors.New("auth: the request carries no token")
	errMissingFederatedClaim = errors.New("auth: the token carries no federated claims")
)

type contextKey string

const (
	tokenContextKey contextKey = "token"
	claimsKey       contextKey = "claims"
	userIDKey       contextKey = "user_id"
)

// Middleware returns a HTTP middleware that verifies a access token and resolves
// the acting user of the request.
func Middleware(
	logger *slog.Logger,
	verifier TokenVerifier,
	resolver IdentityResolver,
) func(http.Handler) http.Handler {
	logger = logger.With(
		slog.String("component", "middleware"),
		slog.String("middleware", "auth"),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := SessionCookie(r)

			claims, user, err := resolve(r.Context(), logger, verifier, resolver, tokenString)
			if err != nil {
				sendAccessDeniedResponse(w, r)

				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, tokenContextKey, tokenString)
			ctx = context.WithValue(ctx, userIDKey, user.ID)
			ctx = context.WithValue(ctx, claimsKey, claims)
			ctx = pluginapi.ContextWithActingUser(ctx, user.ID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// resolve reads the acting user of the request, and reports why it refused.
func resolve(
	ctx context.Context,
	logger *slog.Logger,
	verifier TokenVerifier,
	resolver IdentityResolver,
	tokenString string,
) (*Claims, *model.User, error) {
	if tokenString == "" {
		logger.WarnContext(ctx, "missing auth token")

		return nil, nil, errMissingToken
	}

	claims, err := verifier.Verify(ctx, tokenString)
	if err != nil {
		logger.ErrorContext(ctx, "invalid token", slog.Any("error", err))

		return nil, nil, fmt.Errorf("verifying the token: %w", err)
	}

	if claims.Federated.ConnectorID == "" || claims.Federated.UserID == "" {
		logger.ErrorContext(
			ctx,
			"the token carries no federated claims",
			slog.String("subject", claims.Subject),
		)

		return nil, nil, errMissingFederatedClaim
	}

	user, err := resolver.ResolveByProvider(
		ctx,
		claims.Federated.ConnectorID,
		claims.Federated.UserID,
	)
	if err != nil {
		logger.InfoContext(
			ctx,
			"no active user record holds the external account",
			slog.String("provider", claims.Federated.ConnectorID),
			slog.Any("error", err),
		)

		return nil, nil, fmt.Errorf("resolving the identity: %w", err)
	}

	return claims, user, nil
}

func sendAccessDeniedResponse(w http.ResponseWriter, r *http.Request) {
	problem.Write(w, r, problem.NewAccessDenied())
}

// TokenFromContext retrieves the access token from the context.
func TokenFromContext(ctx context.Context) string {
	token, _ := ctx.Value(tokenContextKey).(string)

	return token
}

// ClaimsFromContext retrieves the claims of the token from the context.
func ClaimsFromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsKey).(*Claims)

	return claims
}

// UserIDFromContext retrieves the Maroid user identifier from the context.
func UserIDFromContext(ctx context.Context) string {
	userID, _ := ctx.Value(userIDKey).(string)

	return userID
}
