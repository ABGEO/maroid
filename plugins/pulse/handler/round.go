package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/pulse/model"
	"github.com/abgeo/maroid/plugins/pulse/permission"
	"github.com/abgeo/maroid/plugins/pulse/repository"
)

const (
	pathRounds = "/rounds"

	paramCheckinID = "checkin_id"
	paramState     = "state"
)

// RoundHandler provides the route that lists the rounds of the acting user.
type RoundHandler struct {
	logger *slog.Logger
	db     pluginapi.PluginDB
}

// NewRoundHandler creates a RoundHandler.
func NewRoundHandler(logger *slog.Logger, db pluginapi.PluginDB) *RoundHandler {
	return &RoundHandler{logger: logger, db: db}
}

// Routes returns the routes of the rounds.
func (h *RoundHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		{
			Method:     http.MethodGet,
			Pattern:    pathRounds,
			Handler:    h.List,
			Permission: permission.CheckinsUse,
		},
	}
}

// List handles GET /rounds: the rounds newest first, each with its questions.
func (h *RoundHandler) List(w http.ResponseWriter, r *http.Request) {
	asked, failure := page.ReadRequest(
		r,
		page.Options{Filters: []string{paramCheckinID, paramState}},
	)
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	filter, ok := readRoundFilter(w, r, asked.Filters)
	if !ok {
		return
	}

	listPage(w, r, h.logger, h.db, asked,
		func(ctx context.Context, tx *sqlx.Tx, seek page.Seek) ([]model.Round, error) {
			return repository.NewRound(tx).List(ctx, seek, filter)
		},
		func(row model.Round) string { return row.ID },
		presentRounds,
	)
}

func readRoundFilter(
	w http.ResponseWriter,
	r *http.Request,
	filters map[string]string,
) (repository.RoundFilter, bool) {
	filter := repository.RoundFilter{
		CheckinID: filters[paramCheckinID],
		State:     model.RoundState(filters[paramState]),
	}

	if filter.CheckinID != "" && !isUUID(filter.CheckinID) {
		problem.Write(w, r, problem.NewRequestInvalid().
			WithDetail("The checkin_id parameter is not valid: it takes the id of a check-in."))

		return repository.RoundFilter{}, false
	}

	if filter.State != "" && !filter.State.Known() {
		problem.Write(w, r, problem.NewRequestInvalid().
			WithDetail("The state parameter is not valid: it takes offered, started, done, declined, or expired."),
		)

		return repository.RoundFilter{}, false
	}

	return filter, true
}

// roundResponse is a round on the wire, with its questions in the order of the round.
type roundResponse struct {
	ID           string                  `json:"id"`
	CheckinID    *string                 `json:"checkin_id,omitempty"`
	CheckinName  string                  `json:"checkin_name"`
	OccurrenceAt time.Time               `json:"occurrence_at"`
	State        model.RoundState        `json:"state"`
	Questions    []roundQuestionResponse `json:"questions"`
}

type roundQuestionResponse struct {
	QuestionID string            `json:"question_id"`
	State      model.PromptState `json:"state"`
	Value      *valueResponse    `json:"value,omitempty"`
}

func presentRounds(ctx context.Context, tx *sqlx.Tx, rows []model.Round) ([]roundResponse, error) {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	prompts, err := repository.NewPrompt(tx).ListOf(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("reading the questions of the rounds: %w", err)
	}

	promptIDs := make([]string, 0, len(prompts))
	for _, prompt := range prompts {
		promptIDs = append(promptIDs, prompt.ID)
	}

	values, err := repository.NewValue(tx).ListOfPrompts(ctx, promptIDs)
	if err != nil {
		return nil, fmt.Errorf("reading the values of the rounds: %w", err)
	}

	byPrompt := make(map[string]valueResponse, len(values))
	for _, value := range values {
		byPrompt[*value.PromptID] = newValueResponse(value)
	}

	items := make([]roundResponse, 0, len(rows))

	for _, row := range rows {
		item := roundResponse{
			ID:           row.ID,
			CheckinID:    row.CheckinID,
			CheckinName:  row.CheckinName,
			OccurrenceAt: row.OccurrenceAt,
			State:        row.State,
			Questions:    []roundQuestionResponse{},
		}

		for _, prompt := range prompts {
			if prompt.RoundID != row.ID {
				continue
			}

			question := roundQuestionResponse{QuestionID: prompt.QuestionID, State: prompt.State}
			if value, found := byPrompt[prompt.ID]; found {
				question.Value = &value
			}

			item.Questions = append(item.Questions, question)
		}

		items = append(items, item)
	}

	return items, nil
}
