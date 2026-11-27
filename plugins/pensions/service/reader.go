package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pensions/errs"
	"github.com/abgeo/maroid/plugins/pensions/model"
)

// SummaryStore reads the stored summary of the acting user.
type SummaryStore interface {
	Summary(ctx context.Context) (*model.AccountSummary, error)
}

// Reader reads what the pages and the widget show first. It calls no external service.
type Reader struct {
	account *account
	store   SummaryStore
}

// NewReader creates a new Reader.
func NewReader(secrets pluginapi.PluginSecrets, store SummaryStore) *Reader {
	return &Reader{account: &account{secrets: secrets}, store: store}
}

// Summary answers the stored summary of the acting user, and whether Maroid holds a
// session for them. A stored session that does not read is no connection.
func (r *Reader) Summary(ctx context.Context) (model.SummaryView, error) {
	summary, err := r.store.Summary(ctx)
	if err != nil {
		return model.SummaryView{}, fmt.Errorf("reading the summary: %w", err)
	}

	_, err = r.account.session(ctx)

	switch {
	case err == nil:
		return model.SummaryView{Connected: true, Summary: summary}, nil
	case errors.Is(err, errs.ErrNotConnected):
		return model.SummaryView{Summary: summary}, nil
	default:
		return model.SummaryView{}, err
	}
}
