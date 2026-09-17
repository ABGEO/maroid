package registry

import (
	"fmt"
	"maps"
	"slices"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
)

// MCPTool is one function that Maroid exposes over the Model Context Protocol.
// Install adds it to a server, with its own input and output types. Two MCPTool
// values do not share a Name.
//
// Install holds the input and the output type of the tool inside its own closure,
// so mcp.AddTool still infers the JSON schema of the tool from the Go types while
// this registry stores one uniform value.
type MCPTool struct {
	Name    string
	Install func(server *mcp.Server)
}

// MCPToolRegistry is a registry for Model Context Protocol tools.
type MCPToolRegistry struct {
	tools map[string]MCPTool
}

// NewMCPToolRegistry creates a new MCPToolRegistry.
func NewMCPToolRegistry() *MCPToolRegistry {
	return &MCPToolRegistry{
		tools: make(map[string]MCPTool),
	}
}

// Register stores one or more tools under their names.
func (r *MCPToolRegistry) Register(tools ...MCPTool) error {
	for _, tool := range tools {
		if _, exists := r.tools[tool.Name]; exists {
			return fmt.Errorf("%w: %s", errs.ErrMCPToolAlreadyRegistered, tool.Name)
		}

		r.tools[tool.Name] = tool
	}

	return nil
}

// All returns every registered tool.
func (r *MCPToolRegistry) All() []MCPTool {
	return slices.Collect(maps.Values(r.tools))
}
