package repository

import (
	"context"
	"fmt"
	"slices"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/pulse/errs"
	"github.com/abgeo/maroid/plugins/pulse/model"
)

const choiceValuesConstraint = "question_values_choice_fkey"

// ChoiceRepository defines the data access contract for Choice entities.
type ChoiceRepository interface {
	ListFor(ctx context.Context, questionIDs []string) ([]model.Choice, error)
	Replace(ctx context.Context, questionID string, choices []model.Choice) error
	DeleteAll(ctx context.Context, questionID string) error
}

// Choice reads and writes the choices of the questions of the acting user.
type Choice struct {
	tx *sqlx.Tx
}

var _ ChoiceRepository = (*Choice)(nil)

// NewChoice creates a Choice repository over the transaction.
func NewChoice(tx *sqlx.Tx) *Choice {
	return &Choice{tx: tx}
}

// ListFor answers the choices of the questions, in the order of each question.
func (r *Choice) ListFor(ctx context.Context, questionIDs []string) ([]model.Choice, error) {
	if len(questionIDs) == 0 {
		return nil, nil
	}

	var entities []model.Choice

	err := selectNamed(ctx, r.tx, &entities, `
		SELECT id, question_id, position, label, created_at, updated_at
		FROM choices
		WHERE question_id IN (:ids)
		ORDER BY question_id, position;`,
		map[string]any{paramIDs: questionIDs},
	)
	if err != nil {
		return nil, fmt.Errorf("listing Choices: %w", err)
	}

	return entities, nil
}

// Replace makes the choices of the question equal the list, in its order. A choice
// with an id keeps that row, a choice with none is new, and a row left out goes. It
// refreshes the id of each new choice.
func (r *Choice) Replace(ctx context.Context, questionID string, choices []model.Choice) error {
	current, err := r.ListFor(ctx, []string{questionID})
	if err != nil {
		return err
	}

	kept, err := keptChoices(current, choices)
	if err != nil {
		return err
	}

	if err = r.removeUnkept(ctx, current, kept); err != nil {
		return err
	}

	for index := range choices {
		if err = r.write(ctx, questionID, index, &choices[index]); err != nil {
			return err
		}
	}

	return nil
}

// DeleteAll deletes every choice of the question, for a type that holds none.
func (r *Choice) DeleteAll(ctx context.Context, questionID string) error {
	_, err := execNamed(ctx, r.tx, "DELETE FROM choices WHERE question_id = :id;",
		map[string]any{paramID: questionID})
	if err != nil {
		return fmt.Errorf("deleting Choices: %w", err)
	}

	return nil
}

func (r *Choice) removeUnkept(ctx context.Context, current []model.Choice, kept []string) error {
	for _, choice := range current {
		if slices.Contains(kept, choice.ID) {
			continue
		}

		_, err := execNamed(ctx, r.tx, "DELETE FROM choices WHERE id = :id;",
			map[string]any{paramID: choice.ID})
		if isForeignKeyViolation(err, choiceValuesConstraint) {
			return &errs.ChoiceInUseError{ID: choice.ID, Label: choice.Label}
		}

		if err != nil {
			return fmt.Errorf("deleting Choice: %w", err)
		}
	}

	return nil
}

func (r *Choice) write(
	ctx context.Context,
	questionID string,
	index int,
	choice *model.Choice,
) error {
	choice.QuestionID = questionID
	choice.Position = index + 1

	args := map[string]any{
		paramID:         choice.ID,
		paramQuestionID: questionID,
		"position":      choice.Position,
		"label":         choice.Label,
	}

	query := `
		UPDATE choices SET position = :position, label = :label
		WHERE id = :id
		RETURNING id, created_at, updated_at;`
	if choice.ID == "" {
		query = `
			INSERT INTO choices (question_id, position, label)
			VALUES (:question_id, :position, :label)
			RETURNING id, created_at, updated_at;`
	}

	if err := getNamed(ctx, r.tx, choice, query, args); err != nil {
		return fmt.Errorf("writing Choice: %w", err)
	}

	return nil
}

// keptChoices answers the ids of the list, and refuses an id that the question does
// not hold.
func keptChoices(current []model.Choice, choices []model.Choice) ([]string, error) {
	held := make([]string, 0, len(current))
	for _, choice := range current {
		held = append(held, choice.ID)
	}

	kept := make([]string, 0, len(choices))

	for index, choice := range choices {
		if choice.ID == "" {
			continue
		}

		if !slices.Contains(held, choice.ID) {
			return nil, &errs.ChoiceUnknownError{Index: index}
		}

		kept = append(kept, choice.ID)
	}

	return kept, nil
}
