package pluginmsg_test

import (
	"context"
	"io/fs"
	"log/slog"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/telegram/pluginmsg"
	"github.com/abgeo/maroid/apps/hub/internal/telegram/telegramtest"
	"github.com/abgeo/maroid/libs/pluginapi"
	messageapi "github.com/abgeo/maroid/libs/pluginapi/telegram/message"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	pluginP = "dev.maroid.p"
	pluginQ = "dev.maroid.q"

	telegramOfNina = int64(7001)
	telegramOfBeka = int64(7002)
)

// scene holds Nina and Beka, who each hold a Telegram identity, and Zura, who holds
// a local account alone. The bot reaches the fake API.
type scene struct {
	instance *testdb.Instance
	bot      *telego.Bot
	api      *telegramtest.API
	service  *pluginmsg.Service

	nina string
	beka string
	zura string
}

func newScene(t *testing.T) *scene {
	t.Helper()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	bot, api := telegramtest.NewBot(t)
	scn := &scene{
		instance: instance,
		bot:      bot,
		api:      api,
		service:  pluginmsg.NewService(slog.New(slog.DiscardHandler), instance.DB, bot),
	}

	scn.nina = scn.person(t, "Nina", "telegram", "7001")
	scn.beka = scn.person(t, "Beka", "telegram", "7002")
	scn.zura = scn.person(t, "Zura", "local", "zura@home.example")

	return scn
}

func (s *scene) person(t *testing.T, name, provider, account string) string {
	t.Helper()

	var id string

	require.NoError(t, s.instance.DB.Get(
		&id,
		`INSERT INTO public.users (first_name) VALUES ($1) RETURNING id;`,
		name,
	))

	_, err := s.instance.DB.Exec(
		`INSERT INTO public.identities (user_id, provider, provider_user_id) VALUES ($1, $2, $3);`,
		id, provider, account,
	)
	require.NoError(t, err)

	return id
}

func (s *scene) as(t *testing.T, user string) context.Context {
	t.Helper()

	return pluginapi.ContextWithActingUser(t.Context(), user)
}

// rows answers the plugin messages that the person reads.
func (s *scene) rows(t *testing.T, user string) []model.TelegramMessage {
	t.Helper()

	var rows []model.TelegramMessage

	require.NoError(t, database.WithScopeTx(s.as(t, user), s.instance.DB, func(tx *sqlx.Tx) error {
		return tx.SelectContext(
			t.Context(),
			&rows,
			`SELECT id, plugin_id, chat_id, message_id, claim_until, false AS claim_open,
				created_at, updated_at
			FROM public.telegram_messages ORDER BY created_at;`,
		)
	}))

	return rows
}

// age moves every plugin message of the person back by 31 days.
func (s *scene) age(t *testing.T, user string) {
	t.Helper()

	require.NoError(t, database.WithScopeTx(s.as(t, user), s.instance.DB, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(
			t.Context(),
			`UPDATE public.telegram_messages
			SET created_at = created_at - INTERVAL '31 days', claim_until = NULL;`,
		)

		return err //nolint:wrapcheck // the test reads the error that the database answered.
	}))
}

func (s *scene) send(
	t *testing.T,
	user, pluginID string,
	msg messageapi.Message,
) messageapi.ID {
	t.Helper()

	id, err := s.service.Send(s.as(t, user), pluginID, msg)
	require.NoError(t, err)

	return id
}
