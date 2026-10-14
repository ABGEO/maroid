package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/telmico/model"
)

const selectTransactions = `
	SELECT t.id, t.customer_number, t.operation, t.transaction_date, t.meter_value,
	       t."usage", t.amount, t.balance, t.tariff, t.meter_serial, t.due_date
	FROM transactions t`

// isCharge matches a transaction that model.Transaction.IsCharge reports as a charge.
const isCharge = `t.meter_value IS NOT NULL AND t.amount IS NOT NULL`

// TransactionRepository defines the data access contract for the transactions of the
// acting workspace.
type TransactionRepository interface {
	Insert(ctx context.Context, transaction *model.Transaction) (bool, error)
	LatestCharges(ctx context.Context, customerNumber string) ([]model.Transaction, error)
	ChargesOn(
		ctx context.Context,
		customerNumber string,
		day time.Time,
	) ([]model.Transaction, error)
	List(
		ctx context.Context,
		seek page.Seek,
		customerNumber string,
		from *time.Time,
	) ([]model.Transaction, error)
}

// Transaction reads and writes the transactions of the acting workspace within one
// transaction of the database.
type Transaction struct {
	tx *sqlx.Tx
}

var _ TransactionRepository = (*Transaction)(nil)

// NewTransaction creates a new Transaction over the transaction of the database.
func NewTransaction(tx *sqlx.Tx) *Transaction {
	return &Transaction{tx: tx}
}

// Insert stores the transaction, and answers true when the workspace held no
// transaction of that kind of the customer on that date with that meter value. A held
// transaction takes the values of the new one when a value changed. Insert sets the
// identifier of a row that it stores for the first time.
func (r *Transaction) Insert(ctx context.Context, transaction *model.Transaction) (bool, error) {
	const query = `
		INSERT INTO transactions (
			customer_number, operation, transaction_date, meter_value, "usage", amount,
			balance, tariff, meter_serial, due_date
		)
		VALUES (
			:customer_number, :operation, CAST(:transaction_date AS date), :meter_value, :usage,
			:amount, :balance, :tariff, :meter_serial, CAST(:due_date AS date)
		)
		ON CONFLICT ON CONSTRAINT transactions_identity_key DO UPDATE
		SET "usage"      = EXCLUDED."usage",
		    amount       = EXCLUDED.amount,
		    balance      = EXCLUDED.balance,
		    tariff       = EXCLUDED.tariff,
		    meter_serial = EXCLUDED.meter_serial,
		    due_date     = EXCLUDED.due_date,
		    collected_at = now()
		WHERE (transactions."usage", transactions.amount, transactions.balance,
		       transactions.tariff, transactions.meter_serial, transactions.due_date)
		      IS DISTINCT FROM
		      (EXCLUDED."usage", EXCLUDED.amount, EXCLUDED.balance,
		       EXCLUDED.tariff, EXCLUDED.meter_serial, EXCLUDED.due_date)
		RETURNING id, (xmax = 0) AS inserted;`

	bound, values, err := sqlx.Named(query, transactionArguments(transaction))
	if err != nil {
		return false, fmt.Errorf("binding the transaction: %w", err)
	}

	var (
		id       string
		inserted bool
	)

	err = r.tx.QueryRowxContext(ctx, r.tx.Rebind(bound), values...).Scan(&id, &inserted)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("upserting the transaction: %w", err)
	}

	if inserted {
		transaction.ID = id
	}

	return inserted, nil
}

func transactionArguments(transaction *model.Transaction) map[string]any {
	var dueDate any
	if transaction.DueDate != nil {
		dueDate = transaction.DueDate.Format(time.DateOnly)
	}

	return map[string]any{
		paramCustomerNumber: transaction.CustomerNumber,
		"operation":         transaction.Operation,
		"transaction_date":  transaction.Date.Format(time.DateOnly),
		"meter_value":       transaction.MeterValue,
		"usage":             transaction.Usage,
		"amount":            transaction.Amount,
		"balance":           transaction.Balance,
		"tariff":            transaction.Tariff,
		"meter_serial":      transaction.MeterSerial,
		"due_date":          dueDate,
	}
}

// LatestCharges answers the charges of the latest date on which the customer holds a
// charge, or none.
func (r *Transaction) LatestCharges(
	ctx context.Context,
	customerNumber string,
) ([]model.Transaction, error) {
	const query = selectTransactions + `
		WHERE t.customer_number = :customer_number
		  AND ` + isCharge + `
		  AND t.transaction_date = (
		      SELECT max(t.transaction_date) FROM transactions t
		      WHERE t.customer_number = :customer_number AND ` + isCharge + `)
		ORDER BY t.meter_value;`

	charges, err := selectNamed[model.Transaction](ctx, r.tx, query, map[string]any{
		paramCustomerNumber: customerNumber,
	})
	if err != nil {
		return nil, fmt.Errorf("selecting the latest charges: %w", err)
	}

	return charges, nil
}

// ChargesOn answers the charges of the customer on the date.
func (r *Transaction) ChargesOn(
	ctx context.Context,
	customerNumber string,
	day time.Time,
) ([]model.Transaction, error) {
	const query = selectTransactions + `
		WHERE t.customer_number = :customer_number
		  AND t.transaction_date = CAST(:day AS date)
		  AND ` + isCharge + `
		ORDER BY t.meter_value;`

	charges, err := selectNamed[model.Transaction](ctx, r.tx, query, map[string]any{
		paramCustomerNumber: customerNumber,
		"day":               day.Format(time.DateOnly),
	})
	if err != nil {
		return nil, fmt.Errorf("selecting the charges of a date: %w", err)
	}

	return charges, nil
}

// List answers one keyset read of the transactions of the customer, the latest date
// first, with the identifier as the tie. A non-nil from keeps the transactions of that
// date and after.
func (r *Transaction) List(
	ctx context.Context,
	seek page.Seek,
	customerNumber string,
	from *time.Time,
) ([]model.Transaction, error) {
	condition, sorting := keyset(seek, "transactions", "transaction_date")
	query := selectTransactions + `
		WHERE t.customer_number = :customer_number
		  AND (CAST(:from AS date) IS NULL OR t.transaction_date >= CAST(:from AS date))
		  AND ` + condition + `
		ORDER BY ` + sorting + `
		LIMIT :limit;`

	var bound any
	if from != nil {
		bound = from.Format(time.DateOnly)
	}

	transactions, err := selectNamed[model.Transaction](ctx, r.tx, query, map[string]any{
		paramCustomerNumber: customerNumber,
		"from":              bound,
		"boundary":          boundaryOf(seek),
		"limit":             seek.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("listing the transactions: %w", err)
	}

	return inOrder(seek, transactions), nil
}
