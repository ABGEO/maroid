package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// WorkspaceArgument names the member of the arguments that carries the workspace
// that a call acts in.
const WorkspaceArgument = "workspace"

const (
	callToolMethod       = "tools/call"
	workspaceDescription = "the identifier of the workspace that the call acts in, from list_workspaces"
)

// workspaceMiddleware reads the workspace of each call of a tool that acts in one,
// checks the membership of the acting user, and puts the workspace into the context.
// A call that names no workspace reaches the tool, whose schema then refuses it.
func workspaceMiddleware(
	tools []registry.MCPTool,
	members repository.WorkspaceMemberRepository,
) mcp.Middleware {
	inWorkspace := make(map[string]bool, len(tools))
	for _, tool := range tools {
		inWorkspace[tool.Name] = tool.ActsInWorkspace
	}

	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			call, ok := req.(*mcp.CallToolRequest)
			if method != callToolMethod || !ok || !inWorkspace[call.Params.Name] {
				return next(ctx, method, req)
			}

			workspaceID, named := workspaceOf(call.Params.Arguments)
			if !named {
				return next(ctx, method, req)
			}

			if !isMember(ctx, members, workspaceID) {
				return nil, unknownTool(call.Params.Name)
			}

			return next(pluginapi.ContextWithActingWorkspace(ctx, workspaceID), method, req)
		}
	}
}

// workspaceOf reads the workspace member of the arguments.
func workspaceOf(arguments json.RawMessage) (string, bool) {
	var named map[string]any

	if json.Unmarshal(arguments, &named) != nil {
		return "", false
	}

	workspaceID, ok := named[WorkspaceArgument].(string)

	return workspaceID, ok
}

func isMember(
	ctx context.Context,
	members repository.WorkspaceMemberRepository,
	workspaceID string,
) bool {
	if uuid.Validate(workspaceID) != nil {
		return false
	}

	_, err := members.Get(ctx, workspaceID, pluginapi.ActingUserFromContext(ctx))

	return err == nil
}

// unknownTool is the error that the SDK answers for a tool it does not hold, so a
// workspace of another person reads as a tool that does not exist.
func unknownTool(name string) error {
	return &jsonrpc.Error{
		Code:    jsonrpc.CodeInvalidParams,
		Message: fmt.Sprintf("unknown tool %q", name),
	}
}

// withWorkspaceMember adds the required member workspace to the input schema. A
// model that declares the member itself fails, because the hub owns it.
func withWorkspaceMember(schema json.RawMessage) (json.RawMessage, error) {
	var document map[string]any

	if err := json.Unmarshal(schema, &document); err != nil {
		return nil, fmt.Errorf("reading the input schema: %w", err)
	}

	properties, _ := document["properties"].(map[string]any)
	if properties == nil {
		properties = map[string]any{}
	}

	if _, declared := properties[WorkspaceArgument]; declared {
		return nil, fmt.Errorf(
			"%w: the model declares the member %q, which the hub adds",
			errs.ErrInvalidMCPToolModel, WorkspaceArgument,
		)
	}

	properties[WorkspaceArgument] = map[string]any{
		"type":        "string",
		"format":      "uuid",
		"description": workspaceDescription,
	}
	document["properties"] = properties

	required, _ := document["required"].([]any)
	document["required"] = append(required, WorkspaceArgument)

	encoded, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encoding the input schema: %w", err)
	}

	return encoded, nil
}

// withoutWorkspaceMember removes the member workspace, so the plugin receives the
// arguments of the model that it declared.
func withoutWorkspaceMember(arguments json.RawMessage) json.RawMessage {
	var named map[string]json.RawMessage

	if json.Unmarshal(arguments, &named) != nil {
		return arguments
	}

	delete(named, WorkspaceArgument)

	stripped, err := json.Marshal(named)
	if err != nil {
		return arguments
	}

	return stripped
}
