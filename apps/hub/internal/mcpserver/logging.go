package mcpserver

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// loggingMiddleware reports each method that the server received.
func loggingMiddleware(logger *slog.Logger) mcp.Middleware {
	logger = logger.With(
		slog.String("component", "middleware"),
		slog.String("middleware", "mcp"),
	)

	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			result, err := next(ctx, method, req)
			if err != nil {
				logger.ErrorContext(
					ctx,
					"the method failed",
					slog.String("method", method),
					slog.Any("error", err),
				)

				return result, err
			}

			logger.DebugContext(ctx, "method handled", slog.String("method", method))

			return result, nil
		}
	}
}
