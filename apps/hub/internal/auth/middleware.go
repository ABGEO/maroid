package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/render"
	"github.com/google/uuid"

	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	//nolint:gosec // G101: this names the cookie, it holds no credential.
	authTokenCookieName = "maroid_token"
	authHeaderName      = "Authorization"
)

var errMissingToken = errors.New("auth: the request carries no token")

type contextKey string

const (
	tokenContextKey contextKey = "token"
	claimsKey       contextKey = "claims"
	userIDKey       contextKey = "user_id"
)

// Middleware returns a HTTP middleware that enforces JWT authentication and
// resolves the acting user of the request.
func Middleware(
	logger *slog.Logger,
	jwtService *JWTService,
	userRepo repository.UserRepository,
) func(http.Handler) http.Handler {
	logger = logger.With(
		slog.String("component", "middleware"),
		slog.String("middleware", "auth"),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := tokenFromRequest(r)

			claims, user, err := resolve(r.Context(), logger, jwtService, userRepo, tokenString)
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
	jwtService *JWTService,
	userRepo repository.UserRepository,
	tokenString string,
) (*Claims, *model.User, error) {
	if tokenString == "" {
		logger.WarnContext(ctx, "missing auth token")

		return nil, nil, errMissingToken
	}

	claims, err := jwtService.Verify(tokenString)
	if err != nil {
		logger.ErrorContext(ctx, "invalid token", slog.Any("error", err))

		return nil, nil, fmt.Errorf("verifying the token: %w", err)
	}

	if err = uuid.Validate(claims.Subject); err != nil {
		logger.ErrorContext(
			ctx,
			"invalid subject claim in token",
			slog.String("subject", claims.Subject),
			slog.Any("error", err),
		)

		return nil, nil, fmt.Errorf("validating the subject claim: %w", err)
	}

	user, err := userRepo.GetActiveByID(ctx, claims.Subject)
	if err != nil {
		logger.InfoContext(
			ctx,
			"no active user record holds the subject",
			slog.String("subject", claims.Subject),
			slog.Any("error", err),
		)

		return nil, nil, fmt.Errorf("reading the user record: %w", err)
	}

	return claims, user, nil
}

func tokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(authTokenCookieName)
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	header := r.Header.Get(authHeaderName)
	if header != "" {
		return strings.TrimPrefix(header, "Bearer ")
	}

	return ""
}

func sendAccessDeniedResponse(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusUnauthorized)
	render.JSON(w, r, map[string]string{"error": "access denied"})
}

// TokenFromContext retrieves the JWT token from the context.
func TokenFromContext(ctx context.Context) string {
	token, _ := ctx.Value(tokenContextKey).(string)

	return token
}

// ClaimsFromContext retrieves the JWT claims from the context.
func ClaimsFromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsKey).(*Claims)

	return claims
}

// UserIDFromContext retrieves the Maroid user identifier from the context.
func UserIDFromContext(ctx context.Context) string {
	userID, _ := ctx.Value(userIDKey).(string)

	return userID
}
