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
	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
)

// paramCustomerNumber names the parameter of the customer in every query of the plugin.
const paramCustomerNumber = "customer_number"

// selectTransactions selects the transactions with the name of their operation.
const selectTransactions = `
	SELECT t.id, t.customer_number, t.transaction_type_id, tt.name AS operation,
	       t.transaction_date, t.consumption, t.meter_value, t.amount, t.balance,
	       t.bill_path, t.meter_photo_path
	FROM transactions t
	JOIN transaction_types tt ON tt.id = t.transaction_type_id`

// TransactionRepository defines the data access contract for the transactions of the
// acting workspace.
type TransactionRepository interface {
	Insert(ctx context.Context, transaction *model.Transaction) (bool, error)
	LatestReading(ctx context.Context, customerNumber string) (*model.Transaction, error)
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
// transaction of that operation of the customer on that date. A held transaction takes
// the values of the new one when a value changed. Insert sets the identifier of a row
// that it stores for the first time.
func (r *Transaction) Insert(ctx context.Context, transaction *model.Transaction) (bool, error) {
	const query = `
		INSERT INTO transactions (
			customer_number, transaction_type_id, transaction_date, consumption,
			meter_value, amount, balance, bill_path, meter_photo_path
		)
		VALUES (
			:customer_number, :transaction_type_id, CAST(:transaction_date AS date), :consumption,
			:meter_value, :amount, :balance, :bill_path, :meter_photo_path
		)
		ON CONFLICT ON CONSTRAINT transactions_workspace_customer_operation_date_key DO UPDATE
		SET consumption      = EXCLUDED.consumption,
		    meter_value      = EXCLUDED.meter_value,
		    amount           = EXCLUDED.amount,
		    balance          = EXCLUDED.balance,
		    bill_path        = EXCLUDED.bill_path,
		    meter_photo_path = EXCLUDED.meter_photo_path,
		    collected_at     = now()
		WHERE (transactions.consumption, transactions.meter_value, transactions.amount,
		       transactions.balance, transactions.bill_path, transactions.meter_photo_path)
		      IS DISTINCT FROM
		      (EXCLUDED.consumption, EXCLUDED.meter_value, EXCLUDED.amount,
		       EXCLUDED.balance, EXCLUDED.bill_path, EXCLUDED.meter_photo_path)
		RETURNING id, (xmax = 0) AS inserted;`

	bound, values, err := sqlx.Named(query, map[string]any{
		paramCustomerNumber:   transaction.CustomerNumber,
		"transaction_type_id": transaction.TypeID,
		"transaction_date":    transaction.Date.Format(time.DateOnly),
		"consumption":         transaction.Consumption,
		"meter_value":         transaction.MeterValue,
		"amount":              transaction.Amount,
		"balance":             transaction.Balance,
		"bill_path":           transaction.BillPath,
		"meter_photo_path":    transaction.MeterPhotoPath,
	})
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

// LatestReading answers the latest reading of the customer, or nil when the customer
// has none.
func (r *Transaction) LatestReading(
	ctx context.Context,
	customerNumber string,
) (*model.Transaction, error) {
	const query = selectTransactions + `
		WHERE t.customer_number = :customer_number
		  AND t.transaction_type_id = :reading_operation
		ORDER BY t.transaction_date DESC, t.id DESC
		LIMIT 1;`

	transactions, err := r.selectNamed(ctx, query, map[string]any{
		paramCustomerNumber: customerNumber,
		"reading_operation": model.OperationReading,
	})
	if err != nil {
		return nil, fmt.Errorf("selecting the latest reading: %w", err)
	}

	if len(transactions) == 0 {
		return nil, nil //nolint:nilnil // no reading is no failure
	}

	return &transactions[0], nil
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
	condition, sorting := transactionKeyset(seek)
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

	boundary := any(nil)
	if seek.Boundary != "" {
		boundary = seek.Boundary
	}

	transactions, err := r.selectNamed(ctx, query, map[string]any{
		paramCustomerNumber: customerNumber,
		"from":              bound,
		"boundary":          boundary,
		"limit":             seek.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("listing the transactions: %w", err)
	}

	if seek.Direction == page.DirectionBackward {
		slices.Reverse(transactions)
	}

	return transactions, nil
}

// transactionKeyset answers the condition and the order of a keyset read beside the
// row that the parameter :boundary names. The collection runs the latest date first, so
// a forward read goes toward the earlier transactions.
func transactionKeyset(seek page.Seek) (string, string) {
	operator, direction := "<", "DESC"
	if seek.Direction == page.DirectionBackward {
		operator, direction = ">", "ASC"
	}

	sorting := "t.transaction_date " + direction + ", t.id " + direction

	if seek.Boundary == "" {
		return "TRUE", sorting
	}

	condition := "(t.transaction_date, t.id) " + operator + ` (
		SELECT b.transaction_date, b.id FROM transactions b
		WHERE b.id = CAST(:boundary AS uuid))`

	return condition, sorting
}

func (r *Transaction) selectNamed(
	ctx context.Context,
	query string,
	arguments map[string]any,
) ([]model.Transaction, error) {
	bound, values, err := sqlx.Named(query, arguments)
	if err != nil {
		return nil, fmt.Errorf("binding the arguments: %w", err)
	}

	var transactions []model.Transaction
	if err = r.tx.SelectContext(ctx, &transactions, r.tx.Rebind(bound), values...); err != nil {
		return nil, fmt.Errorf("selecting: %w", err)
	}

	return transactions, nil
}
