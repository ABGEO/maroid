package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/render"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/pulse/model"
	"github.com/abgeo/maroid/plugins/pulse/permission"
	"github.com/abgeo/maroid/plugins/pulse/repository"
)

const (
	pathCheckins    = "/checkins"
	pathCheckinByID = "/checkins/{id}"

	nameMax        = 40
	localTimesMax  = 6
	weekdaysMax    = 7
	questionIDsMax = 20
)

// CheckinHandler provides the routes of the check-ins.
type CheckinHandler struct {
	logger *slog.Logger
	db     pluginapi.PluginDB
	now    func() time.Time
}

// NewCheckinHandler creates a CheckinHandler.
func NewCheckinHandler(
	logger *slog.Logger,
	db pluginapi.PluginDB,
	now func() time.Time,
) *CheckinHandler {
	return &CheckinHandler{logger: logger, db: db, now: now}
}

// Routes returns the routes of the check-ins.
func (h *CheckinHandler) Routes() []pluginapi.Route {
	use := permission.CheckinsUse

	return []pluginapi.Route{
		{Method: http.MethodGet, Pattern: pathCheckins, Handler: h.List, Permission: use},
		{Method: http.MethodPost, Pattern: pathCheckins, Handler: h.Create, Permission: use},
		{Method: http.MethodGet, Pattern: pathCheckinByID, Handler: h.Get, Permission: use},
		{Method: http.MethodPut, Pattern: pathCheckinByID, Handler: h.Put, Permission: use},
		{Method: http.MethodDelete, Pattern: pathCheckinByID, Handler: h.Delete, Permission: use},
	}
}

