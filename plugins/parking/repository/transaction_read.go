package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/parking/model"
)

// matchesTransactionFilter keeps the transactions of the filter. The alias is t.
const matchesTransactionFilter = `
	(CAST(:from AS timestamptz) IS NULL OR t.occurred_at >= CAST(:from AS timestamptz))
	AND (CAST(:to AS timestamptz) IS NULL OR t.occurred_at < CAST(:to AS timestamptz))
	AND (CAST(:kind AS text) IS NULL OR t.kind = CAST(:kind AS text))`

// isCharge keeps the kinds that take an amount from the balance. The alias is t.
const isCharge = `t.kind IN ('parking_payment', 'permit_purchase')`

// transactionRow is a transaction with the parking lot of its linked session.
type transactionRow struct {
	model.Transaction

	LotNumber *string `db:"lot_number"`
	Address   *string `db:"address"`
}

// Page answers one keyset read of the transactions of the filter, the latest moment
// first, each with the parking lot of its linked session.
//
// The join finds the session through the policy, so a
// session of another user never links.
func (r *Transaction) Page(
	ctx context.Context,
	seek page.Seek,
	filter model.TransactionFilter,
) ([]model.Transaction, error) {
	condition, sorting := keyset(seek, "transactions", "occurred_at")
	query := `
		SELECT t.id, t.external_id, t.kind, t.service_kind, t.amount, t.balance_after,
		       t.occurred_at, t.session_external_id, t.description, s.lot_number, s.address
		FROM transactions t
		LEFT JOIN sessions s ON s.external_id = t.session_external_id
		WHERE ` + matchesTransactionFilter + `
		  AND ` + condition + `
		ORDER BY ` + sorting + `
		LIMIT :limit;`

	rows, err := selectNamed[transactionRow](
		ctx, r.tx, query, seekArguments(seek, transactionArguments(filter)),
	)
	if err != nil {
		return nil, fmt.Errorf("listing the transactions: %w", err)
	}

	transactions := make([]model.Transaction, 0, len(rows))

	for _, row := range rows {
		transaction := row.Transaction
		if row.LotNumber != nil && row.Address != nil {
			transaction.Session = &model.LotRef{LotNumber: *row.LotNumber, Address: *row.Address}
		}

		transactions = append(transactions, transaction)
	}

	return inOrder(seek, transactions), nil
}

// Total answers the count, the top ups, and the charges of every transaction of the
// filter.
func (r *Transaction) Total(
	ctx context.Context,
	filter model.TransactionFilter,
) (model.TransactionTotal, error) {
	query := `
		SELECT count(*) AS count,
		       coalesce(sum(t.amount) FILTER (WHERE t.kind = 'top_up'), 0) AS top_ups,
		       coalesce(sum(t.amount) FILTER (WHERE ` + isCharge + `), 0) AS charges
		FROM transactions t
		WHERE ` + matchesTransactionFilter + `;`

	totals, err := selectNamed[model.TransactionTotal](
		ctx, r.tx, query, transactionArguments(filter),
	)
	if err != nil {
		return model.TransactionTotal{}, fmt.Errorf("summing the transactions: %w", err)
	}

	return totals[0], nil
}

// ChargeMonths answers each month from the point in time that holds a transaction, the
// earliest first. The month is the month of the zone.
//
// The balance of a month is the balance after its
// latest transaction.
func (r *Transaction) ChargeMonths(
	ctx context.Context,
	from *time.Time,
	zone *time.Location,
) ([]model.ChargeMonth, error) {
	query := `
		SELECT to_char(date_trunc('month', t.occurred_at AT TIME ZONE :zone), 'YYYY-MM') AS month,
		       coalesce(sum(t.amount) FILTER (WHERE ` + isCharge + `), 0) AS charges,
		       (array_agg(t.balance_after ORDER BY t.occurred_at DESC, t.id DESC))[1] AS balance
		FROM transactions t
		WHERE CAST(:from AS timestamptz) IS NULL OR t.occurred_at >= CAST(:from AS timestamptz)
		GROUP BY 1
		ORDER BY 1;`

	months, err := selectNamed[model.ChargeMonth](ctx, r.tx, query, map[string]any{
		paramFrom: pointOf(from),
		"zone":    zone.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("summing the charges of each month: %w", err)
	}

	return months, nil
}

func transactionArguments(filter model.TransactionFilter) map[string]any {
	var kind any
	if filter.Kind != nil {
		kind = string(*filter.Kind)
	}

	return map[string]any{
		paramFrom: pointOf(filter.From),
		"to":      pointOf(filter.To),
		"kind":    kind,
	}
}
