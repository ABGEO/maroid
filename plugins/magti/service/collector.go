package service

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/abgeo/maroid/plugins/magti/dto"
	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/model"
)

// CollectionStore holds the assignment, the traffic, and the money entries of the acting
// workspace.
type CollectionStore interface {
	Assignment(ctx context.Context) (model.Assignment, error)
	SaveCollection(
		ctx context.Context,
		days []model.TrafficDay,
		entries []model.MoneyEntry,
	) (model.Collection, error)
}

// Collector collects the traffic and the money entries of the assigned mobile subscriber
// of the acting workspace.
type Collector struct {
	account *Account
	client  APIClientService
	store   CollectionStore
}

// NewCollector creates a new Collector.
func NewCollector(account *Account, client APIClientService, store CollectionStore) *Collector {
	return &Collector{account: account, client: client, store: store}
}

// statisticsAnswer is what the portal answers for one period of one subscriber.
type statisticsAnswer struct {
	usage  map[dto.UsageSeries][]dto.UsageRow
	income []dto.MoneyRow
	charge []dto.MoneyRow
}

// Collect reads the statistics of the assigned subscriber in the period, and stores them
// in one transaction. It answers errs.ErrNotAssigned with no assignment, errs.ErrHomeLine
// for a home line, and errs.ErrNotConnected with no session, and calls no portal for any
// of them.
func (c *Collector) Collect(ctx context.Context, period model.Period) (model.Collection, error) {
	if period.From.After(period.To) {
		return model.Collection{}, errs.ErrReversedPeriod
	}

	assignment, err := c.store.Assignment(ctx)
	if err != nil {
		return model.Collection{}, fmt.Errorf("reading the assigned subscriber: %w", err)
	}

	if assignment.IsHomeLine() {
		return model.Collection{}, errs.ErrHomeLine
	}

	number := assignment.SubscriberNumber

	answer, err := callLocked(
		ctx,
		c.account,
		func(session model.Session) (statisticsAnswer, error) {
			return c.read(ctx, session, number, period)
		},
	)
	if err != nil {
		return model.Collection{}, err
	}

	days, entries, err := mapStatistics(answer, number)
	if err != nil {
		return model.Collection{}, err
	}

	collection, err := c.store.SaveCollection(ctx, days, entries)
	if err != nil {
		return model.Collection{}, fmt.Errorf("storing the statistics: %w", err)
	}

	return collection, nil
}

// read checks that the account still holds the subscriber, opens the statistics, and
// reads the five series.
func (c *Collector) read(
	ctx context.Context,
	session model.Session,
	number string,
	period model.Period,
) (statisticsAnswer, error) {
	subscribers, err := c.client.Subscribers(ctx, session)
	if err != nil {
		return statisticsAnswer{}, fmt.Errorf("reading the subscribers: %w", err)
	}

	if !slices.ContainsFunc(
		subscribers,
		func(s model.Subscriber) bool { return s.Number == number },
	) {
		return statisticsAnswer{}, errs.ErrSubscriberAbsent
	}

	token, err := c.client.OpenStatistics(ctx, session, number)
	if err != nil {
		return statisticsAnswer{}, fmt.Errorf("opening the statistics: %w", err)
	}

	query := StatisticsQuery{Number: number, Token: token, Period: period}
	answer := statisticsAnswer{usage: map[dto.UsageSeries][]dto.UsageRow{}}

	for _, series := range []dto.UsageSeries{dto.SeriesData, dto.SeriesMinutes, dto.SeriesSMS} {
		answer.usage[series], err = c.client.Usage(ctx, session, query, series)
		if err != nil {
			return statisticsAnswer{}, fmt.Errorf("reading the statistics: %w", err)
		}
	}

	if answer.charge, err = c.client.Money(ctx, session, query, ChargeExpense); err != nil {
		return statisticsAnswer{}, fmt.Errorf("reading the statistics: %w", err)
	}

	if answer.income, err = c.client.Money(ctx, session, query, ChargeIncome); err != nil {
		return statisticsAnswer{}, fmt.Errorf("reading the statistics: %w", err)
	}

	return answer, nil
}

func mapStatistics(
	answer statisticsAnswer,
	number string,
) ([]model.TrafficDay, []model.MoneyEntry, error) {
	merged := map[model.Day]*model.TrafficDay{}

	for _, series := range []dto.UsageSeries{dto.SeriesData, dto.SeriesMinutes, dto.SeriesSMS} {
		dto.MergeUsage(merged, series, answer.usage[series])
	}

	days := make([]model.TrafficDay, 0, len(merged))

	for _, day := range slices.SortedFunc(maps.Keys(merged), func(a, b model.Day) int {
		return a.Sub(b)
	}) {
		traffic := *merged[day]
		traffic.SubscriberNumber = number
		days = append(days, traffic)
	}

	income, err := dto.MoneyEntries(answer.income, model.DirectionIn)
	if err != nil {
		return nil, nil, fmt.Errorf("reading the top ups: %w", err)
	}

	charges, err := dto.MoneyEntries(answer.charge, model.DirectionOut)
	if err != nil {
		return nil, nil, fmt.Errorf("reading the charges: %w", err)
	}

	entries := slices.Concat(income, charges)
	for i := range entries {
		entries[i].SubscriberNumber = number
	}

	return days, entries, nil
}
