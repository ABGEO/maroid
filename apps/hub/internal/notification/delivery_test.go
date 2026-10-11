package notification_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/notification"
	"github.com/abgeo/maroid/apps/hub/internal/notification/telegram"
	"github.com/abgeo/maroid/apps/hub/internal/notification/telegramtest"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	telegramToken = "123456:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	cipherPrefix  = "enc:"
	tokenField    = "bot_token"
	chatField     = "chat_id"
)

var errUnprotected = errors.New("the value is not protected by the key")

// recordingCipher protects a value with a readable form that names its key, and
// records each key that a read used.
type recordingCipher struct {
	mu   sync.Mutex
	keys []secret.Key
}

func (c *recordingCipher) Encrypt(
	_ context.Context,
	key secret.Key,
	plaintext string,
) (string, error) {
	return cipherPrefix + string(key) + ":" + plaintext, nil
}

func (c *recordingCipher) Decrypt(
	_ context.Context,
	key secret.Key,
	ciphertext string,
) (string, error) {
	c.record(key)

	plaintext, found := strings.CutPrefix(ciphertext, cipherPrefix+string(key)+":")
	if !found {
		return "", fmt.Errorf("%w: %s", errUnprotected, key)
	}

	return plaintext, nil
}

func (c *recordingCipher) record(key secret.Key) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.keys = append(c.keys, key)
}

// deliveryWorld holds a database with workspace A and workspace B, a Telegram stub,
// and a dispatcher that reaches both.
type deliveryWorld struct {
	instance   *testdb.Instance
	stub       *telegramtest.Server
	cipher     secret.Cipher
	recorder   *recordingCipher
	dispatcher *notification.Dispatcher
	lines      *bytes.Buffer
	workspaceA string
	workspaceB string
}

func newDeliveryWorld(t *testing.T) *deliveryWorld {
	t.Helper()

	recorder := &recordingCipher{}
	world := newDeliveryWorldWith(t, recorder)
	world.recorder = recorder

	return world
}

// newDeliveryWorldWith builds the world around the given cipher. The cipher must hold
// a key for each workspace that the world adds.
func newDeliveryWorldWith(t *testing.T, cipher secret.Cipher) *deliveryWorld {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)
	instance.Migrate(t, "public", migrations)

	topics := registry.NewNotificationTopicRegistry()
	require.NoError(t, topics.Register(
		pluginapi.ParsePluginID(probeID),
		pluginapi.NotificationTopic{ID: reminderTopic, Name: "Reminder"},
		pluginapi.NotificationTopic{ID: reportTopic, Name: "Report"},
	))

	stub := telegramtest.NewServer(t)
	kinds := notification.NewKindRegistry()
	require.NoError(t, kinds.Register(telegram.New(telego.WithAPIServer(stub.URL()))))

	lines := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(lines, &slog.HandlerOptions{Level: slog.LevelDebug}))

	return &deliveryWorld{
		instance:   instance,
		stub:       stub,
		cipher:     cipher,
		dispatcher: notification.NewDispatcher(logger, topics, instance.DB, kinds, cipher),
		lines:      lines,
		workspaceA: addWorkspace(t, instance, "A"),
		workspaceB: addWorkspace(t, instance, "B"),
	}
}

func addWorkspace(t *testing.T, instance *testdb.Instance, name string) string {
	t.Helper()

	var id string

	require.NoError(t, instance.DB.Get(
		&id, `INSERT INTO public.workspaces (name) VALUES ($1) RETURNING id;`, name,
	))

	return id
}

func runIn(workspace string) context.Context {
	return pluginapi.ContextWithActingWorkspace(context.Background(), workspace)
}

// addProvider stores a Telegram provider in the workspace, with its token protected.
func (world *deliveryWorld) addProvider(t *testing.T, workspace string, name string) string {
	t.Helper()

	ctx := runIn(workspace)
	protected, err := world.cipher.Encrypt(ctx, secret.WorkspaceKey(workspace), telegramToken)
	require.NoError(t, err)

	var id string

	require.NoError(t, database.WithScopeTx(ctx, world.instance.DB, func(tx *sqlx.Tx) error {
		stored, createErr := repository.NewNotificationProvider(tx).
			Create(ctx, &model.NotificationProvider{
				Kind:    "telegram",
				Name:    name,
				Fields:  model.Fields{tokenField: {Kind: model.FieldKindSecret, Value: protected}},
				Enabled: true,
			})
		if createErr != nil {
			return createErr //nolint:wrapcheck // the test reports the error as it is
		}

		id = stored.ID

		return nil
	}))

	return id
}

