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
	"github.com/abgeo/maroid/plugins/pulse/model"
)

const roundColumns = `
	id, checkin_id, checkin_name, occurrence_at, state, start_message_id, created_at, updated_at,
	(SELECT count(*) FROM prompts p WHERE p.round_id = rounds.id) AS question_count`

// RoundRepository defines the data access contract for Round entities.
type RoundRepository interface {
	Insert(ctx context.Context, entity *model.Round) (bool, error)
	GetByID(ctx context.Context, id string) (*model.Round, error)
	GetByStartMessage(ctx context.Context, messageID string) (*model.Round, error)
	List(ctx context.Context, seek page.Seek, filter RoundFilter) ([]model.Round, error)
	ListBetween(ctx context.Context, from, to time.Time) ([]model.Round, error)
	Move(
		ctx context.Context,
		id string,
		from model.RoundState,
		to model.RoundState,
	) (*model.Round, error)
	ExpireOpen(ctx context.Context, before *time.Time) ([]model.Round, error)
	SetStartMessage(ctx context.Context, id string, messageID string) error
}

// Round reads and writes the rounds of the acting user.
type Round struct {
	tx *sqlx.Tx
}

var _ RoundRepository = (*Round)(nil)

// NewRound creates a Round repository over the transaction.
func NewRound(tx *sqlx.Tx) *Round {
	return &Round{tx: tx}
}

// Insert stores a new offered round and refreshes the entity with the stored values.
// It answers false and stores nothing when the occurrence of the check-in already
// holds a round.
func (r *Round) Insert(ctx context.Context, entity *model.Round) (bool, error) {
	query := `
		INSERT INTO rounds (checkin_id, checkin_name, occurrence_at)
		VALUES (:checkin_id, :checkin_name, :occurrence_at)
		ON CONFLICT ON CONSTRAINT rounds_occurrence_key DO NOTHING
		RETURNING id, state, created_at, updated_at;
	`

	err := getNamed(ctx, r.tx, entity, query, entity)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("inserting Round: %w", err)
	}

	return true, nil
}

// GetByID answers one round, or sql.ErrNoRows.
func (r *Round) GetByID(ctx context.Context, id string) (*model.Round, error) {
	var entity model.Round

	err := getNamed(ctx, r.tx, &entity, "SELECT "+roundColumns+" FROM rounds WHERE id = :id;",
		map[string]any{paramID: id})
	if err != nil {
		return nil, fmt.Errorf("getting Round: %w", err)
	}

	return &entity, nil
}

// GetByStartMessage answers the round whose start prompt is the message, or
// sql.ErrNoRows.
func (r *Round) GetByStartMessage(ctx context.Context, messageID string) (*model.Round, error) {
	var entity model.Round

	err := getNamed(ctx, r.tx, &entity,
		"SELECT "+roundColumns+" FROM rounds WHERE start_message_id = :message_id;",
		map[string]any{paramMessageID: messageID})
	if err != nil {
		return nil, fmt.Errorf("getting Round by its start prompt: %w", err)
	}

	return &entity, nil
}

// ListBetween answers the rounds whose occurrence lies at from or later and before
// to, oldest first, then by the name of the check-in.
func (r *Round) ListBetween(ctx context.Context, from, to time.Time) ([]model.Round, error) {
	var entities []model.Round

	err := selectNamed(ctx, r.tx, &entities, `
		SELECT `+roundColumns+` FROM rounds
		WHERE occurrence_at >= :from AND occurrence_at < :to
		ORDER BY occurrence_at, checkin_name;`,
		map[string]any{paramFrom: from, "to": to},
	)
	if err != nil {
		return nil, fmt.Errorf("listing Rounds between two moments: %w", err)
	}

	return entities, nil
}

// Move changes the state of the round from one state to another, and answers the
// round. It answers sql.ErrNoRows when the round no longer holds the state from,
// because another press or a tick moved it first.
func (r *Round) Move(
	ctx context.Context,
	id string,
	from model.RoundState,
	to model.RoundState,
) (*model.Round, error) {
	var entity model.Round

	err := getNamed(ctx, r.tx, &entity, `
		UPDATE rounds SET state = :to
		WHERE id = :id AND state = :from
		RETURNING `+roundColumns+`;`,
		map[string]any{paramID: id, paramFrom: from, "to": to},
	)
	if err != nil {
		return nil, fmt.Errorf("moving Round: %w", err)
	}

	return &entity, nil
}

// ExpireOpen marks every offered or started round expired, and answers them. A
// moment keeps the rounds at or after it open.
func (r *Round) ExpireOpen(ctx context.Context, before *time.Time) ([]model.Round, error) {
	var entities []model.Round

	err := selectNamed(ctx, r.tx, &entities, `
		UPDATE rounds SET state = 'expired'
		WHERE state IN ('offered', 'started')
		  AND (CAST(:before AS TIMESTAMPTZ) IS NULL OR occurrence_at < CAST(:before AS TIMESTAMPTZ))
		RETURNING `+roundColumns+`;`,
		map[string]any{"before": before},
	)
	if err != nil {
		return nil, fmt.Errorf("expiring the open Rounds: %w", err)
	}

	return entities, nil
}

// SetStartMessage stores the message of the start prompt of the round.
func (r *Round) SetStartMessage(ctx context.Context, id string, messageID string) error {
	_, err := execNamed(
		ctx,
		r.tx,
		"UPDATE rounds SET start_message_id = :message_id WHERE id = :id;",
		map[string]any{paramID: id, paramMessageID: messageID},
	)
	if err != nil {
		return fmt.Errorf("storing the start prompt of the Round: %w", err)
	}

	return nil
}

// RoundFilter narrows a list of rounds. An empty member narrows nothing.
type RoundFilter struct {
	CheckinID string
	State     model.RoundState
}

// List answers one page of the rounds that the filter keeps, newest first.
func (r *Round) List(
	ctx context.Context,
	seek page.Seek,
	filter RoundFilter,
) ([]model.Round, error) {
	condition, order := everyRow, "occurrence_at DESC, id DESC"

	const boundary = "(SELECT b.occurrence_at, b.id FROM rounds b WHERE b.id = CAST(:boundary AS uuid))"

	switch {
	case seek.Direction == page.DirectionBackward:
		condition, order = "(occurrence_at, id) > "+boundary, "occurrence_at, id"
	case seek.Boundary != "":
		condition = "(occurrence_at, id) < " + boundary
	}

	var entities []model.Round

	err := selectNamed(ctx, r.tx, &entities, `
		SELECT `+roundColumns+` FROM rounds
		WHERE (CAST(:checkin_id AS uuid) IS NULL OR checkin_id = CAST(:checkin_id AS uuid))
		  AND (CAST(:state AS text) IS NULL OR state = CAST(:state AS text))
		  AND `+condition+`
		ORDER BY `+order+` LIMIT :limit;`,
		map[string]any{
			"checkin_id":  nullable(filter.CheckinID),
			"state":       nullable(string(filter.State)),
			paramBoundary: nullable(seek.Boundary),
			paramLimit:    seek.Limit,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("listing Rounds: %w", err)
	}

	if seek.Direction == page.DirectionBackward {
		slices.Reverse(entities)
	}

	return entities, nil
}
