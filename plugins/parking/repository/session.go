package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/parking/model"
)

// Session reads and writes the sessions of the acting user within one database
// transaction.
type Session struct {
	tx *sqlx.Tx
}

// NewSession creates a new Session over the database transaction.
func NewSession(tx *sqlx.Tx) *Session {
	return &Session{tx: tx}
}

// Known answers the external identifiers among ids that the acting user holds.
func (r *Session) Known(ctx context.Context, ids []int64) (map[int64]bool, error) {
	known, err := knownOf(ctx, r.tx, "sessions", ids)
	if err != nil {
		return nil, fmt.Errorf("reading the held sessions: %w", err)
	}

	return known, nil
}

// Upsert stores the sessions in one statement and answers the count of the new rows. A
// held session takes the values of the parking service, and keeps its collected_at when
// no value changed.
//
// PARKHIST-FR-011: One session for one identifier of the parking service.
func (r *Session) Upsert(ctx context.Context, sessions []model.Session) (int, error) {
	if len(sessions) == 0 {
		return 0, nil
	}

	const query = `
		INSERT INTO sessions
		    (external_id, lot_number, address, plate, parking_type, started_at, ended_at, amount)
		SELECT external_id, lot_number, address, plate, parking_type, started_at, ended_at, amount
		FROM unnest(
		    CAST(:external_ids AS bigint[]), CAST(:lot_numbers AS text[]),
		    CAST(:addresses AS text[]), CAST(:plates AS text[]),
		    CAST(:parking_types AS text[]), CAST(:started_ats AS timestamptz[]),
		    CAST(:ended_ats AS timestamptz[]), CAST(:amounts AS numeric[]))
		     AS answered (external_id, lot_number, address, plate, parking_type, started_at,
		                  ended_at, amount)
		ON CONFLICT ON CONSTRAINT sessions_user_external_id_key DO UPDATE
		SET lot_number   = EXCLUDED.lot_number,
		    address      = EXCLUDED.address,
		    plate        = EXCLUDED.plate,
		    parking_type = EXCLUDED.parking_type,
		    started_at   = EXCLUDED.started_at,
		    ended_at     = EXCLUDED.ended_at,
		    amount       = EXCLUDED.amount,
		    collected_at = now()
		WHERE (sessions.lot_number, sessions.address, sessions.plate, sessions.parking_type,
		       sessions.started_at, sessions.ended_at, sessions.amount)
		      IS DISTINCT FROM
		      (EXCLUDED.lot_number, EXCLUDED.address, EXCLUDED.plate, EXCLUDED.parking_type,
		       EXCLUDED.started_at, EXCLUDED.ended_at, EXCLUDED.amount)
		RETURNING (xmax = 0) AS inserted;`

	flags, err := selectNamed[bool](ctx, r.tx, query, sessionArguments(sessions))
	if err != nil {
		return 0, fmt.Errorf("upserting the sessions: %w", err)
	}

	return insertedOf(flags), nil
}

func sessionArguments(sessions []model.Session) map[string]any {
	externalIDs := make([]int64, len(sessions))
	lotNumbers := make([]string, len(sessions))
	addresses := make([]string, len(sessions))
	plates := make([]string, len(sessions))
	parkingTypes := make([]string, len(sessions))
	startedAts := make([]time.Time, len(sessions))
	endedAts := make([]time.Time, len(sessions))
	amounts := make([]float64, len(sessions))

	for i, session := range sessions {
		externalIDs[i] = session.ExternalID
		lotNumbers[i] = session.LotNumber
		addresses[i] = session.Address
		plates[i] = session.Plate
		parkingTypes[i] = session.ParkingType
		startedAts[i] = session.StartedAt
		endedAts[i] = session.EndedAt
		amounts[i] = session.Amount
	}

	return map[string]any{
		"external_ids":  externalIDs,
		"lot_numbers":   lotNumbers,
		"addresses":     addresses,
		"plates":        plates,
		"parking_types": parkingTypes,
		"started_ats":   startedAts,
		"ended_ats":     endedAts,
		"amounts":       amounts,
	}
}
