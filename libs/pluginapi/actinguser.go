package pluginapi

import "context"

type (
	actingUserKey      struct{}
	actingWorkspaceKey struct{}
)

// ContextWithActingUser returns a context that carries the user that the unit of
// work runs for.
func ContextWithActingUser(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, actingUserKey{}, userID)
}

// ActingUserFromContext returns the acting user of the context.
// It returns the empty string when the context carries none, and
// PluginDB.WithTx then reaches no record that a user scopes.
func ActingUserFromContext(ctx context.Context) string {
	userID, _ := ctx.Value(actingUserKey{}).(string)

	return userID
}

// ContextWithActingWorkspace returns a context that carries the workspace that the
// unit of work runs in.
func ContextWithActingWorkspace(ctx context.Context, workspaceID string) context.Context {
	return context.WithValue(ctx, actingWorkspaceKey{}, workspaceID)
}

// ActingWorkspaceFromContext returns the acting workspace of the context.
// It returns the empty string when the context carries none, and
// PluginDB.WithTx then reaches no record that a workspace scopes.
func ActingWorkspaceFromContext(ctx context.Context) string {
	workspaceID, _ := ctx.Value(actingWorkspaceKey{}).(string)

	return workspaceID
}
