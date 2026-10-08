package mcpserver

import (
	"log/slog"

	"github.com/jmoiron/sqlx"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
)

// The implementation that the hub reports to an MCP client at the initialization.
const (
	serverName    = "maroid"
	serverTitle   = "Maroid"
	serverVersion = "0.1.0"
)

// NewServer builds the Model Context Protocol server of the hub, with every tool
// of toolRegistry installed.
func NewServer(
	logger *slog.Logger,
	toolRegistry *registry.MCPToolRegistry,
	db *sqlx.DB,
	enablements workspace.EnablementChecker,
	authorizer authz.Authorizer,
) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Title:   serverTitle,
		Version: serverVersion,
	}, &mcp.ServerOptions{Logger: logger})

	tools := toolRegistry.All()

	server.AddReceivingMiddleware(
		loggingMiddleware(logger),
		actingUserMiddleware(),
		workspaceMiddleware(tools, db, enablements, authorizer),
	)

	for _, tool := range tools {
		tool.Install(server)
	}

	return server
}
