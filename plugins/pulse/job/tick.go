// Package job holds the cron job of pulse.
package job

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pulse/model"
	"github.com/abgeo/maroid/plugins/pulse/repository"
	"github.com/abgeo/maroid/plugins/pulse/round"
	"github.com/abgeo/maroid/plugins/pulse/schedule"
)

const (
	// TickID names the job.
	TickID = "tick"

	// window is how long an occurrence stays able to fire.
	window = 60 * time.Minute
)

// Tick fires the latest due check-in of the acting user.
type Tick struct {
	db     pluginapi.PluginDB
	rounds *round.Service
	now    func() time.Time
}

var _ pluginapi.CronJob = (*Tick)(nil)

// NewTick creates a Tick.
func NewTick(db pluginapi.PluginDB, rounds *round.Service, now func() time.Time) *Tick {
	return &Tick{db: db, rounds: rounds, now: now}
}

// Meta runs the tick every minute for each active user.
func (t *Tick) Meta() pluginapi.CronJobMeta {
	return pluginapi.CronJobMeta{
		ID:       TickID,
		Schedule: "* * * * *",
		Scope:    pluginapi.CronScopePerUser,
	}
}

// Run decides what fires now, and fires it.
func (t *Tick) Run(ctx context.Context) error {
	now := t.now()

	var plan *round.Plan

	err := t.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		current, err := repository.NewSettings(tx).Get(ctx)

		// A person who never turned pulse on costs this one read.
		if errors.Is(err, sql.ErrNoRows) || (err == nil && !current.Enabled) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("reading the settings: %w", err)
		}

		checkins, err := repository.NewCheckin(tx).ListDue(ctx)
		if err != nil {
			return fmt.Errorf("reading the due check-ins: %w", err)
		}

		plan = decide(checkins, current.PausedUntil, pluginapi.ActingTimeZone(ctx), now)

		return nil
	})
	if err != nil {
		return fmt.Errorf("reading the schedule: %w", err)
	}

	if plan == nil {
		return nil
	}

	if err = t.rounds.Fire(ctx, *plan); err != nil {
		return fmt.Errorf("firing the round: %w", err)
	}

	return nil
}

// decide answers the plan of the tick at the moment now, or nil when no occurrence
// is due. The check-ins come ordered by name.
func decide(
	checkins []model.Checkin,
	pausedUntil *time.Time,
	zone *time.Location,
	now time.Time,
) *round.Plan {
	due := make(map[string]time.Time, len(checkins))

	var latest time.Time

	for _, checkin := range checkins {
		// An occurrence more than 60 minutes in the past never fires.
		floor := later(checkin.FiresAfter, now.Add(-window))

		// An occurrence of the pause never fires later.
		if pausedUntil != nil {
			floor = later(floor, *pausedUntil)
		}

		occurrence, ok := schedule.Latest(checkin, zone, floor, now)
		if !ok {
			continue
		}

		due[checkin.ID] = occurrence
		latest = later(latest, occurrence)
	}

	if len(due) == 0 {
		return nil
	}

	// Only the latest due occurrence fires, and an earlier one is marked
	// as passed.
	plan := &round.Plan{Moment: latest, Skip: map[string]time.Time{}}

	for _, checkin := range checkins {
		occurrence, ok := due[checkin.ID]

		switch {
		case !ok:
		case occurrence.Equal(latest):
			plan.Fire = append(plan.Fire, checkin)
		default:
			plan.Skip[checkin.ID] = occurrence
		}
	}

	return plan
}

func later(a time.Time, b time.Time) time.Time {
	if b.After(a) {
		return b
	}

	return a
}
