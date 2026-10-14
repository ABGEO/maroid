package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/model"
)

const selectBills = `
	SELECT t.id, t.customer_number, t.bill_number, t.service_bill_id, t.month, t.issued_at,
	       t.meter_serial
	FROM bills t`

// BillRepository defines the data access contract for the bills of the acting workspace.
type BillRepository interface {
	Insert(ctx context.Context, bill *model.Bill) error
	Get(ctx context.Context, id string, customerNumber string) (model.Bill, error)
	ByMonth(ctx context.Context, customerNumber string, month model.Month) (*model.Bill, error)
	List(ctx context.Context, seek page.Seek, customerNumber string) ([]model.Bill, error)
}

// Bill reads and writes the bills of the acting workspace within one transaction of
// the database.
type Bill struct {
	tx *sqlx.Tx
}

var _ BillRepository = (*Bill)(nil)

// NewBill creates a new Bill over the transaction of the database.
func NewBill(tx *sqlx.Tx) *Bill {
	return &Bill{tx: tx}
}

// Insert stores the bill. A held bill of the same number takes the values of the new one.
func (r *Bill) Insert(ctx context.Context, bill *model.Bill) error {
	const query = `
		INSERT INTO bills (
			customer_number, bill_number, service_bill_id, month, issued_at, meter_serial
		)
		VALUES (
			:customer_number, :bill_number, :service_bill_id, CAST(:month AS date),
			:issued_at, :meter_serial
		)
		ON CONFLICT ON CONSTRAINT bills_workspace_bill_number_key DO UPDATE
		SET service_bill_id = EXCLUDED.service_bill_id,
		    issued_at       = EXCLUDED.issued_at,
		    meter_serial    = EXCLUDED.meter_serial,
		    collected_at    = now();`

	_, err := r.tx.NamedExecContext(ctx, query, map[string]any{
		paramCustomerNumber: bill.CustomerNumber,
		"bill_number":       bill.BillNumber,
		"service_bill_id":   bill.ServiceBillID,
		"month":             bill.Month.Format(time.DateOnly),
		"issued_at":         bill.IssuedAt,
		"meter_serial":      bill.MeterSerial,
	})
	if err != nil {
		return fmt.Errorf("upserting the bill %s: %w", bill.BillNumber, err)
	}

	return nil
}

// Get answers the bill of the customer under the identifier, or errs.ErrBillUnknown.
func (r *Bill) Get(ctx context.Context, id string, customerNumber string) (model.Bill, error) {
	const query = selectBills + `
		WHERE t.id = CAST(:id AS uuid) AND t.customer_number = :customer_number;`

	bills, err := selectNamed[model.Bill](ctx, r.tx, query, map[string]any{
		"id":                id,
		paramCustomerNumber: customerNumber,
	})
	if err != nil {
		return model.Bill{}, fmt.Errorf("selecting the bill: %w", err)
	}

	if len(bills) == 0 {
		return model.Bill{}, errs.ErrBillUnknown
	}

	return bills[0], nil
}

// ByMonth answers the bill of the customer for the month, or nil when it holds none.
func (r *Bill) ByMonth(
	ctx context.Context,
	customerNumber string,
	month model.Month,
) (*model.Bill, error) {
	const query = selectBills + `
		WHERE t.customer_number = :customer_number AND t.month = CAST(:month AS date)
		ORDER BY t.issued_at DESC
		LIMIT 1;`

	bills, err := selectNamed[model.Bill](ctx, r.tx, query, map[string]any{
		paramCustomerNumber: customerNumber,
		"month":             month.First().Format(time.DateOnly),
	})
	if err != nil {
		return nil, fmt.Errorf("selecting the bill of %s: %w", month, err)
	}

	if len(bills) == 0 {
		return nil, nil //nolint:nilnil // no bill is no failure
	}

	return &bills[0], nil
}

// List answers one keyset read of the bills of the customer, the latest month first,
// with the identifier as the tie.
func (r *Bill) List(
	ctx context.Context,
	seek page.Seek,
	customerNumber string,
) ([]model.Bill, error) {
	condition, sorting := keyset(seek, "bills", "month")
	query := selectBills + `
		WHERE t.customer_number = :customer_number
		  AND ` + condition + `
		ORDER BY ` + sorting + `
		LIMIT :limit;`

	bills, err := selectNamed[model.Bill](ctx, r.tx, query, map[string]any{
		paramCustomerNumber: customerNumber,
		"boundary":          boundaryOf(seek),
		"limit":             seek.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("listing the bills: %w", err)
	}

	return inOrder(seek, bills), nil
}
