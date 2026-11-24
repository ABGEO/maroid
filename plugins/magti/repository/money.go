package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/magti/model"
)

// MoneyRepository defines the data access contract for the money entries of the acting
// workspace.
type MoneyRepository interface {
	Insert(ctx context.Context, entry *model.MoneyEntry) (bool, error)
	List(
		ctx context.Context,
		seek page.Seek,
		number string,
		order Order,
	) ([]model.MoneyEntry, error)
	MonthTotals(ctx context.Context, number string, from model.Month) ([]model.MonthTotal, error)
}

// Money reads and writes the money entries of the acting workspace within one
// transaction.
type Money struct {
	tx *sqlx.Tx
}

var _ MoneyRepository = (*Money)(nil)

// NewMoney creates a new Money over the transaction.
func NewMoney(tx *sqlx.Tx) *Money {
	return &Money{tx: tx}
}

// Insert stores the entry, and answers true when the workspace held no entry of that
// moment, kind, direction, and amount. Insert sets the identifier of a row that it stores
// for the first time.
func (r *Money) Insert(ctx context.Context, entry *model.MoneyEntry) (bool, error) {
	const query = `
		INSERT INTO money_entries (subscriber_number, occurred_at, direction, kind, amount)
		VALUES (
			:subscriber_number, :occurred_at, :direction, :kind,
			round(CAST(:amount AS numeric) / 100, 2)
		)
		ON CONFLICT ON CONSTRAINT money_entries_identity_key DO NOTHING
		RETURNING id;`

	bound, values, err := sqlx.Named(query, map[string]any{
		paramSubscriberNumber: entry.SubscriberNumber,
		"occurred_at":         entry.OccurredAt,
		"direction":           string(entry.Direction),
		"kind":                entry.Kind,
		"amount":              int64(entry.Amount),
	})
	if err != nil {
		return false, fmt.Errorf("binding the money entry: %w", err)
	}

	var id string

	err = r.tx.GetContext(ctx, &id, r.tx.Rebind(bound), values...)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("inserting the money entry: %w", err)
	}

	entry.ID = id

	return true, nil
}

// List answers one keyset read of the money entries of the subscriber.
func (r *Money) List(
	ctx context.Context,
	seek page.Seek,
	number string,
	order Order,
) ([]model.MoneyEntry, error) {
	condition, sorting := keyset(seek, "money_entries", "occurred_at", order)
	query := `
		SELECT t.id, t.subscriber_number, t.occurred_at, t.direction, t.kind,
		       CAST(round(t.amount * 100) AS bigint) AS amount
		FROM money_entries t
		WHERE t.subscriber_number = :subscriber_number
		  AND ` + condition + `
		ORDER BY ` + sorting + `
		LIMIT :limit;`

	entries, err := selectNamed[model.MoneyEntry](ctx, r.tx, query, map[string]any{
		paramSubscriberNumber: number,
		"boundary":            boundaryOf(seek),
		"limit":               seek.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("listing the money entries: %w", err)
	}

	return inOrder(seek, entries), nil
}

// monthTotalRow is one month of MonthTotals as the database answers it.
type monthTotalRow struct {
	Month   string      `db:"month"`
	TopUps  model.Tetri `db:"top_ups"`
	Charges model.Tetri `db:"charges"`
}

// MonthTotals answers the sums of each month of Tbilisi from the month on that holds a
// money entry of the subscriber, the earliest first.
func (r *Money) MonthTotals(
	ctx context.Context,
	number string,
	from model.Month,
) ([]model.MonthTotal, error) {
	const query = `
		SELECT to_char((t.occurred_at + interval '4 hours') AT TIME ZONE 'UTC', 'YYYY-MM') AS month,
		       CAST(round(COALESCE(sum(t.amount) FILTER (WHERE t.direction = 'in'), 0) * 100) AS bigint) AS top_ups,
		       CAST(round(COALESCE(sum(t.amount) FILTER (WHERE t.direction = 'out'), 0) * 100) AS bigint) AS charges
		FROM money_entries t
		WHERE t.subscriber_number = :subscriber_number
		  AND t.occurred_at >= CAST(:from AS timestamptz)
		GROUP BY 1
		ORDER BY 1;`

	rows, err := selectNamed[monthTotalRow](ctx, r.tx, query, map[string]any{
		paramSubscriberNumber: number,
		"from":                from.First().Start(),
	})
	if err != nil {
		return nil, fmt.Errorf("summing the money entries: %w", err)
	}

	totals := make([]model.MonthTotal, 0, len(rows))

	for _, row := range rows {
		month, err := model.ParseMonth(row.Month)
		if err != nil {
			return nil, fmt.Errorf("reading the month of a total: %w", err)
		}

		totals = append(
			totals,
			model.MonthTotal{Month: month, TopUps: row.TopUps, Charges: row.Charges},
		)
	}

	return totals, nil
}
