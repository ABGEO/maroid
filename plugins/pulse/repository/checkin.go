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
	"github.com/abgeo/maroid/plugins/pulse/errs"
	"github.com/abgeo/maroid/plugins/pulse/model"
)

// checkinColumns reads the arrays as comma lists, which model.Clocks, model.Weekdays,
// and model.IDs scan. A clock reads as the text of a TIME cut to HH:MM, because
// sqlx takes the colon of a format such as HH24:MI for a named parameter.
const checkinColumns = `
	c.id, c.name,
	array_to_string(ARRAY(
		SELECT substr(CAST(t.clock AS text), 1, 5)
		FROM unnest(c.local_times) WITH ORDINALITY AS t (clock, place)
		ORDER BY t.place), ',') AS local_times,
	array_to_string(c.weekdays, ',') AS weekdays,
	c.enabled, c.fires_after, c.created_at, c.updated_at`

const selectCheckins = `
	SELECT ` + checkinColumns + `,
	       array_to_string(ARRAY(
	           SELECT CAST(cq.question_id AS text) FROM checkin_questions cq
	           WHERE cq.checkin_id = c.id ORDER BY cq.position), ',') AS question_ids
	FROM checkins c`

// CheckinRepository defines the data access contract for Checkin entities.
type CheckinRepository interface {
	Insert(ctx context.Context, entity *model.Checkin) error
	GetByID(ctx context.Context, id string) (*model.Checkin, error)
	List(ctx context.Context, seek page.Seek) ([]model.Checkin, error)
	ListDue(ctx context.Context) ([]model.Checkin, error)
	Update(ctx context.Context, entity *model.Checkin, ifMatch *time.Time) error
	Delete(ctx context.Context, id string, ifMatch *time.Time) error
	SetFiresAfter(ctx context.Context, ids []string, moment time.Time) error
	SetEveryFiresAfter(ctx context.Context, moment time.Time) error
}

// Checkin reads and writes the check-ins of the acting user.
type Checkin struct {
	tx *sqlx.Tx
}

var _ CheckinRepository = (*Checkin)(nil)

// NewCheckin creates a Checkin repository over the transaction.
func NewCheckin(tx *sqlx.Tx) *Checkin {
	return &Checkin{tx: tx}
}

// Insert stores a new check-in with its questions and refreshes the entity.
func (r *Checkin) Insert(ctx context.Context, entity *model.Checkin) error {
	err := getNamed(ctx, r.tx, entity, `
		INSERT INTO checkins (name, local_times, weekdays, enabled, fires_after)
		VALUES (:name, CAST(string_to_array(CAST(:local_times AS text), ',') AS time[]),
		        CAST(string_to_array(CAST(:weekdays AS text), ',') AS smallint[]),
		        :enabled, :fires_after)
		RETURNING id, created_at, updated_at;`,
		checkinArgs(entity),
	)
	if err != nil {
		return fmt.Errorf("inserting Checkin: %w", err)
	}

	return r.replaceQuestions(ctx, entity)
}

// GetByID answers one check-in with its questions, or sql.ErrNoRows.
func (r *Checkin) GetByID(ctx context.Context, id string) (*model.Checkin, error) {
	var entity model.Checkin

	err := getNamed(ctx, r.tx, &entity, selectCheckins+" WHERE c.id = :id;",
		map[string]any{paramID: id})
	if err != nil {
		return nil, fmt.Errorf("getting Checkin: %w", err)
	}

	return &entity, nil
}

// List answers one page of the check-ins, sorted by the first local time, then by id.
func (r *Checkin) List(ctx context.Context, seek page.Seek) ([]model.Checkin, error) {
	const boundary = "(SELECT b.local_times[1], b.id FROM checkins b WHERE b.id = CAST(:boundary AS uuid))"

	condition, order := everyRow, "c.local_times[1], c.id"

	switch {
	case seek.Direction == page.DirectionBackward:
		condition = "(c.local_times[1], c.id) < " + boundary
		order = "c.local_times[1] DESC, c.id DESC"
	case seek.Boundary != "":
		condition = "(c.local_times[1], c.id) > " + boundary
	}

	var entities []model.Checkin

	err := selectNamed(ctx, r.tx, &entities,
		selectCheckins+" WHERE "+condition+" ORDER BY "+order+" LIMIT :limit;",
		map[string]any{paramBoundary: nullable(seek.Boundary), paramLimit: seek.Limit},
	)
	if err != nil {
		return nil, fmt.Errorf("listing Checkins: %w", err)
	}

	if seek.Direction == page.DirectionBackward {
		slices.Reverse(entities)
	}

	return entities, nil
}

