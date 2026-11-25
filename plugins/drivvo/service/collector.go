package service

import (
	"context"
	"fmt"

	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// AccountReader reads the car log account with a token.
type AccountReader interface {
	Settings(ctx context.Context, token string) (model.CarLogSettings, error)
	ReferenceItems(ctx context.Context, token string) ([]model.ReferenceItem, error)
	Vehicles(ctx context.Context, token string) ([]model.Vehicle, error)
	VehicleEntries(ctx context.Context, token string, vehicle int32) ([]model.LogEntry, error)
	Reminders(ctx context.Context, token string, vehicle int32) ([]model.Reminder, error)
}

// SnapshotStore replaces the car log copy of the acting workspace.
type SnapshotStore interface {
	Replace(ctx context.Context, snapshot model.Snapshot) error
}

// Collector reads the whole car log account of the acting workspace, and replaces its car
// log copy.
type Collector struct {
	account *Account
	reader  AccountReader
	store   SnapshotStore
}

// NewCollector creates a new Collector.
func NewCollector(account *Account, reader AccountReader, store SnapshotStore) *Collector {
	return &Collector{account: account, reader: reader, store: store}
}

// Collect reads the account into a snapshot, then stores it. A failed read stores nothing.
// It answers errs.ErrNotConnected, and calls no service, when the workspace holds no session.
func (c *Collector) Collect(ctx context.Context) (model.Snapshot, error) {
	snapshot, err := withToken(ctx, c.account, func(token string) (model.Snapshot, error) {
		return c.read(ctx, token)
	})
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("reading the car log account: %w", err)
	}

	if err = c.store.Replace(ctx, snapshot); err != nil {
		return model.Snapshot{}, fmt.Errorf("storing the car log account: %w", err)
	}

	return snapshot, nil
}

// read reads the account one call at a time.
func (c *Collector) read(ctx context.Context, token string) (model.Snapshot, error) {
	var (
		snapshot model.Snapshot
		err      error
	)

	if snapshot.Settings, err = c.reader.Settings(ctx, token); err != nil {
		return model.Snapshot{}, fmt.Errorf("reading the settings: %w", err)
	}

	if snapshot.ReferenceItems, err = c.reader.ReferenceItems(ctx, token); err != nil {
		return model.Snapshot{}, fmt.Errorf("reading the reference items: %w", err)
	}

	if snapshot.Vehicles, err = c.reader.Vehicles(ctx, token); err != nil {
		return model.Snapshot{}, fmt.Errorf("reading the vehicles: %w", err)
	}

	for _, vehicle := range snapshot.Vehicles {
		entries, err := c.reader.VehicleEntries(ctx, token, vehicle.SourceID)
		if err != nil {
			return model.Snapshot{}, fmt.Errorf(
				"reading the entries of vehicle %d: %w",
				vehicle.SourceID,
				err,
			)
		}

		reminders, err := c.reader.Reminders(ctx, token, vehicle.SourceID)
		if err != nil {
			return model.Snapshot{}, fmt.Errorf(
				"reading the reminders of vehicle %d: %w",
				vehicle.SourceID,
				err,
			)
		}

		snapshot.LogEntries = append(snapshot.LogEntries, entries...)
		snapshot.Reminders = append(snapshot.Reminders, reminders...)
	}

	return snapshot, nil
}
