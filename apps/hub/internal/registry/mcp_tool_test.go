package registry_test

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

func noInstall(*mcp.Server) {}

// PLG-011: A registry rejects a second registration under an identifier that it
// already holds, and returns a sentinel error. MCPHUB-DD-008 builds this registry
// so that a plugin's own tool later reaches the same Register.
func TestTheMCPToolRegistryRefusesATwiceNamedTool(t *testing.T) {
	t.Parallel()

	toolRegistry := registry.NewMCPToolRegistry()

	require.NoError(t, toolRegistry.Register(
		registry.MCPTool{Name: "ping", Install: noInstall},
		registry.MCPTool{Name: "whoami", Install: noInstall},
	))
	require.Len(t, toolRegistry.All(), 2)

	err := toolRegistry.Register(registry.MCPTool{Name: "ping", Install: noInstall})

	require.ErrorIs(t, err, errs.ErrMCPToolAlreadyRegistered)
	require.Len(t, toolRegistry.All(), 2)
}
