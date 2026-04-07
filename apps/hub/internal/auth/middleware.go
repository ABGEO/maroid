package auth

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/go-chi/render"
)

const (
	authTokenCookieName = "maroid_token"
	authHeaderName      = "Authorization"
)

type contextKey string

const (
	tokenContextKey contextKey = "token"
	claimsKey       contextKey = "claims"
	userIDKey       contextKey = "user_id"
)

// Middleware returns a HTTP middleware that enforces JWT authentication and user authorization.
func Middleware(
	logger *slog.Logger,
	jwtService *JWTService,
	allowedUsers []int64,
) func(http.Handler) http.Handler {
	logger = logger.With(
		slog.String("component", "middleware"),
		slog.String("middleware", "auth"),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := tokenFromRequest(r)
			if tokenString == "" {
				logger.Warn("missing auth token")
				sendAccessDeniedResponse(w, r)

				return
			}

			claims, err := jwtService.Verify(tokenString)
			if err != nil {
				logger.Error(
					"invalid token",
					slog.Any("error", err),
				)
				sendAccessDeniedResponse(w, r)

				return
			}

			userID, err := strconv.ParseInt(claims.Subject, 10, 64)
			if err != nil {
				logger.Error(
					"invalid subject claim in token",
					slog.String("subject", claims.Subject),
					slog.Any("error", err),
				)
				sendAccessDeniedResponse(w, r)

				return
			}

			if !slices.Contains(allowedUsers, userID) {
				logger.Info(
					"user is not allowed",
					slog.Int64("user_id", userID),
				)
				sendAccessDeniedResponse(w, r)

				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, tokenContextKey, tokenString)
			ctx = context.WithValue(ctx, userIDKey, userID)
			ctx = context.WithValue(ctx, claimsKey, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
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

// UserIDFromContext retrieves the user ID from the context.
func UserIDFromContext(ctx context.Context) int64 {
	userID, _ := ctx.Value(userIDKey).(int64)

	return userID
}
