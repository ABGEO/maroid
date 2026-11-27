package service

import (
	"context"
	"fmt"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pensions/model"
)

// SnapshotStore writes everything that one collection read.
type SnapshotStore interface {
	Save(ctx context.Context, snapshot model.Snapshot) error
}

// Collector collects one month of the pension account of the acting user.
type Collector struct {
	account *account
	client  APIClientService
	store   SnapshotStore
}

// NewCollector creates a new Collector.
func NewCollector(
	client APIClientService,
	settings pluginapi.PluginSettings,
	secrets pluginapi.PluginSecrets,
	store SnapshotStore,
) *Collector {
	return &Collector{
		account: &account{client: client, settings: settings, secrets: secrets},
		client:  client,
		store:   store,
	}
}

// Collect reads the month, then writes it in one transaction, so a failed call writes
// nothing. It answers errs.ErrNotConnected with no session, and calls no service for it.
func (c *Collector) Collect(ctx context.Context, month model.Month) (model.Collection, error) {
	session, err := c.account.session(ctx)
	if err != nil {
		return model.Collection{}, err
	}

	snapshot, err := c.read(ctx, session, month)
	if err != nil {
		return model.Collection{}, err
	}

	if err = c.store.Save(ctx, snapshot); err != nil {
		return model.Collection{}, fmt.Errorf("storing the collection of %s: %w", month, err)
	}

	return model.Collection{
		Contributions: len(snapshot.Contributions),
		AssetValues:   len(snapshot.AssetValues),
		UnitPrices:    len(snapshot.UnitPrices),
		Shares:        len(snapshot.Shares),
	}, nil
}

// read makes every call of the month. Each call takes the session that the call before
// it answered.
func (c *Collector) read(
	ctx context.Context,
	session model.Session,
	month model.Month,
) (model.Snapshot, error) {
	participant, session, err := callWith(ctx, c.account, session,
		func(session model.Session) (model.Participant, error) {
			return c.client.Participant(ctx, session)
		})
	if err != nil {
		return model.Snapshot{}, err
	}

	shares, session, err := callWith(ctx, c.account, session,
		func(session model.Session) ([]model.PortfolioShare, error) {
			return c.client.PortfolioShares(ctx, session)
		})
	if err != nil {
		return model.Snapshot{}, err
	}

	values, session, err := callWith(ctx, c.account, session,
		func(session model.Session) ([]model.AssetValue, error) {
			return c.client.AssetValues(ctx, session, month)
		})
	if err != nil {
		return model.Snapshot{}, err
	}

	contributions, session, err := callWith(ctx, c.account, session,
		func(session model.Session) ([]model.Contribution, error) {
			return c.client.Contributions(ctx, session, month)
		})
	if err != nil {
		return model.Snapshot{}, err
	}

	prices, _, err := callWith(ctx, c.account, session,
		func(session model.Session) ([]model.PortfolioUnitPrice, error) {
			return c.client.PortfolioUnitPrices(ctx, session, portfolioIDsOf(shares))
		})
	if err != nil {
		return model.Snapshot{}, err
	}

	summary := participant.Summary
	for _, share := range shares {
		summary.Units += share.Units
	}

	return model.Snapshot{
		Contributions: contributions,
		AssetValues:   values,
		UnitPrices:    prices,
		Summary:       summary,
		Shares:        shares,
	}, nil
}

func portfolioIDsOf(shares []model.PortfolioShare) []int64 {
	ids := make([]int64, 0, len(shares))
	for _, share := range shares {
		ids = append(ids, share.PortfolioID)
	}

	return ids
}
