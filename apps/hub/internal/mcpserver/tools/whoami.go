package tools

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/maroid/apps/hub/internal/mcpserver"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

const whoAmIName = "whoami"

// WhoAmIInput carries no argument.
type WhoAmIInput struct{}

// WhoAmIOutput names the user that the call acts for.
type WhoAmIOutput struct {
	Name     string `json:"name"     jsonschema:"the name that the user record carries"`
	Provider string `json:"provider" jsonschema:"the provider that signed the current session in"`
}

// NewWhoAmI builds the identity tool: the name and the provider of the acting user.
func NewWhoAmI() registry.MCPTool {
	return registry.MCPTool{
		Name: whoAmIName,
		Install: func(server *mcp.Server) {
			mcp.AddTool(server, &mcp.Tool{
				Name:        whoAmIName,
				Title:       "Report the acting user",
				Description: "Report the name of the Maroid user that this session acts for, and the provider that signed them in.",
				Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
			}, handleWhoAmI)
		},
	}
}

func handleWhoAmI(
	_ context.Context,
	request *mcp.CallToolRequest,
	_ WhoAmIInput,
) (*mcp.CallToolResult, WhoAmIOutput, error) {
	info := request.Extra.TokenInfo

	return nil, WhoAmIOutput{
		Name:     fullName(mcpserver.UserFromTokenInfo(info)),
		Provider: mcpserver.ClaimsFromTokenInfo(info).Federated.ConnectorID,
	}, nil
}

// fullName joins the two halves of the name, and trims the space when one is empty.
func fullName(user *model.User) string {
	return strings.TrimSpace(nameHalf(user.FirstName) + " " + nameHalf(user.LastName))
}

func nameHalf(half *string) string {
	if half == nil {
		return ""
	}

	return *half
}
