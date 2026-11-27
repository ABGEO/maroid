package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/parking/model"
)

// Transaction reads and writes the transactions of the acting user within one database
// transaction.
type Transaction struct {
	tx *sqlx.Tx
}

// NewTransaction creates a new Transaction over the database transaction.
func NewTransaction(tx *sqlx.Tx) *Transaction {
	return &Transaction{tx: tx}
}

// Known answers the external identifiers among ids that the acting user holds.
func (r *Transaction) Known(ctx context.Context, ids []int64) (map[int64]bool, error) {
	known, err := knownOf(ctx, r.tx, "transactions", ids)
	if err != nil {
		return nil, fmt.Errorf("reading the held transactions: %w", err)
	}

	return known, nil
}

// Upsert stores the transactions in one statement and answers the count of the new rows.
//
// PARKHIST-FR-011: One transaction for one identifier of the parking service.
func (r *Transaction) Upsert(ctx context.Context, transactions []model.Transaction) (int, error) {
	if len(transactions) == 0 {
		return 0, nil
	}

	const query = `
		INSERT INTO transactions
		    (external_id, kind, service_kind, amount, balance_after, occurred_at,
		     session_external_id, description)
		SELECT external_id, kind, service_kind, amount, balance_after, occurred_at,
		       session_external_id, description
		FROM unnest(
		    CAST(:external_ids AS bigint[]), CAST(:kinds AS text[]),
		    CAST(:service_kinds AS text[]), CAST(:amounts AS numeric[]),
		    CAST(:balances_after AS numeric[]), CAST(:occurred_ats AS timestamptz[]),
		    CAST(:session_external_ids AS bigint[]), CAST(:descriptions AS text[]))
		     AS answered (external_id, kind, service_kind, amount, balance_after, occurred_at,
		                  session_external_id, description)
		ON CONFLICT ON CONSTRAINT transactions_user_external_id_key DO UPDATE
		SET kind                = EXCLUDED.kind,
		    service_kind        = EXCLUDED.service_kind,
		    amount              = EXCLUDED.amount,
		    balance_after       = EXCLUDED.balance_after,
		    occurred_at         = EXCLUDED.occurred_at,
		    session_external_id = EXCLUDED.session_external_id,
		    description         = EXCLUDED.description,
		    collected_at        = now()
		WHERE (transactions.kind, transactions.service_kind, transactions.amount,
		       transactions.balance_after, transactions.occurred_at,
		       transactions.session_external_id, transactions.description)
		      IS DISTINCT FROM
		      (EXCLUDED.kind, EXCLUDED.service_kind, EXCLUDED.amount, EXCLUDED.balance_after,
		       EXCLUDED.occurred_at, EXCLUDED.session_external_id, EXCLUDED.description)
		RETURNING (xmax = 0) AS inserted;`

	flags, err := selectNamed[bool](ctx, r.tx, query, transactionArguments(transactions))
	if err != nil {
		return 0, fmt.Errorf("upserting the transactions: %w", err)
	}

	return insertedOf(flags), nil
}

func transactionArguments(transactions []model.Transaction) map[string]any {
	externalIDs := make([]int64, len(transactions))
	kinds := make([]string, len(transactions))
	serviceKinds := make([]string, len(transactions))
	amounts := make([]float64, len(transactions))
	balancesAfter := make([]float64, len(transactions))
	occurredAts := make([]time.Time, len(transactions))
	sessionExternalIDs := make([]*int64, len(transactions))
	descriptions := make([]string, len(transactions))

	for i, transaction := range transactions {
		externalIDs[i] = transaction.ExternalID
		kinds[i] = string(transaction.Kind)
		serviceKinds[i] = transaction.ServiceKind
		amounts[i] = transaction.Amount
		balancesAfter[i] = transaction.BalanceAfter
		occurredAts[i] = transaction.OccurredAt
		sessionExternalIDs[i] = transaction.SessionExternalID
		descriptions[i] = transaction.Description
	}

	return map[string]any{
		"external_ids":         externalIDs,
		"kinds":                kinds,
		"service_kinds":        serviceKinds,
		"amounts":              amounts,
		"balances_after":       balancesAfter,
		"occurred_ats":         occurredAts,
		"session_external_ids": sessionExternalIDs,
		"descriptions":         descriptions,
	}
}
