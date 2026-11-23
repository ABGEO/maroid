package repository

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/pulse/errs"
	"github.com/abgeo/maroid/plugins/pulse/model"
)

const valuePromptConstraint = "question_values_prompt_key"

// ValueRepository defines the data access contract for Value entities.
type ValueRepository interface {
	Insert(ctx context.Context, entity *model.Value) error
	List(
		ctx context.Context,
		questionID string,
		from, to time.Time,
		seek page.Seek,
	) ([]model.Value, error)
	Latest(ctx context.Context, questionIDs []string) ([]model.Value, error)
	Days(
		ctx context.Context,
		questionID string,
		from, to time.Time,
		zone *time.Location,
	) ([]model.Day, error)
}

// Value reads and writes the values of the acting user. It never changes a value.
type Value struct {
	tx *sqlx.Tx
}

var _ ValueRepository = (*Value)(nil)

// NewValue creates a Value repository over the transaction.
func NewValue(tx *sqlx.Tx) *Value {
	return &Value{tx: tx}
}

// Insert stores a new value and refreshes the entity with the stored values. A
// prompt that already holds a value answers errs.ErrPromptAnswered.
func (r *Value) Insert(ctx context.Context, entity *model.Value) error {
	query := `
		INSERT INTO question_values (
			question_id, type, occurred_at, answered_at, source, prompt_id,
			number_value, boolean_value, choice_id, text_value, latitude, longitude
		)
		VALUES (
			:question_id, :type, :occurred_at, :answered_at, :source, :prompt_id,
			:number_value, :boolean_value, :choice_id, :text_value, :latitude, :longitude
		)
		RETURNING id, created_at, updated_at;
	`

	err := getNamed(ctx, r.tx, entity, query, entity)
	if isUniqueViolation(err, valuePromptConstraint) {
		return errs.ErrPromptAnswered
	}

	if err != nil {
		return fmt.Errorf("inserting Value: %w", err)
	}

	return nil
}

const valueColumns = `v.id, v.question_id, v.type, v.occurred_at, v.answered_at, v.source,
	v.prompt_id, CAST(v.number_value AS float8) AS number_value, v.boolean_value, v.choice_id,
	v.text_value, v.latitude, v.longitude, v.created_at, v.updated_at`

// List answers one page of the values of the question with an occurrence from the
// moment from, before the moment to, newest first.
func (r *Value) List(
	ctx context.Context,
	questionID string,
	from time.Time,
	to time.Time,
	seek page.Seek,
) ([]model.Value, error) {
	condition, order := everyRow, "v.occurred_at DESC, v.id DESC"

	const boundary = "(SELECT b.occurred_at, b.id FROM question_values b WHERE b.id = CAST(:boundary AS uuid))"

	switch {
	case seek.Direction == page.DirectionBackward:
		condition, order = "(v.occurred_at, v.id) > "+boundary, "v.occurred_at, v.id"
	case seek.Boundary != "":
		condition = "(v.occurred_at, v.id) < " + boundary
	}

	var entities []model.Value

	err := selectNamed(ctx, r.tx, &entities, `
		SELECT `+valueColumns+` FROM question_values v
		WHERE v.question_id = :question_id AND v.occurred_at >= :from AND v.occurred_at < :to
		  AND `+condition+`
		ORDER BY `+order+` LIMIT :limit;`,
		map[string]any{
			paramQuestionID: questionID,
			paramFrom:       from,
			"to":            to,
			paramBoundary:   nullable(seek.Boundary),
			paramLimit:      seek.Limit,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("listing Values: %w", err)
	}

	if seek.Direction == page.DirectionBackward {
		slices.Reverse(entities)
	}

	return entities, nil
}

// Latest answers the newest value of each question that holds one.
func (r *Value) Latest(ctx context.Context, questionIDs []string) ([]model.Value, error) {
	if len(questionIDs) == 0 {
		return nil, nil
	}

	var entities []model.Value

	err := selectNamed(ctx, r.tx, &entities, `
		SELECT DISTINCT ON (v.question_id) `+valueColumns+`
		FROM question_values v
		WHERE v.question_id IN (:ids)
		ORDER BY v.question_id, v.occurred_at DESC, v.id DESC;`,
		map[string]any{paramIDs: questionIDs},
	)
	if err != nil {
		return nil, fmt.Errorf("reading the latest Values: %w", err)
	}

	return entities, nil
}

// Days groups the values of the question by the local day of their occurrence in
// the zone, from the day from to the day to, both included, oldest first.
func (r *Value) Days(
	ctx context.Context,
	questionID string,
	from time.Time,
	to time.Time,
	zone *time.Location,
) ([]model.Day, error) {
	var days []model.Day

	err := selectNamed(ctx, r.tx, &days, `
		SELECT to_char(CAST(v.occurred_at AT TIME ZONE :zone AS date), 'YYYY-MM-DD') AS day,
		       count(*) AS count,
		       CAST(avg(v.number_value) AS float8) AS mean,
		       CAST(min(v.number_value) AS float8) AS min,
		       CAST(max(v.number_value) AS float8) AS max,
		       count(*) FILTER (WHERE v.boolean_value) AS yes_count
		FROM question_values v
		WHERE v.question_id = :question_id
		  AND CAST(v.occurred_at AT TIME ZONE :zone AS date)
		      BETWEEN CAST(:from AS date) AND CAST(:to AS date)
		GROUP BY 1
		ORDER BY 1;`,
		map[string]any{
			paramQuestionID: questionID,
			"zone":          zone.String(),
			paramFrom:       from.Format(time.DateOnly),
			"to":            to.Format(time.DateOnly),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("grouping the Values by day: %w", err)
	}

	return days, nil
}