// ListDue answers the enabled check-ins that ask at least one question that is not
// archived. QuestionIDs holds only those questions, in the order of the round.
func (r *Checkin) ListDue(ctx context.Context) ([]model.Checkin, error) {
	var entities []model.Checkin

	err := r.tx.SelectContext(ctx, &entities, `
		SELECT `+checkinColumns+`,
		       array_to_string(ARRAY(
		           SELECT CAST(cq.question_id AS text)
		           FROM checkin_questions cq
		           JOIN questions q ON q.id = cq.question_id
		           WHERE cq.checkin_id = c.id AND q.archived_at IS NULL
		           ORDER BY cq.position), ',') AS question_ids
		FROM checkins c
		WHERE c.enabled
		  AND EXISTS (
		      SELECT 1
		      FROM checkin_questions cq
		      JOIN questions q ON q.id = cq.question_id
		      WHERE cq.checkin_id = c.id AND q.archived_at IS NULL)
		ORDER BY c.name;`)
	if err != nil {
		return nil, fmt.Errorf("listing the due Checkins: %w", err)
	}

	return entities, nil
}

// Update writes the check-in and its questions under the validator, and refreshes
// the entity.
func (r *Checkin) Update(ctx context.Context, entity *model.Checkin, ifMatch *time.Time) error {
	args := checkinArgs(entity)
	args[paramID] = entity.ID
	args[paramIfMatch] = ifMatch

	err := getNamed(ctx, r.tx, entity, `
		UPDATE checkins
		SET name = :name,
		    local_times = CAST(string_to_array(CAST(:local_times AS text), ',') AS time[]),
		    weekdays = CAST(string_to_array(CAST(:weekdays AS text), ',') AS smallint[]),
		    enabled = :enabled, fires_after = :fires_after
		WHERE id = :id
		  AND (CAST(:if_match AS TIMESTAMPTZ) IS NULL OR updated_at = CAST(:if_match AS TIMESTAMPTZ))
		RETURNING updated_at;`,
		args,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return missed(ifMatch)
	}

	if err != nil {
		return fmt.Errorf("updating Checkin: %w", err)
	}

	return r.replaceQuestions(ctx, entity)
}

// Delete deletes the check-in under the validator. Its rounds and values stay.
func (r *Checkin) Delete(ctx context.Context, id string, ifMatch *time.Time) error {
	if err := deleteGuarded(ctx, r.tx, "checkins", id, ifMatch); err != nil {
		return fmt.Errorf("deleting Checkin: %w", err)
	}

	return nil
}

// SetFiresAfter moves the mark of the check-ins to the moment.
func (r *Checkin) SetFiresAfter(ctx context.Context, ids []string, moment time.Time) error {
	if len(ids) == 0 {
		return nil
	}

	_, err := execNamed(ctx, r.tx, "UPDATE checkins SET fires_after = :moment WHERE id IN (:ids);",
		map[string]any{paramIDs: ids, "moment": moment})
	if err != nil {
		return fmt.Errorf("moving the mark of the Checkins: %w", err)
	}

	return nil
}

// SetEveryFiresAfter moves the mark of every check-in of the person to the moment.
func (r *Checkin) SetEveryFiresAfter(ctx context.Context, moment time.Time) error {
	_, err := execNamed(ctx, r.tx, "UPDATE checkins SET fires_after = :moment;",
		map[string]any{"moment": moment})
	if err != nil {
		return fmt.Errorf("moving the mark of every Checkin: %w", err)
	}

	return nil
}

// replaceQuestions writes the questions of the check-in in the order of the list. A
// question that is archived or unknown refuses the write.
func (r *Checkin) replaceQuestions(ctx context.Context, entity *model.Checkin) error {
	var usable []string

	err := selectNamed(ctx, r.tx, &usable,
		"SELECT CAST(id AS text) FROM questions WHERE id IN (:ids) AND archived_at IS NULL;",
		map[string]any{paramIDs: []string(entity.QuestionIDs)})
	if err != nil {
		return fmt.Errorf("reading the questions of the Checkin: %w", err)
	}

	for index, id := range entity.QuestionIDs {
		if !slices.Contains(usable, id) {
			return &errs.QuestionUnusableError{Index: index}
		}
	}

	_, err = execNamed(ctx, r.tx, "DELETE FROM checkin_questions WHERE checkin_id = :id;",
		map[string]any{paramID: entity.ID})
	if err != nil {
		return fmt.Errorf("clearing the questions of the Checkin: %w", err)
	}

	_, err = execNamed(ctx, r.tx, `
		INSERT INTO checkin_questions (checkin_id, question_id, position)
		SELECT CAST(:id AS uuid), CAST(chosen.question_id AS uuid), chosen.place
		FROM unnest(string_to_array(CAST(:question_ids AS text), ','))
		     WITH ORDINALITY AS chosen (question_id, place);`,
		map[string]any{paramID: entity.ID, "question_ids": entity.QuestionIDs},
	)
	if err != nil {
		return fmt.Errorf("writing the questions of the Checkin: %w", err)
	}

	return nil
}

func checkinArgs(entity *model.Checkin) map[string]any {
	return map[string]any{
		"name":        entity.Name,
		"local_times": entity.LocalTimes,
		"weekdays":    entity.Weekdays,
		"enabled":     entity.Enabled,
		"fires_after": entity.FiresAfter,
	}
}
