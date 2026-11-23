package widget

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pulse/model"
	"github.com/abgeo/maroid/plugins/pulse/permission"
	"github.com/abgeo/maroid/plugins/pulse/repository"
	"github.com/abgeo/maroid/plugins/pulse/schedule"
)

const (
	notSetUpText = "Turn pulse on to receive check-ins."
	pausedText   = "Check-ins paused until %s."
	noneText     = "No check-in today."
	upcomingText = "Upcoming"
	progressText = "%d of %d answered"
	clockLayout  = "15:04"

	historyLink     = "/history"
	preferencesLink = "/preferences"
)

// stateTexts holds the words of the history page for each state of a round.
//
//nolint:gochecknoglobals // a lookup table that nothing writes.
var stateTexts = map[model.RoundState]string{
	model.RoundStateOffered:  "Waiting to start",
	model.RoundStateDone:     "Done",
	model.RoundStateDeclined: "Declined",
	model.RoundStateExpired:  "Expired",
}

// NewToday builds the widget of today: each round of the local day of the acting
// user, and each occurrence that the schedule will still fire that day.
func NewToday(db pluginapi.PluginDB, now func() time.Time) *pluginapi.FuncWidget {
	return pluginapi.NewWidget(
		pluginapi.WidgetMeta{
			Key:        "today",
			Title:      "Today",
			Kind:       pluginapi.WidgetKindList,
			Size:       pluginapi.WidgetSizeM,
			Permission: permission.CheckinsUse,
		},
		func(ctx context.Context) (pluginapi.WidgetAnswer, error) {
			var answer pluginapi.WidgetAnswer

			err := db.WithTx(ctx, func(tx *sqlx.Tx) error {
				var loadErr error

				answer, loadErr = load(ctx, tx, pluginapi.ActingTimeZone(ctx), now())

				return loadErr
			})
			if err != nil {
				return pluginapi.WidgetAnswer{}, fmt.Errorf(
					"reading the check-ins of today: %w",
					err,
				)
			}

			return answer, nil
		},
	)
}

type entry struct {
	moment time.Time
	name   string
	item   pluginapi.ListItem
}

func load(
	ctx context.Context,
	tx *sqlx.Tx,
	zone *time.Location,
	now time.Time,
) (pluginapi.WidgetAnswer, error) {
	current, err := repository.NewSettings(tx).Get(ctx)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !current.Enabled) {
		return pluginapi.NotSetUpAnswer(notSetUpText, preferencesLink), nil
	}

	if err != nil {
		return pluginapi.WidgetAnswer{}, fmt.Errorf("reading the settings: %w", err)
	}

	fired, err := roundEntries(ctx, tx, zone, now)
	if err != nil {
		return pluginapi.WidgetAnswer{}, err
	}

	upcoming, err := upcomingEntries(ctx, tx, current, zone, now)
	if err != nil {
		return pluginapi.WidgetAnswer{}, err
	}

	entries := slices.Concat(fired, upcoming)
	if len(entries) == 0 {
		return emptyAnswer(current, zone, now), nil
	}

	slices.SortStableFunc(entries, func(a, b entry) int {
		return cmp.Or(a.moment.Compare(b.moment), cmp.Compare(a.name, b.name))
	})

	items := make([]pluginapi.ListItem, 0, len(entries))
	for _, each := range entries {
		items = append(items, each.item)
	}

	return pluginapi.ListAnswer(pluginapi.ListData{Items: items, More: historyLink}), nil
}

func roundEntries(
	ctx context.Context,
	tx *sqlx.Tx,
	zone *time.Location,
	now time.Time,
) ([]entry, error) {
	local := now.In(zone)
	from := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, zone)
	to := time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, zone)

	rounds, err := repository.NewRound(tx).ListBetween(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("listing the rounds of today: %w", err)
	}

	answered, err := answeredCounts(ctx, tx, rounds)
	if err != nil {
		return nil, err
	}

	entries := make([]entry, 0, len(rounds))
	for i := range rounds {
		fired := &rounds[i]

		state := stateTexts[fired.State]
		if fired.State == model.RoundStateStarted {
			state = fmt.Sprintf(progressText, answered[fired.ID], fired.QuestionCount)
		}

		link := historyLink
		if fired.CheckinID != nil {
			link = historyLink + "/" + *fired.CheckinID
		}

		entries = append(
			entries,
			newEntry(fired.OccurrenceAt, fired.CheckinName, state, link, zone),
		)
	}

	return entries, nil
}

func answeredCounts(
	ctx context.Context,
	tx *sqlx.Tx,
	rounds []model.Round,
) (map[string]int, error) {
	ids := make([]string, 0, len(rounds))
	for i := range rounds {
		ids = append(ids, rounds[i].ID)
	}

	prompts, err := repository.NewPrompt(tx).ListOf(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("listing the questions of the rounds of today: %w", err)
	}

	counts := make(map[string]int, len(rounds))

	for i := range prompts {
		if prompts[i].State == model.PromptStateAnswered {
			counts[prompts[i].RoundID]++
		}
	}

	return counts, nil
}

func upcomingEntries(
	ctx context.Context,
	tx *sqlx.Tx,
	current *model.Settings,
	zone *time.Location,
	now time.Time,
) ([]entry, error) {
	checkins, err := repository.NewCheckin(tx).ListDue(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing the check-ins: %w", err)
	}

	var entries []entry

	for i := range checkins {
		checkin := &checkins[i]

		floor := checkin.FiresAfter
		if current.PausedUntil != nil && current.PausedUntil.After(floor) {
			floor = *current.PausedUntil
		}

		for _, moment := range schedule.Upcoming(*checkin, zone, floor, now) {
			entries = append(entries,
				newEntry(moment, checkin.Name, upcomingText, historyLink+"/"+checkin.ID, zone))
		}
	}

	return entries, nil
}

func newEntry(moment time.Time, name, state, link string, zone *time.Location) entry {
	return entry{
		moment: moment,
		name:   name,
		item: pluginapi.ListItem{
			Label: name,
			Meta:  moment.In(zone).Format(clockLayout) + " · " + state,
			Link:  link,
		},
	}
}

func emptyAnswer(
	current *model.Settings,
	zone *time.Location,
	now time.Time,
) pluginapi.WidgetAnswer {
	if current.PausedUntil != nil && current.PausedUntil.After(now) {
		return pluginapi.EmptyAnswer(
			fmt.Sprintf(pausedText, current.PausedUntil.In(zone).Format(model.PauseLayout)))
	}

	return pluginapi.EmptyAnswer(noneText)
}
