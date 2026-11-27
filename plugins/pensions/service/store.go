package service

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pensions/model"
	"github.com/abgeo/maroid/plugins/pensions/repository"
)

// Store runs each operation on the data of the acting user in a transaction of its own,
// and creates each repository inside it.
type Store struct {
	db pluginapi.PluginDB
}

var _ SnapshotStore = (*Store)(nil)

// NewStore creates a new Store over the database of the plugin.
func NewStore(db pluginapi.PluginDB) *Store {
	return &Store{db: db}
}

// Save writes the whole snapshot in one transaction, so a failure leaves the data as it
// was. The employers go first, because a contribution references its employer.
func (s *Store) Save(ctx context.Context, snapshot model.Snapshot) error {
	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		if err := saveContributions(ctx, tx, snapshot.Contributions); err != nil {
			return err
		}

		values := repository.NewAssetValue(tx)
		for i := range snapshot.AssetValues {
			if err := values.Upsert(ctx, &snapshot.AssetValues[i]); err != nil {
				return fmt.Errorf("writing an asset value: %w", err)
			}
		}

		prices := repository.NewPortfolioUnitPrice(tx)
		if _, err := prices.Upsert(ctx, snapshot.UnitPrices); err != nil {
			return fmt.Errorf("writing the unit prices: %w", err)
		}

		if err := repository.NewAccountSummary(tx).Put(ctx, &snapshot.Summary); err != nil {
			return fmt.Errorf("writing the summary: %w", err)
		}

		if err := repository.NewPortfolioShare(tx).Replace(ctx, snapshot.Shares); err != nil {
			return fmt.Errorf("writing the portfolio shares: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("saving the snapshot: %w", err)
	}

	return nil
}

func saveContributions(ctx context.Context, tx *sqlx.Tx, contributions []model.Contribution) error {
	employers := repository.NewEmployer(tx)
	seen := map[string]bool{}

	for _, contribution := range contributions {
		if contribution.Employer == nil || seen[contribution.Employer.Code] {
			continue
		}

		seen[contribution.Employer.Code] = true

		if err := employers.Upsert(ctx, contribution.Employer); err != nil {
			return fmt.Errorf("writing an employer: %w", err)
		}
	}

	held := repository.NewContribution(tx)
	for i := range contributions {
		if err := held.Upsert(ctx, &contributions[i]); err != nil {
			return fmt.Errorf("writing a contribution: %w", err)
		}
	}

	return nil
}
