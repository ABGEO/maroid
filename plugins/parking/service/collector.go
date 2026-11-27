package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/parking/dto"
	"github.com/abgeo/maroid/plugins/parking/errs"
	"github.com/abgeo/maroid/plugins/parking/model"
)

// CollectionStore holds the history that a collection reads.
type CollectionStore interface {
	KnownSessions(ctx context.Context, ids []int64) (map[int64]bool, error)
	KnownTransactions(ctx context.Context, ids []int64) (map[int64]bool, error)
	Save(
		ctx context.Context,
		sessions []model.Session,
		transactions []model.Transaction,
	) (model.Collection, error)
}

// Collector reads the sessions and the transactions of the acting user from the parking
// service, and stores them.
type Collector struct {
	client HistoryClient
	store  CollectionStore
	logger *slog.Logger
}

// NewCollector creates a new Collector.
func NewCollector(client HistoryClient, store CollectionStore, logger *slog.Logger) *Collector {
	return &Collector{
		client: client,
		store:  store,
		logger: logger.With(
			slog.String("component", "collector"),
			slog.String("collector", "history"),
		),
	}
}

// Collect reads both lists as far as reach gives, then writes them in one database
// transaction. It answers errs.ErrNotConnected when the user holds no connection, or the
// parking service rejects it.
//
// Nothing is written until both lists are read.
func (c *Collector) Collect(ctx context.Context, reach model.Reach) (model.Collection, error) {
	sessions, err := readList(ctx, c, reach, historyList[dto.HistorySession, model.Session]{
		name:  "sessions",
		read:  c.client.ParkingHistory,
		known: c.store.KnownSessions,
		id:    func(row dto.HistorySession) int64 { return row.ID },
		keep:  dto.HistorySession.Finished,
		model: dto.HistorySession.ToModel,
	})
	if err != nil {
		return model.Collection{}, err
	}

	transactions, err := readList(ctx, c, reach, historyList[dto.PaymentEntry, model.Transaction]{
		name:  "transactions",
		read:  c.client.PaymentHistory,
		known: c.store.KnownTransactions,
		id:    func(row dto.PaymentEntry) int64 { return row.ID },
		keep:  func(dto.PaymentEntry) bool { return true },
		model: dto.PaymentEntry.ToModel,
	})
	if err != nil {
		return model.Collection{}, err
	}

	if len(sessions) == 0 && len(transactions) == 0 {
		return model.Collection{}, nil
	}

	collection, err := c.store.Save(ctx, sessions, transactions)
	if err != nil {
		return model.Collection{}, fmt.Errorf("storing the collection: %w", err)
	}

	return collection, nil
}

// historyList tells readList how to read one list of the parking service.
type historyList[R, M any] struct {
	name  string
	read  func(ctx context.Context, page int) (dto.Page[R], error)
	known func(ctx context.Context, ids []int64) (map[int64]bool, error)
	id    func(row R) int64
	keep  func(row R) bool
	model func(row R) (M, error)
}

// readList reads the pages of one list from page 0, and answers the rows to store.
//
// The recent reach stops after a page with no new row, because the
// ledger identifiers do not follow the dates. The full reach keeps every row, and the
// upsert counts the new ones.
func readList[R, M any](
	ctx context.Context,
	collector *Collector,
	reach model.Reach,
	list historyList[R, M],
) ([]M, error) {
	var (
		rows       []M
		newOnLast  bool
		pageNumber int
	)

	for ; ; pageNumber++ {
		if pageNumber >= reach.Pages {
			return rows, collector.atLimit(ctx, reach, list.name, newOnLast)
		}

		page, err := list.read(ctx, pageNumber)
		if err != nil {
			return nil, fmt.Errorf("reading page %d of the %s: %w", pageNumber, list.name, err)
		}

		if len(page.Items) == 0 {
			return rows, nil
		}

		fresh, err := freshRows(ctx, reach, list, page.Items)
		if err != nil {
			return nil, fmt.Errorf("reading page %d of the %s: %w", pageNumber, list.name, err)
		}

		rows = append(rows, fresh...)
		newOnLast = len(fresh) > 0

		if (reach.StopAtKnown && !newOnLast) || page.Meta.Last() {
			return rows, nil
		}
	}
}

// freshRows answers the rows of one page to store. The recent reach drops each row that
// the user holds. A row that keep refuses counts as held.
//
// An active session is not stored.
func freshRows[R, M any](
	ctx context.Context,
	reach model.Reach,
	list historyList[R, M],
	items []R,
) ([]M, error) {
	kept := make([]R, 0, len(items))
	ids := make([]int64, 0, len(items))

	for _, item := range items {
		if list.keep(item) {
			kept = append(kept, item)
			ids = append(ids, list.id(item))
		}
	}

	held := map[int64]bool{}

	if reach.StopAtKnown && len(ids) > 0 {
		var err error
		if held, err = list.known(ctx, ids); err != nil {
			return nil, err
		}
	}

	fresh := make([]M, 0, len(kept))

	for _, item := range kept {
		if held[list.id(item)] {
			continue
		}

		row, err := list.model(item)
		if err != nil {
			return nil, err
		}

		fresh = append(fresh, row)
	}

	return fresh, nil
}

// atLimit ends a list at the most pages of the reach.
//
// A scheduled collection that stops with new rows on its last page
// leaves a gap that only the command line collection fills.
func (c *Collector) atLimit(
	ctx context.Context,
	reach model.Reach,
	list string,
	newOnLast bool,
) error {
	if reach.FailAtLimit {
		return fmt.Errorf("%w: the %s pass %d pages", errs.ErrPageLimit, list, reach.Pages)
	}

	if newOnLast {
		c.logger.WarnContext(
			ctx,
			"stopped at the page limit with new rows left: run backfill",
			slog.String("user_id", pluginapi.ActingUserFromContext(ctx)),
			slog.String("list", list),
			slog.Int("pages", reach.Pages),
		)
	}

	return nil
}
