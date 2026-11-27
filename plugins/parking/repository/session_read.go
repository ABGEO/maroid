package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/parking/model"
)

// lotUsageLimit is the count of parking lots of the overview.
const lotUsageLimit = 5

// matchesSessionFilter keeps the sessions of the filter. The alias is t.
//
// PARKHIST-DD-005, PARKHIST-DD-015: A free parking holds an amount of 0. From is included
// and to is excluded.
const matchesSessionFilter = `
	(CAST(:from AS timestamptz) IS NULL OR t.started_at >= CAST(:from AS timestamptz))
	AND (CAST(:to AS timestamptz) IS NULL OR t.started_at < CAST(:to AS timestamptz))
	AND (CAST(:plate AS text) IS NULL OR t.plate = CAST(:plate AS text))
	AND (CAST(:free AS boolean) IS NULL OR (t.amount = 0) = CAST(:free AS boolean))`

// Page answers one keyset read of the sessions of the filter, the latest start first.
//
// PARKHIST-FR-029, PARKHIST-FR-031: The latest start first, a page at a time.
func (r *Session) Page(
	ctx context.Context,
	seek page.Seek,
	filter model.SessionFilter,
) ([]model.Session, error) {
	condition, sorting := keyset(seek, "sessions", "started_at")
	query := `
		SELECT t.id, t.external_id, t.lot_number, t.address, t.plate, t.parking_type,
		       t.started_at, t.ended_at, t.amount
		FROM sessions t
		WHERE ` + matchesSessionFilter + `
		  AND ` + condition + `
		ORDER BY ` + sorting + `
		LIMIT :limit;`

	sessions, err := selectNamed[model.Session](
		ctx, r.tx, query, seekArguments(seek, sessionArguments(filter)),
	)
	if err != nil {
		return nil, fmt.Errorf("listing the sessions: %w", err)
	}

	return inOrder(seek, sessions), nil
}

// Total answers the count, the durations, and the amounts of every session of the filter.
//
// PARKHIST-FR-035: The total holds every match, not only the shown page.
func (r *Session) Total(
	ctx context.Context,
	filter model.SessionFilter,
) (model.SessionTotal, error) {
	query := `
		SELECT count(*) AS count,
		       CAST(coalesce(sum(extract(epoch FROM t.ended_at - t.started_at)), 0) AS bigint)
		           AS duration_seconds,
		       coalesce(sum(t.amount), 0) AS amount
		FROM sessions t
		WHERE ` + matchesSessionFilter + `;`

	totals, err := selectNamed[model.SessionTotal](ctx, r.tx, query, sessionArguments(filter))
	if err != nil {
		return model.SessionTotal{}, fmt.Errorf("summing the sessions: %w", err)
	}

	return totals[0], nil
}

// Plates answers each plate of a session of the acting user, in the order of the text.
func (r *Session) Plates(ctx context.Context) ([]string, error) {
	plates, err := selectNamed[string](ctx, r.tx,
		`SELECT DISTINCT t.plate FROM sessions t ORDER BY t.plate;`, map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("listing the plates: %w", err)
	}

	return plates, nil
}

// LotUsages answers the 5 parking lots with the most sessions from the point in time.
//
// PARKHIST-FR-028, PARKHIST-DD-014: A tie of the count goes to the later last session, and
// one lot keeps the address of its latest session.
func (r *Session) LotUsages(ctx context.Context, from *time.Time) ([]model.LotUsage, error) {
	const query = `
		SELECT t.lot_number,
		       (array_agg(t.address ORDER BY t.started_at DESC, t.id DESC))[1] AS address,
		       count(*) AS count,
		       coalesce(sum(t.amount), 0) AS amount
		FROM sessions t
		WHERE CAST(:from AS timestamptz) IS NULL OR t.started_at >= CAST(:from AS timestamptz)
		GROUP BY t.lot_number
		ORDER BY count(*) DESC, max(t.started_at) DESC
		LIMIT :limit;`

	usages, err := selectNamed[model.LotUsage](ctx, r.tx, query, map[string]any{
		paramFrom: pointOf(from),
		"limit":   lotUsageLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("counting the sessions of each parking lot: %w", err)
	}

	return usages, nil
}

func sessionArguments(filter model.SessionFilter) map[string]any {
	return map[string]any{
		paramFrom: pointOf(filter.From),
		"to":      pointOf(filter.To),
		"plate":   valueOf(filter.Plate),
		"free":    valueOf(filter.Free),
	}
}
