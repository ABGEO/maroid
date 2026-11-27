package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/pensions/model"
)

// AccountSummaryRepository defines the data access contract for the summary of the
// acting user.
type AccountSummaryRepository interface {
	Put(ctx context.Context, summary *model.AccountSummary) error
	Get(ctx context.Context) (*model.AccountSummary, error)
}

// AccountSummary reads and writes the summary of the acting user within one transaction
// of the database.
type AccountSummary struct {
	tx *sqlx.Tx
}

var _ AccountSummaryRepository = (*AccountSummary)(nil)

// NewAccountSummary creates a new AccountSummary over the transaction of the database.
func NewAccountSummary(tx *sqlx.Tx) *AccountSummary {
	return &AccountSummary{tx: tx}
}

// Put replaces the summary of the acting user. The collection time moves on each write,
// because it dates the summary that the person reads.
func (r *AccountSummary) Put(ctx context.Context, summary *model.AccountSummary) error {
	const query = `
		INSERT INTO account_summaries (
			asset_value, units, gain, withheld_amount, person_contributions,
			employer_contributions, state_contributions, total_contributions
		)
		VALUES (
			:asset_value, :units, :gain, :withheld_amount, :person_contributions,
			:employer_contributions, :state_contributions, :total_contributions
		)
		ON CONFLICT ON CONSTRAINT account_summaries_user_key DO UPDATE
		SET asset_value            = EXCLUDED.asset_value,
		    units                  = EXCLUDED.units,
		    gain                   = EXCLUDED.gain,
		    withheld_amount        = EXCLUDED.withheld_amount,
		    person_contributions   = EXCLUDED.person_contributions,
		    employer_contributions = EXCLUDED.employer_contributions,
		    state_contributions    = EXCLUDED.state_contributions,
		    total_contributions    = EXCLUDED.total_contributions,
		    collected_at           = now();`

	if _, err := r.tx.NamedExecContext(ctx, query, summary); err != nil {
		return fmt.Errorf("putting the summary: %w", err)
	}

	return nil
}

// Get answers the summary of the acting user, or nil when no collection stored one.
func (r *AccountSummary) Get(ctx context.Context) (*model.AccountSummary, error) {
	const query = `
		SELECT asset_value, units, gain, withheld_amount, person_contributions,
		       employer_contributions, state_contributions, total_contributions, collected_at
		FROM account_summaries;`

	summary := new(model.AccountSummary)

	err := r.tx.GetContext(ctx, summary, query)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil //nolint:nilnil // no summary is no failure
	}

	if err != nil {
		return nil, fmt.Errorf("selecting the summary: %w", err)
	}

	return summary, nil
}