// addRoute stores an enabled route of the topic to the chat.
func (world *deliveryWorld) addRoute(t *testing.T, workspace, provider, topic, chat string) {
	t.Helper()

	ctx := runIn(workspace)

	require.NoError(t, database.WithScopeTx(ctx, world.instance.DB, func(tx *sqlx.Tx) error {
		_, err := repository.NewNotificationRoute(tx).Create(ctx, &model.NotificationRoute{
			ProviderID:  provider,
			PluginID:    probeID,
			TopicID:     topic,
			Destination: model.Destination{chatField: chat},
			Enabled:     true,
		})

		return err //nolint:wrapcheck // the test reports the error as it is
	}))
}

func (world *deliveryWorld) send(workspace string, topic string) error {
	//nolint:wrapcheck // the test reads the error that the dispatcher answered.
	return world.dispatcher.Send(
		runIn(workspace), pluginapi.ParsePluginID(probeID), topic,
		pluginapi.Message{Title: "Plant care due", Body: "E001 Office"},
	)
}

func (world *deliveryWorld) chats() []string {
	calls := world.stub.Calls()
	chats := make([]string, 0, len(calls))

	for _, call := range calls {
		chats = append(chats, call.Params[chatField])
	}

	return chats
}

// NOTIF-SC-019: A send reaches each enabled route of the topic in the acting
// workspace, in the order of the route identifiers, and no route of another.
func TestSendDeliversThroughEachRouteOfTheWorkspace(t *testing.T) {
	t.Parallel()

	world := newDeliveryWorld(t)
	family := world.addProvider(t, world.workspaceA, "Family bot")
	work := world.addProvider(t, world.workspaceA, "Work bot")
	other := world.addProvider(t, world.workspaceB, "Their bot")

	world.addRoute(t, world.workspaceA, family, reminderTopic, "-100")
	world.addRoute(t, world.workspaceA, work, reminderTopic, "-200")
	world.addRoute(t, world.workspaceB, other, reminderTopic, "-900")

	require.NoError(t, world.send(world.workspaceA, reminderTopic))

	require.Equal(t, []string{"-100", "-200"}, world.chats())
	require.Contains(t, world.recorder.keys, secret.WorkspaceKey(world.workspaceA))
	require.NotContains(t, world.recorder.keys, secret.WorkspaceKey(world.workspaceB))
}

// NOTIF-SC-020: A failed route does not stop the next one. The plugin gets an error
// only when every route fails, and each failure leaves one error log.
func TestSendFailsOnlyWhenEveryRouteFails(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		refused  []string
		failure  bool
		failures int
	}{
		"one fails": {refused: []string{"-100"}, failure: false, failures: 1},
		"both fail": {refused: []string{"-100", "-200"}, failure: true, failures: 2},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			world := newDeliveryWorld(t)
			provider := world.addProvider(t, world.workspaceA, "Family bot")
			world.addRoute(t, world.workspaceA, provider, reminderTopic, "-100")
			world.addRoute(t, world.workspaceA, provider, reminderTopic, "-200")

			for _, chat := range tc.refused {
				world.stub.Refuse(chat, "Bad Request: chat not found")
			}

			err := world.send(world.workspaceA, reminderTopic)

			if tc.failure {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, []string{"-100", "-200"}, world.chats())

			var errors []map[string]any

			for _, record := range logLines(t, world.lines) {
				if record["level"] == "ERROR" {
					errors = append(errors, record)
				}
			}

			require.Len(t, errors, tc.failures)

			for _, record := range errors {
				require.Equal(t, world.workspaceA, record["workspace_id"])
				require.Equal(t, probeID, record["plugin"])
				require.Equal(t, reminderTopic, record["topic"])
				require.NotEmpty(t, record["route_id"])
				require.Contains(t, record["error"], "Bad Request: chat not found")
			}
		})
	}
}

// NOTIF-SC-021: An unrouted topic delivers nothing, logs it, and reports no failure.
func TestSendOfAnUnroutedTopicDropsTheNotification(t *testing.T) {
	t.Parallel()

	world := newDeliveryWorld(t)
	provider := world.addProvider(t, world.workspaceA, "Family bot")
	world.addRoute(t, world.workspaceA, provider, reminderTopic, "-100")

	require.NoError(t, world.send(world.workspaceA, reportTopic))
	require.Empty(t, world.stub.Calls())

	records := logLines(t, world.lines)
	require.Len(t, records, 1)
	require.Equal(t, "INFO", records[0]["level"])
	require.Equal(t, "notification unrouted", records[0]["msg"])
	require.Equal(t, world.workspaceA, records[0]["workspace_id"])
	require.Equal(t, probeID, records[0]["plugin"])
	require.Equal(t, reportTopic, records[0]["topic"])
}
