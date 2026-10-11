package repository_test

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	reminderTopic = "reminder"
	chatField     = "chat_id"
)

func createProvider(t *testing.T, instance *testdb.Instance, user string, workspace string) string {
	t.Helper()

	var id string

	require.NoError(
		t,
		inScope(t, instance, user, workspace, func(ctx context.Context, tx *sqlx.Tx) error {
			stored, err := repository.NewNotificationProvider(tx).
				Create(ctx, &model.NotificationProvider{
					Kind: "telegram",
					Name: "Family bot",
					Fields: model.Fields{
						"bot_token": {Kind: model.FieldKindSecret, Value: "protected"},
					},
					Enabled: true,
				})
			if err != nil {
				return err //nolint:wrapcheck // the test reports the error as it is
			}

			id = stored.ID

			return nil
		}),
	)

	return id
}

func createRoute(
	t *testing.T,
	instance *testdb.Instance,
	user string,
	workspace string,
	route *model.NotificationRoute,
) error {
	t.Helper()

	return inScope(t, instance, user, workspace, func(ctx context.Context, tx *sqlx.Tx) error {
		_, err := repository.NewNotificationRoute(tx).Create(ctx, route)

		return err //nolint:wrapcheck // the test reads the error that the repository answered
	})
}

// NOTIF-SC-028: A route names a provider of its own workspace only, and a read in
// one workspace answers no row of another.
func TestNotificationRowsStayInTheirWorkspace(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	userA := insertUser(t, instance, nameOfA)
	userB := insertUser(t, instance, "Nino")
	workspaceA := createWorkspace(t, instance, "A", userA).ID
	workspaceB := createWorkspace(t, instance, "B", userB).ID

	providerA := createProvider(t, instance, userA, workspaceA)
	providerB := createProvider(t, instance, userB, workspaceB)

	require.NoError(t, createRoute(t, instance, userB, workspaceB, &model.NotificationRoute{
		ProviderID: providerB, PluginID: probePluginID, TopicID: reminderTopic,
		Destination: model.Destination{chatField: "-900"}, Enabled: true,
	}))

	err := createRoute(t, instance, userA, workspaceA, &model.NotificationRoute{
		ProviderID: providerB, PluginID: probePluginID, TopicID: reminderTopic,
		Destination: model.Destination{chatField: "-100"}, Enabled: true,
	})
	require.ErrorContains(t, err, "notification_routes_provider_fkey")

	require.NoError(t, createRoute(t, instance, userA, workspaceA, &model.NotificationRoute{
		ProviderID: providerA, PluginID: probePluginID, TopicID: reminderTopic,
		Destination: model.Destination{chatField: "-100"}, Enabled: true,
	}))

	var providers, routes []string

	require.NoError(
		t,
		inScope(t, instance, userA, workspaceA, func(ctx context.Context, tx *sqlx.Tx) error {
			if err := tx.SelectContext(
				ctx,
				&providers,
				`SELECT id FROM public.notification_providers;`,
			); err != nil {
				return err //nolint:wrapcheck // the test reports the error as it is
			}

			return tx.SelectContext(
				ctx,
				&routes,
				`SELECT provider_id FROM public.notification_routes;`,
			)
		}),
	)

	require.Equal(t, []string{providerA}, providers)
	require.Equal(t, []string{providerA}, routes)
}
