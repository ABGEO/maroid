package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

const pingName = "ping"

// PingInput carries no argument.
type PingInput struct{}

// PingOutput confirms that the call reached the hub.
type PingOutput struct {
	Message string `json:"message" jsonschema:"the answer of the hub"`
}

// NewPing builds the connectivity tool.
func NewPing() registry.MCPTool {
	return registry.MCPTool{
		Name: pingName,
		Install: func(server *mcp.Server) {
			mcp.AddTool(server, &mcp.Tool{
				Name:        pingName,
				Title:       "Ping the hub",
				Description: "Confirm that Maroid is reachable and that this session is authenticated.",
				Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
			}, handlePing)
		},
	}
}

func handlePing(
	_ context.Context,
	_ *mcp.CallToolRequest,
	_ PingInput,
) (*mcp.CallToolResult, PingOutput, error) {
	return nil, PingOutput{Message: "pong"}, nil
}
