package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/pulse/model"
	"github.com/abgeo/maroid/plugins/pulse/permission"
	"github.com/abgeo/maroid/plugins/pulse/repository"
)

const (
	pathValues = "/questions/{id}/values"
	pathDays   = "/questions/{id}/days"

	paramFrom = "from"
	paramTo   = "to"

	daysMax     = 366
	hoursPerDay = 24
)

// ValueHandler provides the routes that read the values of a question.
type ValueHandler struct {
	logger *slog.Logger
	db     pluginapi.PluginDB
}

// NewValueHandler creates a ValueHandler.
func NewValueHandler(logger *slog.Logger, db pluginapi.PluginDB) *ValueHandler {
	return &ValueHandler{logger: logger, db: db}
}

// Routes returns the routes of the values.
func (h *ValueHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		{
			Method:     http.MethodGet,
			Pattern:    pathValues,
			Handler:    h.List,
			Permission: permission.CheckinsUse,
		},
		{
			Method:     http.MethodGet,
			Pattern:    pathDays,
			Handler:    h.Days,
			Permission: permission.CheckinsUse,
		},
	}
}

// List handles GET /questions/{id}/values: the values of the local days from and to,
// newest first.
func (h *ValueHandler) List(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	asked, failure := page.ReadRequest(r, page.Options{Filters: []string{paramFrom, paramTo}})
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	days, ok := readRange(w, r, asked.Filters[paramFrom], asked.Filters[paramTo])
	if !ok {
		return
	}

	if !h.questionExists(w, r, id) {
		return
	}

	end := days.to.AddDate(0, 0, 1)

	listPage(w, r, h.logger, h.db, asked,
		func(ctx context.Context, tx *sqlx.Tx, seek page.Seek) ([]model.Value, error) {
			return repository.NewValue(tx).List(ctx, id, days.from, end, seek)
		},
		func(row model.Value) string { return row.ID },
		func(_ context.Context, _ *sqlx.Tx, rows []model.Value) ([]valueResponse, error) {
			items := make([]valueResponse, 0, len(rows))
			for _, row := range rows {
				items = append(items, newValueResponse(row))
			}

			return items, nil
		},
	)
}

// Days handles GET /questions/{id}/days: one item for each local day that holds a
// value, oldest first, over at most 366 days.
func (h *ValueHandler) Days(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	query := r.URL.Query()

	days, ok := readRange(w, r, query.Get(paramFrom), query.Get(paramTo))
	if !ok {
		return
	}

	if days.count() > daysMax {
		writeField(w, r, paramTo, "must lie at most 366 days after from")

		return
	}

	var items []dayResponse

	err := inTx(
		r.Context(),
		h.db,
		"reading the days",
		func(ctx context.Context, tx *sqlx.Tx) error {
			question, txErr := repository.NewQuestion(tx).GetByID(ctx, id)
			if txErr != nil {
				return fmt.Errorf("reading the question %s: %w", id, txErr)
			}

			grouped, txErr := repository.NewValue(tx).Days(ctx, id, days.from, days.to,
				pluginapi.ActingTimeZone(ctx))
			if txErr != nil {
				return fmt.Errorf("reading the days: %w", txErr)
			}

			items = presentDays(question.Type, grouped)

			return nil
		},
	)
	if err != nil {
		failRead(w, r, h.logger, "failed to read the days", err)

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]any{"items": items})
}

func (h *ValueHandler) questionExists(w http.ResponseWriter, r *http.Request, id string) bool {
	err := inTx(
		r.Context(),
		h.db,
		"reading the question",
		func(ctx context.Context, tx *sqlx.Tx) error {
			if _, txErr := repository.NewQuestion(tx).GetByID(ctx, id); txErr != nil {
				return fmt.Errorf("reading the question %s: %w", id, txErr)
			}

			return nil
		},
	)
	if err != nil {
		failRead(w, r, h.logger, "failed to read the question", err)

		return false
	}

	return true
}

// valueResponse is a value on the wire. The type of the question chooses the member
// that holds it.
type valueResponse struct {
	ID         string    `json:"id"`
	OccurredAt time.Time `json:"occurred_at"`
	AnsweredAt time.Time `json:"answered_at"`
	Source     string    `json:"source"`
	Number     *float64  `json:"number,omitempty"`
	Boolean    *bool     `json:"boolean,omitempty"`
	ChoiceID   *string   `json:"choice_id,omitempty"`
	Text       *string   `json:"text,omitempty"`
	Latitude   *float64  `json:"latitude,omitempty"`
	Longitude  *float64  `json:"longitude,omitempty"`
}

func newValueResponse(value model.Value) valueResponse {
	return valueResponse{
		ID:         value.ID,
		OccurredAt: value.OccurredAt,
		AnsweredAt: value.AnsweredAt,
		Source:     value.Source,
		Number:     value.Number,
		Boolean:    value.Boolean,
		ChoiceID:   value.ChoiceID,
		Text:       value.Text,
		Latitude:   value.Latitude,
		Longitude:  value.Longitude,
	}
}

// dayRange is a range of local days, both included.
type dayRange struct {
	from time.Time
	to   time.Time
}

// count answers the count of local days in the range. The days are read in UTC, so a
// change of daylight saving time adds no hour.
func (d dayRange) count() int {
	day := func(moment time.Time) time.Time {
		return time.Date(moment.Year(), moment.Month(), moment.Day(), 0, 0, 0, 0, time.UTC)
	}

	return int(day(d.to).Sub(day(d.from)).Hours()/hoursPerDay) + 1
}

// readRange reads the local days from and to in the zone of the acting user.
func readRange(w http.ResponseWriter, r *http.Request, from string, to string) (dayRange, bool) {
	zone := pluginapi.ActingTimeZone(r.Context())

	first, err := time.ParseInLocation(time.DateOnly, from, zone)
	if err != nil {
		problem.Write(w, r, problem.NewRequestInvalid().
			WithDetail("The from parameter is not valid: it takes a date, 2026-11-24."))

		return dayRange{}, false
	}

	last, err := time.ParseInLocation(time.DateOnly, to, zone)
	if err != nil || last.Before(first) {
		problem.Write(w, r, problem.NewRequestInvalid().
			WithDetail("The to parameter is not valid: it takes a date that is from or later."))

		return dayRange{}, false
	}

	return dayRange{from: first, to: last}, true
}

// dayResponse is one local day of the values of a question.
type dayResponse struct {
	Date     string   `json:"date"`
	Count    int      `json:"count"`
	Mean     *float64 `json:"mean,omitempty"`
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
	YesCount *int     `json:"yes_count,omitempty"`
}

func presentDays(kind model.QuestionType, grouped []model.Day) []dayResponse {
	items := make([]dayResponse, 0, len(grouped))

	for _, day := range grouped {
		item := dayResponse{Date: day.Date, Count: day.Count}

		switch kind {
		case model.QuestionTypeScale, model.QuestionTypeNumber:
			item.Mean, item.Min, item.Max = day.Mean, day.Min, day.Max
		case model.QuestionTypeBoolean:
			yes := day.YesCount
			item.YesCount = &yes
		case model.QuestionTypeChoice, model.QuestionTypeText, model.QuestionTypeLocation:
		}

		items = append(items, item)
	}

	return items
}
