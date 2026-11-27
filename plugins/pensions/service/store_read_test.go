package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/pensions/model"
	"github.com/abgeo/maroid/plugins/pensions/service"
)

const (
	employerOther = "405112233"
	employerThird = "211998877"
	pageLimit     = 20
)

func on(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// paidBy answers one contribution of the employer, or of no employer for an empty code.
func paidBy(code string, paidOn time.Time, amount float64) model.Contribution {
	contribution := contributionOf(uuid.New(), sourcePerson, amount)
	contribution.PaidOn = paidOn
	contribution.EmployerCode, contribution.Employer = nil, nil

	if code != "" {
		contribution.EmployerCode = &code
		contribution.Employer = &model.Employer{Code: code, Name: "შპს " + code}
	}

	return contribution
}

func saveContributions(
	ctx context.Context,
	t *testing.T,
	store *service.Store,
	rows []model.Contribution,
) {
	t.Helper()

	require.NoError(t, store.Save(ctx, model.Snapshot{Contributions: rows, Summary: summaryOf(1)}))
}

// everyPage follows next from the first page to the last, and answers the size of each
// page and every row in order.
func everyPage[T any](
	t *testing.T,
	readPage func(page.Request) (page.Window[T], error),
) ([]int, []T) {
	t.Helper()

	var (
		sizes []int
		read  []T
		asked = page.Request{Limit: pageLimit}
	)

	for {
		window, err := readPage(asked)
		require.NoError(t, err)

		sizes = append(sizes, len(window.Items))
		read = append(read, window.Items...)

		if window.Next == nil {
			return sizes, read
		}

		asked.Cursor = window.Next
	}
}

func codeOf(filter string) model.ContributionFilter {
	return model.ContributionFilter{EmployerCode: &filter}
}

// PENSION-SC-036: The sums of the months hold each month of the user that holds a
// contribution, and no month of another user.
func TestTheMonthsSumTheContributionsOfTheUser(t *testing.T) {
	t.Parallel()

	store, _, _ := storeUnderTest(t)
	ctxA, ctxB := actingAs(t, userA), actingAs(t, userB)

	saveContributions(ctxA, t, store, []model.Contribution{
		paidBy(employerE, on(2025, time.December, 20), 50),
		paidBy(employerE, on(2026, time.January, 10), 100),
		paidBy(employerE, on(2026, time.January, 25), 120.5),
		paidBy(employerE, on(2026, time.March, 12), 80),
	})
	saveContributions(
		ctxB,
		t,
		store,
		[]model.Contribution{paidBy(employerE, on(2026, time.February, 3), 999)},
	)

	from := on(2026, time.January, 1)
	sums, err := store.ContributionMonths(ctxA, &from)

	require.NoError(t, err)
	assert.Equal(t, []model.MonthSum{
		{Month: on(2026, time.January, 1), Amount: 220.5},
		{Month: on(2026, time.March, 1), Amount: 80},
	}, sums)
}

// PENSION-SC-037: The pages run the latest date first, hold each contribution once, and
// break a tie of dates by the identifier. A contribution with no employer holds none.
func TestThePagesHoldEachContributionOnce(t *testing.T) {
	t.Parallel()

	store, _, _ := storeUnderTest(t)
	ctxA := actingAs(t, userA)

	rows := make([]model.Contribution, 0, 45)
	for i := range 42 {
		rows = append(rows, paidBy(employerE, on(2026, time.October, 14).AddDate(0, 0, -7*i), 100))
	}

	rows = append(rows,
		paidBy(employerE, october15(), 1),
		paidBy(employerOther, october15(), 2),
		paidBy("", october15(), 3),
	)
	saveContributions(ctxA, t, store, rows)

	sizes, read := everyPage(t, func(asked page.Request) (page.Window[model.Contribution], error) {
		return store.ContributionsPage(ctxA, asked, model.ContributionFilter{})
	})

	assert.Equal(t, []int{20, 20, 5}, sizes)

	seen := map[uuid.UUID]bool{}
	for i, row := range read {
		assert.False(t, seen[row.ID], "row %d repeats", i)
		seen[row.ID] = true

		if i > 0 {
			previous := read[i-1]
			assert.False(
				t,
				row.PaidOn.After(previous.PaidOn),
				"row %d is later than row %d",
				i,
				i-1,
			)
		}
	}

	byAmount := map[float64]model.Contribution{}
	for _, row := range read[:3] {
		byAmount[*row.Amount] = row
	}

	assert.Nil(t, byAmount[3].Employer)
	require.NotNil(t, byAmount[2].Employer)
	assert.Equal(
		t,
		model.Employer{Code: employerOther, Name: "შპს " + employerOther},
		*byAmount[2].Employer,
	)
}

// PENSION-SC-038: The page and the total hold the contributions of the employer within
// the period alone. The total counts every match. An employer of no contribution answers
// an empty page and a total of zero.
func TestTheFilterNarrowsThePageAndTheTotal(t *testing.T) {
	t.Parallel()

	store, _, _ := storeUnderTest(t)
	ctxA := actingAs(t, userA)

	rows := make([]model.Contribution, 0, 45)
	for i := range 25 {
		rows = append(rows, paidBy(employerE, on(2026, time.January, 1).AddDate(0, 0, 3*i), 10))
	}

	for i := range 5 {
		rows = append(rows, paidBy(employerE, on(2026, time.April, 1).AddDate(0, 0, i), 1000))
	}

	for i := range 15 {
		rows = append(rows, paidBy(employerOther, on(2026, time.February, 1).AddDate(0, 0, i), 500))
	}

	saveContributions(ctxA, t, store, rows)

	from, to := on(2026, time.January, 1), on(2026, time.March, 31)
	filter := codeOf(employerE)
	filter.From, filter.To = &from, &to

	window, err := store.ContributionsPage(ctxA, page.Request{Limit: pageLimit}, filter)
	require.NoError(t, err)
	assert.Len(t, window.Items, pageLimit)

	for _, row := range window.Items {
		assert.Equal(t, employerE, *row.EmployerCode)
		assert.False(t, row.PaidOn.Before(from) || row.PaidOn.After(to), row.PaidOn)
	}

	total, err := store.ContributionTotal(ctxA, filter)
	require.NoError(t, err)
	assert.Equal(t, model.ContributionTotal{Amount: 250, Count: 25}, total)

	unknown := codeOf("999999999")

	window, err = store.ContributionsPage(ctxA, page.Request{Limit: pageLimit}, unknown)
	require.NoError(t, err)
	assert.Empty(t, window.Items)

	total, err = store.ContributionTotal(ctxA, unknown)
	require.NoError(t, err)
	assert.Equal(t, model.ContributionTotal{}, total)
}

// PENSION-SC-040: The employers hold those that a contribution of the user names, and
// no employer of another user alone.
func TestTheEmployersAreThoseOfTheUser(t *testing.T) {
	t.Parallel()

	store, _, _ := storeUnderTest(t)
	ctxA, ctxB := actingAs(t, userA), actingAs(t, userB)

	saveContributions(ctxA, t, store, []model.Contribution{
		paidBy(employerE, october15(), 1),
		paidBy(employerE, october15(), 2),
		paidBy(employerOther, october15(), 3),
	})
	saveContributions(ctxB, t, store, []model.Contribution{paidBy(employerThird, october15(), 4)})

	employers, err := store.Employers(ctxA)

	require.NoError(t, err)
	assert.ElementsMatch(t, []model.Employer{
		{Code: employerE, Name: "შპს " + employerE},
		{Code: employerOther, Name: "შპს " + employerOther},
	}, employers)
}

// PENSION-SC-042: The pages of the unit prices hold the last price of each ISO week, each
// row once, for each portfolio that the user holds a share of, and no other portfolio.
func TestTheUnitPricesHoldTheLastPriceOfEachWeek(t *testing.T) {
	t.Parallel()

	store, _, _ := storeUnderTest(t)
	ctxA := actingAs(t, userA)

	first, last := on(2026, time.January, 1), on(2026, time.December, 31)

	var prices []model.PortfolioUnitPrice

	lastOfWeek := map[time.Time]time.Time{}

	for day := first; !day.After(last); day = day.AddDate(0, 0, 1) {
		for _, portfolio := range []int64{7, 9} {
			prices = append(prices, model.PortfolioUnitPrice{
				PortfolioID: portfolio, PricedOn: day, UnitPrice: float64(day.YearDay()),
			})
		}

		monday := day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
		lastOfWeek[monday] = day
	}

	require.NoError(t, store.Save(ctxA, model.Snapshot{
		UnitPrices: prices,
		Summary:    summaryOf(1),
		Shares:     []model.PortfolioShare{shareRow(7, 1)},
	}))

	_, weekly := everyPage(
		t,
		func(asked page.Request) (page.Window[model.PortfolioUnitPrice], error) {
			return store.UnitPrices(ctxA, asked, &first)
		},
	)

	got := map[time.Time]time.Time{}

	for _, price := range weekly {
		assert.Equal(t, int64(7), price.PortfolioID)

		monday := price.PricedOn.AddDate(0, 0, -((int(price.PricedOn.Weekday()) + 6) % 7))
		assert.NotContains(t, got, monday, "two prices in the week of %s", monday)
		got[monday] = price.PricedOn.UTC()
	}

	assert.Equal(t, lastOfWeek, got)
}
