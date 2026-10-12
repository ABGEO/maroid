package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/gwp/dto"
	"github.com/abgeo/maroid/plugins/gwp/errs"
	"github.com/abgeo/maroid/plugins/gwp/model"
)

// ReadingStore holds the assignment and the readings of the acting workspace.
type ReadingStore interface {
	AssignedCustomer(ctx context.Context) (string, error)
	InsertReadings(ctx context.Context, readings []model.Reading) error
}

// Collector collects the readings of the assigned customer of the acting workspace.
type Collector struct {
	account *account
	client  APIClientService
	store   ReadingStore
}

// NewCollector creates a new Collector.
func NewCollector(
	client APIClientService,
	settings *pluginapi.PluginSettings,
	secrets *pluginapi.PluginSecrets,
	store ReadingStore,
) *Collector {
	return &Collector{
		account: &account{client: client, settings: settings, secrets: secrets},
		client:  client,
		store:   store,
	}
}

// Collect reads the readings of the assigned customer of the acting workspace and
// inserts each new one. It answers errs.ErrNotAssigned with no assigned customer and
// errs.ErrNotConnected with no connection. The job treats both as a skip.
func (c *Collector) Collect(ctx context.Context) error {
	customerNumber, err := c.store.AssignedCustomer(ctx)
	if err != nil {
		return fmt.Errorf("reading the assigned customer: %w", err)
	}

	token, err := c.account.token(ctx)
	if err != nil {
		return err
	}

	responses, err := c.readings(ctx, token)
	if err != nil {
		return err
	}

	readings, err := readingsOf(responses, customerNumber)
	if err != nil {
		return err
	}

	if err = c.store.InsertReadings(ctx, readings); err != nil {
		return fmt.Errorf("storing the readings: %w", err)
	}

	return nil
}

// readings reads the readings with the token. A rejected token gives way to one new
// sign in, because the stored credentials need no person to send a code.
func (c *Collector) readings(ctx context.Context, token string) ([]dto.ReadingResponse, error) {
	responses, err := c.client.GetReadings(ctx, token)

	switch {
	case err == nil:
		return responses, nil
	case !errors.Is(err, errs.ErrNotConnected):
		return nil, fmt.Errorf("collecting with the stored token: %w", err)
	}

	token, err = c.account.signIn(ctx)
	if err != nil {
		return nil, fmt.Errorf("replacing a rejected token: %w", err)
	}

	responses, err = c.client.GetReadings(ctx, token)

	switch {
	case errors.Is(err, errs.ErrNotConnected):
		return nil, fmt.Errorf("%w: the water service rejected a new token", errs.ErrRequestFailed)
	case err != nil:
		return nil, fmt.Errorf("collecting with a new token: %w", err)
	}

	return responses, nil
}

func readingsOf(responses []dto.ReadingResponse, customerNumber string) ([]model.Reading, error) {
	for _, response := range responses {
		if response.CustomerNumber != customerNumber {
			continue
		}

		readings := make([]model.Reading, 0, len(response.Items))

		for i := range response.Items {
			reading, err := response.Items[i].MapToModel(customerNumber)
			if err != nil {
				return nil, fmt.Errorf("mapping a reading of the customer: %w", err)
			}

			readings = append(readings, reading)
		}

		return readings, nil
	}

	return nil, errs.ErrCustomerAbsent
}
