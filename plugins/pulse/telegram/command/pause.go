package command

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pulse/errs"
	"github.com/abgeo/maroid/plugins/pulse/model"
	"github.com/abgeo/maroid/plugins/pulse/permission"
)

const (
	pauseHelp  = "Send /pause with a date, 2026-12-01, or a duration, 3d or 12h."
	pausedText = "Check-ins paused until %s."

	day        = 24 * time.Hour
	daysMax    = 366
	hoursMax   = daysMax * 24
	pauseLimit = daysMax * day
)

var durationPattern = regexp.MustCompile(`^([0-9]{1,4})([dh])$`)

// Pause is the command /pause, which stops every check-in until a date or for a
// duration.
type Pause struct {
	pauser Pauser
	bot    pluginapi.TelegramBot
	now    func() time.Time
}

var _ pluginapi.TelegramCommand = (*Pause)(nil)

// NewPause creates a Pause.
func NewPause(pauser Pauser, bot pluginapi.TelegramBot, now func() time.Time) *Pause {
	return &Pause{pauser: pauser, bot: bot, now: now}
}

// Meta names the command.
func (p *Pause) Meta() pluginapi.TelegramCommandMeta {
	return pluginapi.TelegramCommandMeta{
		Command:     "pause",
		Description: "Pause the check-ins until a date or for a duration",
		Scope:       tu.ScopeAllPrivateChats(),
		Permission:  permission.CheckinsUse,
	}
}

// Validate accepts every update. Reply answers a wrong argument.
func (p *Pause) Validate(_ telego.Update) error {
	return nil
}

// Handle answers the command in its chat.
func (p *Pause) Handle(ctx *th.Context, update telego.Update) error {
	text, err := p.Reply(ctx, update.Message.Text)
	if err != nil {
		return err
	}

	return answer(ctx, p.bot, update.Message, text)
}

// Reply sets the pause that the text of the command names, and answers the text for
// the person.
func (p *Pause) Reply(ctx context.Context, text string) (string, error) {
	zone := pluginapi.ActingTimeZone(ctx)

	until, err := ParseUntil(argument(text), p.now(), zone)
	if err != nil {
		return pauseHelp, nil //nolint:nilerr // a wrong argument is an answer, not a failure.
	}

	if err = p.pauser.Pause(ctx, until); err != nil {
		return offOr(err)
	}

	return fmt.Sprintf(pausedText, until.In(zone).Format(model.PauseLayout)), nil
}

// ParseUntil reads the end of a pause: YYYY-MM-DD, the start of that day in the zone,
// or a duration of 1 to 366 days, <n>d, or of 1 to 8784 hours, <n>h. The end lies
// in the future and at most 366 days ahead.
func ParseUntil(argument string, now time.Time, zone *time.Location) (time.Time, error) {
	until, err := readUntil(argument, now, zone)
	if err != nil {
		return time.Time{}, err
	}

	if !until.After(now) || until.After(now.Add(pauseLimit)) {
		return time.Time{}, fmt.Errorf(
			"%w: %s lies outside the next 366 days",
			errs.ErrPauseArgument,
			argument,
		)
	}

	return until, nil
}

func readUntil(argument string, now time.Time, zone *time.Location) (time.Time, error) {
	if date, err := time.ParseInLocation(time.DateOnly, argument, zone); err == nil {
		return date, nil
	}

	parts := durationPattern.FindStringSubmatch(argument)
	if parts == nil {
		return time.Time{}, fmt.Errorf("%w: %q", errs.ErrPauseArgument, argument)
	}

	count, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %q", errs.ErrPauseArgument, argument)
	}

	unit, most := day, daysMax
	if parts[2] == "h" {
		unit, most = time.Hour, hoursMax
	}

	if count < 1 || count > most {
		return time.Time{}, fmt.Errorf("%w: %q", errs.ErrPauseArgument, argument)
	}

	return now.Add(time.Duration(count) * unit), nil
}
