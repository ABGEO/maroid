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
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/pulse/errs"
	"github.com/abgeo/maroid/plugins/pulse/model"
	"github.com/abgeo/maroid/plugins/pulse/permission"
	"github.com/abgeo/maroid/plugins/pulse/repository"
)

const (
	pathQuestions    = "/questions"
	pathQuestionByID = "/questions/{id}"
	filterArchived   = "archived"
)

// QuestionHandler provides the routes of the questions.
type QuestionHandler struct {
	logger *slog.Logger
	db     pluginapi.PluginDB
	now    func() time.Time
}

// NewQuestionHandler creates a QuestionHandler.
func NewQuestionHandler(
	logger *slog.Logger,
	db pluginapi.PluginDB,
	now func() time.Time,
) *QuestionHandler {
	return &QuestionHandler{logger: logger, db: db, now: now}
}

// Routes returns the routes of the questions.
func (h *QuestionHandler) Routes() []pluginapi.Route {
	use := permission.CheckinsUse

	return []pluginapi.Route{
		{Method: http.MethodGet, Pattern: pathQuestions, Handler: h.List, Permission: use},
		{Method: http.MethodPost, Pattern: pathQuestions, Handler: h.Create, Permission: use},
		{Method: http.MethodGet, Pattern: pathQuestionByID, Handler: h.Get, Permission: use},
		{Method: http.MethodPut, Pattern: pathQuestionByID, Handler: h.Put, Permission: use},
		{Method: http.MethodDelete, Pattern: pathQuestionByID, Handler: h.Delete, Permission: use},
	}
}

// List handles GET /questions. With no archived=true, it lists the questions that
// are not archived.
func (h *QuestionHandler) List(w http.ResponseWriter, r *http.Request) {
	asked, failure := page.ReadRequest(r, page.Options{Filters: []string{filterArchived}})
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	archived := false

	switch asked.Filters[filterArchived] {
	case "", "false":
	case "true":
		archived = true
	default:
		problem.Write(w, r, problem.NewRequestInvalid().
			WithDetail("The archived parameter is not valid: it takes true or false."))

		return
	}

	listPage(w, r, h.logger, h.db, asked,
		func(ctx context.Context, tx *sqlx.Tx, seek page.Seek) ([]model.QuestionSummary, error) {
			return repository.NewQuestion(tx).List(ctx, seek, archived)
		},
		func(row model.QuestionSummary) string { return row.ID },
		presentQuestions,
	)
}

// presentQuestions adds the choices and the latest value to each question.
func presentQuestions(
	ctx context.Context,
	tx *sqlx.Tx,
	rows []model.QuestionSummary,
) ([]questionResponse, error) {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	choices, err := repository.NewChoice(tx).ListFor(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("reading the choices: %w", err)
	}

	latest, err := repository.NewValue(tx).Latest(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("reading the latest values: %w", err)
	}

	items := make([]questionResponse, 0, len(rows))

	for index := range rows {
		row := &rows[index]

		for _, choice := range choices {
			if choice.QuestionID == row.ID {
				row.Choices = append(row.Choices, choice)
			}
		}

		for _, value := range latest {
			if value.QuestionID == row.ID {
				row.Latest = &value
			}
		}

		items = append(items, newQuestionResponse(row))
	}

	return items, nil
}

// Get handles GET /questions/{id}.
func (h *QuestionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	var item questionResponse

	err := inTx(
		r.Context(),
		h.db,
		"reading the question",
		func(ctx context.Context, tx *sqlx.Tx) error {
			var txErr error

			item, txErr = readQuestionResponse(ctx, tx, id)

			return txErr
		},
	)
	if err != nil {
		failRead(w, r, h.logger, "failed to read the question", err)

		return
	}

	answer(w, r, http.StatusOK, item.UpdatedAt, item)
}

func readQuestionResponse(ctx context.Context, tx *sqlx.Tx, id string) (questionResponse, error) {
	summary, err := repository.NewQuestion(tx).GetByID(ctx, id)
	if err != nil {
		return questionResponse{}, fmt.Errorf("reading the question %s: %w", id, err)
	}

	items, err := presentQuestions(ctx, tx, []model.QuestionSummary{*summary})
	if err != nil {
		return questionResponse{}, err
	}

	return items[0], nil
}

