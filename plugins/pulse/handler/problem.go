package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/pulse/errs"
)

const (
	typeQuestionHasValues = "/problems/dev.maroid.pulse/question-has-values"
	typeChoiceInUse       = "/problems/dev.maroid.pulse/choice-in-use"
)

// failWrite answers a write that did not land. Every cause that no rule names
// answers 500, and the log record is the one report of it.
func failWrite(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	text string,
	err error,
) {
	var (
		choiceUnknown    *errs.ChoiceUnknownError
		choiceInUse      *errs.ChoiceInUseError
		questionUnusable *errs.QuestionUnusableError
	)

	switch {
	case errors.Is(err, precondition.ErrModified):
		problem.Write(w, r, problem.NewPreconditionFailed())
	case errors.Is(err, sql.ErrNoRows):
		problem.Write(w, r, problem.NewNotFound())
	case errors.Is(err, errs.ErrQuestionHasValues):
		problem.Write(w, r, &problem.Problem{
			Type:   typeQuestionHasValues,
			Title:  "The question holds a value.",
			Status: http.StatusConflict,
			Detail: "A question with a value keeps its type, and stays. Archive it instead.",
		})
	case errors.As(err, &choiceInUse):
		problem.Write(w, r, &problem.Problem{
			Type:   typeChoiceInUse,
			Title:  "A value records the choice.",
			Status: http.StatusConflict,
			Detail: fmt.Sprintf("A value records the choice %q (%s), so the question keeps it.",
				choiceInUse.Label, choiceInUse.ID),
		})
	case errors.As(err, &choiceUnknown):
		writeField(w, r, fmt.Sprintf("config/choices/%d/id", choiceUnknown.Index),
			"names no choice of the question")
	case errors.As(err, &questionUnusable):
		writeField(w, r, fmt.Sprintf("question_ids/%d", questionUnusable.Index),
			"names no question that is not archived")
	default:
		logger.ErrorContext(r.Context(), text, slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())
	}
}

// failRead answers a read that did not land: 404 for no record, else 500.
func failRead(w http.ResponseWriter, r *http.Request, logger *slog.Logger, text string, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		problem.Write(w, r, problem.NewNotFound())

		return
	}

	logger.ErrorContext(r.Context(), text, slog.Any("error", err))
	problem.Write(w, r, problem.NewInternal())
}

func writeField(w http.ResponseWriter, r *http.Request, pointer string, detail string) {
	var failures fieldFailures

	failures.add(pointer, detail)
	problem.Write(w, r, problem.NewValidationFailed(failures...))
}
