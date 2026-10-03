package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const listWorkspacesName = "list_workspaces"

// ListWorkspacesInput carries no argument.
type ListWorkspacesInput struct{}

// WorkspaceEntry names one workspace of the acting user.
type WorkspaceEntry struct {
	ID   string `json:"id"   jsonschema:"the identifier that a call names as its workspace"`
	Name string `json:"name" jsonschema:"the name of the workspace"`
}

// ListWorkspacesOutput reports every workspace of the acting user.
type ListWorkspacesOutput struct {
	Workspaces []WorkspaceEntry `json:"workspaces" jsonschema:"every workspace that the acting user is a member of"`
}

// listWorkspaces holds the repository that the report reads.
type listWorkspaces struct {
	workspaces repository.WorkspaceRepository
}

// NewListWorkspaces builds the tool that names the workspaces of the acting user.
func NewListWorkspaces(workspaces repository.WorkspaceRepository) registry.MCPTool {
	tool := &listWorkspaces{workspaces: workspaces}

	return registry.MCPTool{
		Name: listWorkspacesName,
		Install: func(server *mcp.Server) {
			mcp.AddTool(server, &mcp.Tool{
				Name:  listWorkspacesName,
				Title: "List the workspaces",
				Description: "Report every workspace that the acting user is a member of. " +
					"A tool that acts in a workspace takes one of these identifiers.",
				Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
			}, tool.handle)
		},
	}
}

func (t *listWorkspaces) handle(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ ListWorkspacesInput,
) (*mcp.CallToolResult, ListWorkspacesOutput, error) {
	memberships, err := t.workspaces.ListOfUser(ctx, pluginapi.ActingUserFromContext(ctx))
	if err != nil {
		return nil, ListWorkspacesOutput{}, fmt.Errorf("listing the workspaces: %w", err)
	}

	entries := make([]WorkspaceEntry, 0, len(memberships))
	for _, membership := range memberships {
		entries = append(entries, WorkspaceEntry{ID: membership.ID, Name: membership.Name})
	}

	return nil, ListWorkspacesOutput{Workspaces: entries}, nil
}
