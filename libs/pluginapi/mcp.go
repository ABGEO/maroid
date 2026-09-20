package pluginapi

import (
	"context"
	"encoding/json"
	"fmt"
)

// MCPToolPlugin is a plugin that exposes a function over the Model Context Protocol.
type MCPToolPlugin interface {
	Plugin
	MCPTools() ([]MCPTool, error)
}

// MCPTool is one function that a plugin exposes over the Model Context Protocol.
//
// The hub validates the input against the schema of Meta().InputModel before it
// calls Handle, so Handle receives a document that the schema accepts.
type MCPTool interface {
	Meta() MCPToolMeta
	Handle(ctx context.Context, input json.RawMessage) (any, error)
}

// MCPToolMeta describes one tool to an MCP client. InputModel and OutputModel are
// each a struct value, and the hub reflects the JSON schema of each one.
type MCPToolMeta struct {
	Name        string
	Title       string
	Description string
	Annotations MCPToolAnnotations
	InputModel  any
	OutputModel any
}

// MCPToolAnnotations tells an MCP client what calling a tool does, so an agent
// can ask a person before it acts. Every member is a hint. The hub reports what
// the plugin declared and verifies none of it.
//
// The zero value describes the tool that needs the most care: one that writes,
// that may destroy, and that reaches an open world. A plugin that declares
// nothing therefore gets the answer that makes an agent ask first.
type MCPToolAnnotations struct {
	// ReadOnlyHint marks a tool that changes nothing. Default false.
	ReadOnlyHint bool
	// DestructiveHint marks a tool that may destroy rather than only add. It
	// means something only when ReadOnlyHint is false. Nil leaves the default of
	// the protocol, which is true.
	DestructiveHint *bool
	// IdempotentHint marks a tool that a second call with the same arguments
	// leaves alone. It means something only when ReadOnlyHint is false.
	// Default false.
	IdempotentHint bool
	// OpenWorldHint marks a tool that reaches an external world, the way a search
	// does and a stored note does not. Nil leaves the default of the protocol,
	// which is true.
	OpenWorldHint *bool
}

// TypedTool binds one handler to the Go types that it reads and answers with.
// NewTypedTool builds it.
type TypedTool[In, Out any] struct {
	meta   MCPToolMeta
	handle func(ctx context.Context, input In) (Out, error)
}

var _ MCPTool = (*TypedTool[struct{}, struct{}])(nil)

// NewTypedTool builds an MCPTool whose handler takes and returns its own Go types.
// It fills InputModel and OutputModel from the type arguments, so a caller
// declares each shape once.
func NewTypedTool[In, Out any](
	meta MCPToolMeta,
	handle func(ctx context.Context, input In) (Out, error),
) *TypedTool[In, Out] {
	var (
		input  In
		output Out
	)

	meta.InputModel = input
	meta.OutputModel = output

	return &TypedTool[In, Out]{meta: meta, handle: handle}
}

// Meta describes the tool to an MCP client.
func (t *TypedTool[In, Out]) Meta() MCPToolMeta {
	return t.meta
}

// Handle reads the arguments of the call into the input type of the tool.
func (t *TypedTool[In, Out]) Handle(ctx context.Context, input json.RawMessage) (any, error) {
	var decoded In

	if len(input) > 0 {
		if err := json.Unmarshal(input, &decoded); err != nil {
			return nil, fmt.Errorf("reading the arguments of the tool: %w", err)
		}
	}

	output, err := t.handle(ctx, decoded)
	if err != nil {
		return nil, err
	}

	return output, nil
}
