package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
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
// checks the membership of the acting user and the permission of the tool, and puts
// the workspace and the role into the context. A call that names no workspace reaches
// the tool, whose schema then refuses it.
func workspaceMiddleware(
	tools []registry.MCPTool,
	members repository.WorkspaceMemberRepository,
	enablements workspace.EnablementChecker,
	authorizer authz.Authorizer,
) mcp.Middleware {
	gate := toolGate{members: members, enablements: enablements, authorizer: authorizer}
	inWorkspace := make(map[string]registry.MCPTool, len(tools))

	for _, tool := range tools {
		if tool.ActsInWorkspace {
			inWorkspace[tool.Name] = tool
		}
	}

	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			call, ok := req.(*mcp.CallToolRequest)
			if method != callToolMethod || !ok {
				return next(ctx, method, req)
			}

			tool, guarded := inWorkspace[call.Params.Name]
			if !guarded {
				return next(ctx, method, req)
			}

			workspaceID, named := workspaceOf(call.Params.Arguments)
			if !named {
				return next(ctx, method, req)
			}

			admitted, refused, err := gate.pass(ctx, tool, workspaceID)
			if err != nil {
				return nil, err
			}

			if refused != nil {
				return refused, nil
			}

			return next(admitted, method, req)
		}
	}
}

// toolGate holds the three checks of a call of a tool in a workspace: the membership,
// the enablement of the plugin of the tool, and the permission of the tool.
type toolGate struct {
	members     repository.WorkspaceMemberRepository
	enablements workspace.EnablementChecker
	authorizer  authz.Authorizer
}

// pass answers the context of a call that passes every check, with the workspace and
// the role in it. A call that fails answers either the error of an unknown tool, for a
// workspace of no membership or a plugin it does not enable, or a refusal that names
// the permission.
func (gate toolGate) pass(
	ctx context.Context,
	tool registry.MCPTool,
	workspaceID string,
) (context.Context, *mcp.CallToolResult, error) {
	role, member := roleIn(ctx, gate.members, workspaceID)
	if !member || !enabledIn(ctx, gate.enablements, workspaceID, tool.PluginID) {
		return nil, nil, unknownTool(tool.Name)
	}

	allowed, lowest, err := gate.authorizer.Allowed(role, tool.Permission)
	if err != nil {
		return nil, nil, fmt.Errorf("checking the permission of the tool: %w", err)
	}

	if !allowed {
		return nil, refusal(tool.Permission, lowest), nil
	}

	return workspace.ContextWithRole(
		pluginapi.ContextWithActingWorkspace(ctx, workspaceID),
		role,
	), nil, nil
}

// refusal is the result of a call whose role does not hold the permission of the
// tool. A member knows the workspace, so the result names what they lack.
func refusal(permission string, lowest pluginapi.Role) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: authz.Refusal(permission, lowest)}},
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

func roleIn(
	ctx context.Context,
	members repository.WorkspaceMemberRepository,
	workspaceID string,
) (pluginapi.Role, bool) {
	if uuid.Validate(workspaceID) != nil {
		return "", false
	}

	member, err := members.Get(ctx, workspaceID, pluginapi.ActingUserFromContext(ctx))
	if err != nil {
		return "", false
	}

	return member.Role, true
}

// enabledIn reports whether the workspace enables the plugin of the tool. A tool of the
// hub names no plugin. A failed read answers as a plugin that the workspace does not
// enable, so the call reaches no record.
func enabledIn(
	ctx context.Context,
	enablements workspace.EnablementChecker,
	workspaceID string,
	pluginID string,
) bool {
	if pluginID == "" {
		return true
	}

	enabled, err := enablements.IsEnabled(ctx, workspaceID, pluginID)

	return err == nil && enabled
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
