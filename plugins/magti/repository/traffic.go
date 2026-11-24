package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/magti/model"
)

// TrafficRepository defines the data access contract for the traffic of the acting
// workspace.
type TrafficRepository interface {
	Upsert(ctx context.Context, day *model.TrafficDay) error
	List(
		ctx context.Context,
		seek page.Seek,
		number string,
		period model.Period,
		order Order,
	) ([]model.TrafficDay, error)
}

// Traffic reads and writes the traffic of the acting workspace within one transaction.
type Traffic struct {
	tx *sqlx.Tx
}

var _ TrafficRepository = (*Traffic)(nil)

// NewTraffic creates a new Traffic over the transaction.
func NewTraffic(tx *sqlx.Tx) *Traffic {
	return &Traffic{tx: tx}
}

// Upsert stores the traffic of the day, and replaces every value of a day that the
// workspace holds. It sets the identifier of the row.
func (r *Traffic) Upsert(ctx context.Context, day *model.TrafficDay) error {
	const query = `
		INSERT INTO traffic_days (
			subscriber_number, day, data_mb, call_in_seconds, call_out_seconds,
			sms_received, sms_sent
		)
		VALUES (
			:subscriber_number, CAST(:day AS date), :data_mb, :call_in_seconds,
			:call_out_seconds, :sms_received, :sms_sent
		)
		ON CONFLICT ON CONSTRAINT traffic_days_identity_key DO UPDATE
		SET data_mb          = EXCLUDED.data_mb,
		    call_in_seconds  = EXCLUDED.call_in_seconds,
		    call_out_seconds = EXCLUDED.call_out_seconds,
		    sms_received     = EXCLUDED.sms_received,
		    sms_sent         = EXCLUDED.sms_sent,
		    collected_at     = now()
		RETURNING id;`

	bound, values, err := sqlx.Named(query, map[string]any{
		paramSubscriberNumber: day.SubscriberNumber,
		"day":                 day.Day.Format(time.DateOnly),
		"data_mb":             day.DataMB,
		"call_in_seconds":     day.CallInSeconds,
		"call_out_seconds":    day.CallOutSeconds,
		"sms_received":        day.SMSReceived,
		"sms_sent":            day.SMSSent,
	})
	if err != nil {
		return fmt.Errorf("binding the traffic of a day: %w", err)
	}

	if err = r.tx.GetContext(ctx, &day.ID, r.tx.Rebind(bound), values...); err != nil {
		return fmt.Errorf("upserting the traffic of a day: %w", err)
	}

	return nil
}

// List answers one keyset read of the traffic of the subscriber in the period.
func (r *Traffic) List(
	ctx context.Context,
	seek page.Seek,
	number string,
	period model.Period,
	order Order,
) ([]model.TrafficDay, error) {
	condition, sorting := keyset(seek, "traffic_days", "day", order)
	query := `
		SELECT t.id, t.subscriber_number, t.day, t.data_mb, t.call_in_seconds,
		       t.call_out_seconds, t.sms_received, t.sms_sent
		FROM traffic_days t
		WHERE t.subscriber_number = :subscriber_number
		  AND t.day BETWEEN CAST(:from AS date) AND CAST(:to AS date)
		  AND ` + condition + `
		ORDER BY ` + sorting + `
		LIMIT :limit;`

	days, err := selectNamed[model.TrafficDay](ctx, r.tx, query, map[string]any{
		paramSubscriberNumber: number,
		"from":                period.From.String(),
		"to":                  period.To.String(),
		"boundary":            boundaryOf(seek),
		"limit":               seek.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("listing the traffic: %w", err)
	}

	return inOrder(seek, days), nil
}
