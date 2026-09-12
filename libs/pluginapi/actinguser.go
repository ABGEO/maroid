package pluginapi

import "context"

type actingUserKey struct{}

// ContextWithActingUser returns a context that carries the user that the unit of
// work runs for.
func ContextWithActingUser(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, actingUserKey{}, userID)
}

// ActingUserFromContext returns the acting user of the context.
// It returns the empty string when the context carries none, and
// PluginDB.WithTx then reaches no scoped record. See OWN-006.
func ActingUserFromContext(ctx context.Context) string {
	userID, _ := ctx.Value(actingUserKey{}).(string)

	return userID
}
