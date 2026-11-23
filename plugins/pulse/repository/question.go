package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/plugins/pulse/errs"
	"github.com/abgeo/maroid/plugins/pulse/model"
)

const (
	questionValuesConstraint = "question_values_question_fkey"

	questionColumns = `q.id, q.type, q.prompt, q.config, q.archived_at, q.created_at, q.updated_at`

	selectQuestionSummaries = `
		SELECT ` + questionColumns + `,
		       (SELECT count(*) FROM question_values v WHERE v.question_id = q.id) AS value_count
		FROM questions q`
)

// QuestionRepository defines the data access contract for Question entities.
type QuestionRepository interface {
	Insert(ctx context.Context, entity *model.Question) error
	GetByID(ctx context.Context, id string) (*model.QuestionSummary, error)
	GetMany(ctx context.Context, ids []string) ([]model.Question, error)
	List(ctx context.Context, seek page.Seek, archived bool) ([]model.QuestionSummary, error)
	Update(ctx context.Context, entity *model.Question, typeChanges bool, ifMatch *time.Time) error
	Delete(ctx context.Context, id string, ifMatch *time.Time) error
}

// Question reads and writes the questions of the acting user.
type Question struct {
	tx *sqlx.Tx
}

var _ QuestionRepository = (*Question)(nil)

// NewQuestion creates a Question repository over the transaction.
func NewQuestion(tx *sqlx.Tx) *Question {
	return &Question{tx: tx}
}

// Insert stores a new question and refreshes the entity with the stored values.
func (r *Question) Insert(ctx context.Context, entity *model.Question) error {
	err := getNamed(ctx, r.tx, entity, `
		INSERT INTO questions (type, prompt, config, archived_at)
		VALUES (:type, :prompt, CAST(:config AS jsonb), :archived_at)
		RETURNING id, created_at, updated_at;`,
		questionArgs(entity),
	)
	if err != nil {
		return fmt.Errorf("inserting Question: %w", err)
	}

	return nil
}

// GetByID answers one question with its count of values, or sql.ErrNoRows.
func (r *Question) GetByID(ctx context.Context, id string) (*model.QuestionSummary, error) {
	var entity model.QuestionSummary

	err := getNamed(ctx, r.tx, &entity, selectQuestionSummaries+" WHERE q.id = :id;",
		map[string]any{paramID: id})
	if err != nil {
		return nil, fmt.Errorf("getting Question: %w", err)
	}

	return &entity, nil
}

// GetMany answers the questions that the identifiers name, in no order.
func (r *Question) GetMany(ctx context.Context, ids []string) ([]model.Question, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var entities []model.Question

	err := selectNamed(ctx, r.tx, &entities,
		"SELECT "+questionColumns+" FROM questions q WHERE q.id IN (:ids);",
		map[string]any{paramIDs: ids})
	if err != nil {
		return nil, fmt.Errorf("getting Questions: %w", err)
	}

	return entities, nil
}

// List answers one page of the questions, sorted by id, either the archived ones or
// the others.
func (r *Question) List(
	ctx context.Context,
	seek page.Seek,
	archived bool,
) ([]model.QuestionSummary, error) {
	condition, order := keyset(seek, "q.id")
	query := selectQuestionSummaries + `
		WHERE ` + condition + ` AND (q.archived_at IS NOT NULL) = :archived
		ORDER BY ` + order + ` LIMIT :limit;`

	var entities []model.QuestionSummary

	err := selectNamed(ctx, r.tx, &entities, query, map[string]any{
		paramBoundary: nullable(seek.Boundary),
		paramLimit:    seek.Limit,
		"archived":    archived,
	})
	if err != nil {
		return nil, fmt.Errorf("listing Questions: %w", err)
	}

	if seek.Direction == page.DirectionBackward {
		slices.Reverse(entities)
	}

	return entities, nil
}

// Update writes the question under the validator and refreshes the entity. A change
// of the type lands only while the question holds no value, in the one statement.
func (r *Question) Update(
	ctx context.Context,
	entity *model.Question,
	typeChanges bool,
	ifMatch *time.Time,
) error {
	args := questionArgs(entity)
	args[paramID] = entity.ID
	args[paramIfMatch] = ifMatch
	args["type_changes"] = typeChanges

	err := getNamed(ctx, r.tx, entity, `
		UPDATE questions q
		SET type = :type, prompt = :prompt, config = CAST(:config AS jsonb),
		    archived_at = :archived_at
		WHERE q.id = :id
		  AND (CAST(:if_match AS TIMESTAMPTZ) IS NULL OR q.updated_at = CAST(:if_match AS TIMESTAMPTZ))
		  AND (NOT :type_changes
		       OR NOT EXISTS (SELECT 1 FROM question_values v WHERE v.question_id = q.id))
		RETURNING q.updated_at;`,
		args,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return r.whyMissed(ctx, entity.ID, ifMatch)
	}

	if err != nil {
		return fmt.Errorf("updating Question: %w", err)
	}

	return nil
}

// Delete deletes the question under the validator. A question with a value stays.
func (r *Question) Delete(ctx context.Context, id string, ifMatch *time.Time) error {
	err := deleteGuarded(ctx, r.tx, "questions", id, ifMatch)
	if isForeignKeyViolation(err, questionValuesConstraint) {
		return errs.ErrQuestionHasValues
	}

	if err != nil {
		return fmt.Errorf("deleting Question: %w", err)
	}

	return nil
}

// whyMissed answers the reason that a conditional update of the question changed no
// row: no record, a stale validator, or a value that freezes the type.
func (r *Question) whyMissed(ctx context.Context, id string, ifMatch *time.Time) error {
	current, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if ifMatch != nil && !current.UpdatedAt.Equal(*ifMatch) {
		return precondition.ErrModified
	}

	return errs.ErrQuestionHasValues
}

func questionArgs(entity *model.Question) map[string]any {
	config := string(entity.Config)
	if config == "" {
		config = "{}"
	}

	return map[string]any{
		"type":        entity.Type,
		"prompt":      entity.Prompt,
		"config":      config,
		"archived_at": entity.ArchivedAt,
	}
}