// Create handles POST /questions.
func (h *QuestionHandler) Create(w http.ResponseWriter, r *http.Request) {
	input, ok := readQuestion(w, r)
	if !ok {
		return
	}

	input.question.ArchivedAt = archivedAt(nil, input.archived, h.now())

	for index, choice := range input.choices {
		if choice.ID != "" {
			failWrite(w, r, h.logger, "", &errs.ChoiceUnknownError{Index: index})

			return
		}
	}

	var item questionResponse

	err := inTx(
		r.Context(),
		h.db,
		"creating the question",
		func(ctx context.Context, tx *sqlx.Tx) error {
			question := input.question
			if err := repository.NewQuestion(tx).Insert(ctx, &question); err != nil {
				return fmt.Errorf("storing the question: %w", err)
			}

			if err := writeChoices(ctx, tx, &question, input.choices); err != nil {
				return err
			}

			var txErr error

			item, txErr = readQuestionResponse(ctx, tx, question.ID)

			return txErr
		},
	)
	if err != nil {
		failWrite(w, r, h.logger, "failed to create the question", err)

		return
	}

	w.Header().Set("Location", location(r, item.ID))
	answer(w, r, http.StatusCreated, item.UpdatedAt, item)
}

// Put handles PUT /questions/{id}. A question with a value keeps its type, and keeps
// each choice that a value records.
func (h *QuestionHandler) Put(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	input, ok := readQuestion(w, r)
	if !ok {
		return
	}

	ifMatch := precondition.IfMatchFromContext(r.Context())

	var item questionResponse

	err := inTx(
		r.Context(),
		h.db,
		"replacing the question",
		func(ctx context.Context, tx *sqlx.Tx) error {
			repo := repository.NewQuestion(tx)

			current, txErr := repo.GetByID(ctx, id)
			if txErr != nil {
				return fmt.Errorf("reading the question %s: %w", id, txErr)
			}

			question := input.question
			question.ID = id
			question.ArchivedAt = archivedAt(current.ArchivedAt, input.archived, h.now())

			// The update itself refuses a new type while a value exists.
			if txErr = repo.Update(
				ctx,
				&question,
				current.Type != question.Type,
				ifMatch,
			); txErr != nil {
				return fmt.Errorf("storing the question %s: %w", id, txErr)
			}

			if txErr = writeChoices(ctx, tx, &question, input.choices); txErr != nil {
				return txErr
			}

			item, txErr = readQuestionResponse(ctx, tx, id)

			return txErr
		},
	)
	if err != nil {
		failWrite(w, r, h.logger, "failed to replace the question", err)

		return
	}

	answer(w, r, http.StatusOK, item.UpdatedAt, item)
}

// writeChoices makes the choices of the question follow its type. A type other than
// choice holds no row of choices.
func writeChoices(
	ctx context.Context,
	tx *sqlx.Tx,
	question *model.Question,
	choices []model.Choice,
) error {
	repo := repository.NewChoice(tx)

	if question.Type != model.QuestionTypeChoice {
		if err := repo.DeleteAll(ctx, question.ID); err != nil {
			return fmt.Errorf("deleting the choices: %w", err)
		}

		return nil
	}

	if err := repo.Replace(ctx, question.ID, choices); err != nil {
		return fmt.Errorf("storing the choices: %w", err)
	}

	return nil
}

// Delete handles DELETE /questions/{id}. A question with a value stays.
func (h *QuestionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	err := inTx(
		r.Context(),
		h.db,
		"deleting the question",
		func(ctx context.Context, tx *sqlx.Tx) error {
			return repository.NewQuestion(tx).
				Delete(ctx, id, precondition.IfMatchFromContext(ctx))
		},
	)
	if err != nil {
		failWrite(w, r, h.logger, "failed to delete the question", err)

		return
	}

	render.NoContent(w, r)
}