// checkinBody is a check-in on the wire.
type checkinBody struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	LocalTimes  []string  `json:"local_times"`
	Weekdays    []int     `json:"weekdays"`
	Enabled     bool      `json:"enabled"`
	QuestionIDs []string  `json:"question_ids"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// checkinRequest is the body of a write of a check-in.
type checkinRequest struct {
	Name        *string  `json:"name"`
	LocalTimes  []string `json:"local_times"`
	Weekdays    []int    `json:"weekdays"`
	Enabled     *bool    `json:"enabled"`
	QuestionIDs []string `json:"question_ids"`

	clocks model.Clocks
}

func (c *checkinRequest) validate() []problem.FieldFailure {
	var failures fieldFailures

	if c.Name == nil {
		failures.add("name", "is required")
	} else {
		failures.length("name", *c.Name, nameMax)
	}

	if c.Enabled == nil {
		failures.add("enabled", "is required")
	}

	c.validateTimes(&failures)
	c.validateWeekdays(&failures)
	c.validateQuestions(&failures)

	return failures
}

func (c *checkinRequest) validateWeekdays(failures *fieldFailures) {
	switch {
	case c.Weekdays == nil:
		failures.add("weekdays", "is required")
	case len(c.Weekdays) > weekdaysMax:
		failures.add("weekdays", "must hold at most 7 days")
	}

	for index, day := range c.Weekdays {
		switch {
		case day < 1 || day > weekdaysMax:
			failures.add(fmt.Sprintf("weekdays/%d", index), "must be an ISO weekday from 1 to 7")
		case slices.Index(c.Weekdays, day) != index:
			failures.add(fmt.Sprintf("weekdays/%d", index), "must not hold a day two times")
		}
	}
}

func (c *checkinRequest) validateQuestions(failures *fieldFailures) {
	if count := len(c.QuestionIDs); count < 1 || count > questionIDsMax {
		failures.add("question_ids", fmt.Sprintf("must hold 1 to %d questions", questionIDsMax))
	}

	for index, id := range c.QuestionIDs {
		switch {
		case !isUUID(id):
			failures.add(fmt.Sprintf("question_ids/%d", index), "must be a valid UUID")
		case slices.Index(c.QuestionIDs, id) != index:
			failures.add(
				fmt.Sprintf("question_ids/%d", index),
				"must not name a question two times",
			)
		}
	}
}

func (c *checkinRequest) validateTimes(failures *fieldFailures) {
	if count := len(c.LocalTimes); count < 1 || count > localTimesMax {
		failures.add("local_times", fmt.Sprintf("must hold 1 to %d times", localTimesMax))
	}

	for index, value := range c.LocalTimes {
		pointer := fmt.Sprintf("local_times/%d", index)

		clock, err := model.ParseClock(value)

		switch {
		case err != nil:
			failures.add(pointer, "must be a time from 00:00 to 23:59")
		case slices.Index(c.LocalTimes, value) != index:
			failures.add(pointer, "must not hold a time two times")
		default:
			c.clocks = append(c.clocks, clock)
		}
	}
}

func (c *checkinRequest) checkin() model.Checkin {
	return model.Checkin{
		Name:        *c.Name,
		LocalTimes:  c.clocks,
		Weekdays:    c.Weekdays,
		Enabled:     *c.Enabled,
		QuestionIDs: c.QuestionIDs,
	}
}

func newCheckinBody(checkin *model.Checkin) checkinBody {
	times := make([]string, 0, len(checkin.LocalTimes))
	for _, clock := range checkin.LocalTimes {
		times = append(times, clock.String())
	}

	return checkinBody{
		ID:          checkin.ID,
		Name:        checkin.Name,
		LocalTimes:  times,
		Weekdays:    append([]int{}, checkin.Weekdays...),
		Enabled:     checkin.Enabled,
		QuestionIDs: append([]string{}, checkin.QuestionIDs...),
		CreatedAt:   checkin.CreatedAt,
		UpdatedAt:   checkin.UpdatedAt,
	}
}

// List handles GET /checkins, sorted by the first local time, then by id.
func (h *CheckinHandler) List(w http.ResponseWriter, r *http.Request) {
	asked, failure := page.ReadRequest(r, page.Options{})
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	listPage(w, r, h.logger, h.db, asked,
		func(ctx context.Context, tx *sqlx.Tx, seek page.Seek) ([]model.Checkin, error) {
			return repository.NewCheckin(tx).List(ctx, seek)
		},
		func(row model.Checkin) string { return row.ID },
		func(_ context.Context, _ *sqlx.Tx, rows []model.Checkin) ([]checkinBody, error) {
			items := make([]checkinBody, 0, len(rows))
			for index := range rows {
				items = append(items, newCheckinBody(&rows[index]))
			}

			return items, nil
		},
	)
}

// Get handles GET /checkins/{id}.
func (h *CheckinHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	var checkin *model.Checkin

	err := inTx(
		r.Context(),
		h.db,
		"reading the check-in",
		func(ctx context.Context, tx *sqlx.Tx) error {
			var txErr error

			checkin, txErr = repository.NewCheckin(tx).GetByID(ctx, id)
			if txErr != nil {
				return fmt.Errorf("reading the check-in %s: %w", id, txErr)
			}

			return nil
		},
	)
	if err != nil {
		failRead(w, r, h.logger, "failed to read the check-in", err)

		return
	}

	answer(w, r, http.StatusOK, checkin.UpdatedAt, newCheckinBody(checkin))
}

// Create handles POST /checkins. The new check-in fires first at its next occurrence.
func (h *CheckinHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req checkinRequest
	if !readBody(w, r, &req) {
		return
	}

	checkin := req.checkin()

	// A new check-in fires first after the moment of its creation.
	checkin.FiresAfter = h.now()

	err := inTx(
		r.Context(),
		h.db,
		"creating the check-in",
		func(ctx context.Context, tx *sqlx.Tx) error {
			return repository.NewCheckin(tx).Insert(ctx, &checkin)
		},
	)
	if err != nil {
		failWrite(w, r, h.logger, "failed to create the check-in", err)

		return
	}

	w.Header().Set("Location", location(r, checkin.ID))
	answer(w, r, http.StatusCreated, checkin.UpdatedAt, newCheckinBody(&checkin))
}

// Put handles PUT /checkins/{id}. A change of the times, of the weekdays, or a
// turn-on fires first at the next occurrence.
func (h *CheckinHandler) Put(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	var req checkinRequest
	if !readBody(w, r, &req) {
		return
	}

	checkin := req.checkin()
	checkin.ID = id

	err := inTx(
		r.Context(),
		h.db,
		"replacing the check-in",
		func(ctx context.Context, tx *sqlx.Tx) error {
			repo := repository.NewCheckin(tx)

			current, txErr := repo.GetByID(ctx, id)
			if txErr != nil {
				return fmt.Errorf("reading the check-in %s: %w", id, txErr)
			}

			checkin.FiresAfter = current.FiresAfter
			checkin.CreatedAt = current.CreatedAt

			// A check-in whose times change, or that turns on, fires first at
			// its next occurrence.
			if !slices.Equal(current.LocalTimes, checkin.LocalTimes) ||
				!slices.Equal(current.Weekdays, checkin.Weekdays) ||
				(checkin.Enabled && !current.Enabled) {
				checkin.FiresAfter = h.now()
			}

			return repo.Update(ctx, &checkin, precondition.IfMatchFromContext(ctx))
		},
	)
	if err != nil {
		failWrite(w, r, h.logger, "failed to replace the check-in", err)

		return
	}

	answer(w, r, http.StatusOK, checkin.UpdatedAt, newCheckinBody(&checkin))
}

// Delete handles DELETE /checkins/{id}. Its values stay.
func (h *CheckinHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	err := inTx(
		r.Context(),
		h.db,
		"deleting the check-in",
		func(ctx context.Context, tx *sqlx.Tx) error {
			return repository.NewCheckin(tx).
				Delete(ctx, id, precondition.IfMatchFromContext(ctx))
		},
	)
	if err != nil {
		failWrite(w, r, h.logger, "failed to delete the check-in", err)

		return
	}

	render.NoContent(w, r)
}
