package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// actingUserMiddleware puts the acting user of the verified token, and whether they
// are an administrator, into the context that every tool handler receives, as
// auth.Middleware does for a web request.
func actingUserMiddleware() mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if extra := req.GetExtra(); extra != nil {
				if user := UserFromTokenInfo(extra.TokenInfo); user != nil {
					ctx = auth.ContextWithAdministrator(
						pluginapi.ContextWithActingUser(ctx, user.ID),
						user.IsAdministrator,
					)
				}
			}

			return next(ctx, method, req)
		}
	}
}
