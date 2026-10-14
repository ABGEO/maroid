package service

import (
	"context"
	"fmt"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/telmico/model"
)

// DocumentStore finds a stored bill of the assigned customer of the acting workspace.
type DocumentStore interface {
	Bill(ctx context.Context, id string) (model.Bill, error)
}

// Documents downloads the document of a stored bill from the portal. Maroid keeps no copy.
type Documents struct {
	account *account
	client  APIClientService
	store   DocumentStore
}

// NewDocuments creates a new Documents.
func NewDocuments(
	client APIClientService,
	settings *pluginapi.PluginSettings,
	secrets *pluginapi.PluginSecrets,
	store DocumentStore,
) *Documents {
	return &Documents{
		account: &account{client: client, settings: settings, secrets: secrets},
		client:  client,
		store:   store,
	}
}

// Open answers the document of the bill. It answers errs.ErrNotAssigned with no
// assignment and errs.ErrBillUnknown for a bill of no assigned customer, both with no
// call to the portal, and errs.ErrNotConnected with no session that works.
func (d *Documents) Open(ctx context.Context, id string) (model.Document, error) {
	bill, err := d.store.Bill(ctx, id)
	if err != nil {
		return model.Document{}, fmt.Errorf("reading the bill: %w", err)
	}

	session, err := d.account.session(ctx)
	if err != nil {
		return model.Document{}, err
	}

	content, _, err := callWith(
		ctx,
		d.account,
		session,
		func(session model.Session) ([]byte, error) {
			return d.client.ReceiptDocument(ctx, session, bill.ServiceBillID)
		},
	)
	if err != nil {
		return model.Document{}, notConnectedOnRejection(
			fmt.Errorf("downloading the bill: %w", err),
		)
	}

	return model.Document{Filename: bill.BillNumber + ".pdf", Content: content}, nil
}
