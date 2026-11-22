package pluginapi

import (
	"context"
	"time"
)

type (
	actingUserKey      struct{}
	actingWorkspaceKey struct{}
	actingTimeZoneKey  struct{}
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

// ContextWithActingTimeZone returns a context that carries the time zone of the
// acting user. The hub sets it from the record of that user.
func ContextWithActingTimeZone(ctx context.Context, location *time.Location) context.Context {
	return context.WithValue(ctx, actingTimeZoneKey{}, location)
}

// ActingTimeZone returns the time zone of the acting user of the context, to compute
// a local hour or a local day of that person. It returns UTC when the record holds
// no zone, or when the unit of work has no acting user.
func ActingTimeZone(ctx context.Context) *time.Location {
	location, _ := ctx.Value(actingTimeZoneKey{}).(*time.Location)
	if location == nil {
		return time.UTC
	}

	return location
}
