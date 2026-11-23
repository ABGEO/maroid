package pulsetest

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginapi/telegram/message"
	"github.com/abgeo/maroid/plugins/pulse/job"
	"github.com/abgeo/maroid/plugins/pulse/round"
	"github.com/abgeo/maroid/plugins/pulse/settings"
)

// Pulse holds the services of pulse over one schema, one clock, and one fake
// messenger.
type Pulse struct {
	DB        pluginapi.PluginDB
	Clock     *Clock
	Messenger *Messenger
	Rounds    *round.Service
	Settings  *settings.Service
	Tick      *job.Tick
}

// The clock of New starts on 24 November 2026 at 06:00.
const (
	startDay  = 24
	startHour = 6
)

// New starts a schema and builds the services over it, with the clock at the moment
// of Nina on 24 November 2026 at 06:00.
func New(t *testing.T) *Pulse {
	t.Helper()

	database := Schema(t)
	clock := NewClock(At(startDay, startHour, 0))
	messenger := &Messenger{}
	rounds := round.NewService(slog.New(slog.DiscardHandler), database, messenger, clock.Now)

	return &Pulse{
		DB:        database,
		Clock:     clock,
		Messenger: messenger,
		Rounds:    rounds,
		Settings:  settings.NewService(database, rounds, clock.Now),
		Tick:      job.NewTick(database, rounds, clock.Now),
	}
}

// TurnOn turns pulse on for the user at the moment of the clock.
func (p *Pulse) TurnOn(t *testing.T, user string) {
	t.Helper()

	_, err := p.Settings.Put(ActingAs(user), true, nil, nil)
	require.NoError(t, err)
}

// TickAt moves the clock to the moment and runs the tick for the user.
func (p *Pulse) TickAt(t *testing.T, user string, moment time.Time) {
	t.Helper()

	p.Clock.Set(moment)
	require.NoError(t, p.Tick.Run(ActingAs(user)))
}

// Press presses the button of the message as Nina at the moment, and answers the
// notice.
func (p *Pulse) Press(t *testing.T, moment time.Time, id message.ID, value string) string {
	t.Helper()

	p.Clock.Set(moment)

	notice, err := p.Rounds.Press(ActingAs(Nina), message.Press{Message: id, Value: value})
	require.NoError(t, err)

	return notice
}

// Reply sends a reply on the message as Nina at the moment.
func (p *Pulse) Reply(t *testing.T, moment time.Time, reply message.Reply) {
	t.Helper()

	p.Clock.Set(moment)
	require.NoError(t, p.Rounds.Reply(ActingAs(Nina), reply))
}

// QuestionID answers the id of the question of Nina with the prompt.
func (p *Pulse) QuestionID(t *testing.T, prompt string) string {
	t.Helper()

	return ID(t, p.DB, Nina, "SELECT id::text FROM questions WHERE prompt = $1", prompt)
}
