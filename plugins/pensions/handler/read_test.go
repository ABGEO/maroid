package handler_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/pensions/handler"
	"github.com/abgeo/maroid/plugins/pensions/model"
	"github.com/abgeo/maroid/plugins/pensions/service"
)

// heldReads answers fixed reads, and records the filter of each read of the contributions.
type heldReads struct {
	summary *model.AccountSummary
	filters []model.ContributionFilter
}

func (h *heldReads) Summary(context.Context) (*model.AccountSummary, error) {
	return h.summary, nil
}

func (h *heldReads) ContributionsPage(
	_ context.Context,
	_ page.Request,
	filter model.ContributionFilter,
) (page.Window[model.Contribution], error) {
	h.filters = append(h.filters, filter)

	return page.Window[model.Contribution]{Items: []model.Contribution{{
		ID: uuid.New(), Source: "დამსაქმებელი", ContributionType: "ჩვეულებრივი",
		PaidOn: time.Date(2026, time.October, 15, 0, 0, 0, 0, time.UTC), GrossSalary: 5000,
	}}}, nil
}

func (h *heldReads) ContributionTotal(
	_ context.Context,
	filter model.ContributionFilter,
) (model.ContributionTotal, error) {
	h.filters = append(h.filters, filter)

	return model.ContributionTotal{Amount: 100, Count: 1}, nil
}

func (h *heldReads) ContributionMonths(context.Context, *time.Time) ([]model.MonthSum, error) {
	return nil, nil
}

func (h *heldReads) Employers(context.Context) ([]model.Employer, error) {
	return []model.Employer{{Code: "204567890", Name: "შპს მაგალითი"}}, nil
}

func (h *heldReads) AssetValues(
	context.Context,
	page.Request,
	*time.Time,
) (page.Window[model.AssetValue], error) {
	return page.Window[model.AssetValue]{}, nil
}

func (h *heldReads) PortfolioShares(context.Context) ([]model.PortfolioShare, error) {
	return []model.PortfolioShare{{
		PortfolioID: 7, PortfolioNameKa: "ზომიერი", PortfolioNameEn: "Balanced",
		AssetManagerNameKa: "სააგენტო", AssetManagerNameEn: "Agency", Ticker: "GPA-B",
		RiskTierNameKa: "საშუალო", RiskTierNameEn: "Medium", TargetShare: 100,
		CurrentShare: 98.7, Amount: 12480.55, Units: 1000.5, UnitPrice: 12.47,
	}}, nil
}

func (h *heldReads) UnitPrices(
	context.Context,
	page.Request,
	*time.Time,
) (page.Window[model.PortfolioUnitPrice], error) {
	return page.Window[model.PortfolioUnitPrice]{}, nil
}

// readRoutes answers every read route over the held reads and the secrets.
func readRoutes(reads *heldReads, secrets *fakeSecrets) []pluginapi.Route {
	logger := slog.New(slog.DiscardHandler)
	reader := service.NewReader(secrets, reads)

	return slices.Concat(
		handler.NewSummaryHandler(logger, reader).Routes(),
		handler.NewAssetValueHandler(logger, reads).Routes(),
		handler.NewContributionHandler(logger, reads).Routes(),
		handler.NewPortfolioHandler(logger, reads).Routes(),
	)
}

func get(t *testing.T, routes []pluginapi.Route, target string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(
		pluginapi.ContextWithActingUser(t.Context(), userA),
		http.MethodGet,
		target,
		nil,
	)

	for _, route := range routes {
		if route.Method == http.MethodGet && route.Pattern == request.URL.Path {
			recorder := httptest.NewRecorder()
			address.Middleware("https://maroid.test")(route.Handler).ServeHTTP(recorder, request)

			return recorder
		}
	}

	t.Fatalf("no route GET %s", request.URL.Path)

	return nil
}

// PENSION-SC-034: The summary reports whether the user is connected, and holds the
// summary when a collection stored one, also after the end of the connection.
func TestTheSummaryNamesTheStateOfTheUser(t *testing.T) {
	t.Parallel()

	stored := &model.AccountSummary{
		AssetValue: 12480.55, Units: 1000.5, Gain: 1904.10, PersonContributions: 3500,
		EmployerContributions: 3500, StateContributions: 3576.45, TotalContributions: 10576.45,
		CollectedAt: time.Date(2026, time.November, 1, 6, 0, 0, 0, time.UTC),
	}

	cases := map[string]struct {
		summary   *model.AccountSummary
		token     string
		connected bool
	}{
		"user B": {},
		"user C": {token: tokenNew, connected: true},
		"user D": {summary: stored},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			recorder := get(
				t,
				readRoutes(&heldReads{summary: setup.summary}, secretsHolding(setup.token)),
				"/account-summaries/self",
			)
			require.Equal(t, http.StatusOK, recorder.Code)

			var answer map[string]any
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answer))
			assert.Equal(t, setup.connected, answer["connected"])

			if setup.summary == nil {
				assert.NotContains(t, answer, "summary")

				return
			}

			assert.JSONEq(t, `{
				"connected": false,
				"summary": {
					"asset_value": {"amount": 12480.55, "currency": "GEL"},
					"units": 1000.5,
					"gain": {"amount": 1904.10, "currency": "GEL"},
					"withheld_amount": {"amount": 0.00, "currency": "GEL"},
					"contributions": {
						"person": {"amount": 3500.00, "currency": "GEL"},
						"employer": {"amount": 3500.00, "currency": "GEL"},
						"state": {"amount": 3576.45, "currency": "GEL"},
						"total": {"amount": 10576.45, "currency": "GEL"}
					},
					"collected_at": "2026-11-01T06:00:00Z"
				}
			}`, recorder.Body.String())
		})
	}
}

// PENSION-SC-039: A first date after the last date, a date that does not parse, and a
// parameter that the route does not declare answer 400 and name the parameter.
func TestTheContributionsRefuseABadFilter(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		target string
		names  string
	}{
		"reversed":   {target: "/contributions?from=2026-04-01&to=2026-03-01", names: "from"},
		"bad month":  {target: "/contributions?from=2026-13-01", names: "from"},
		"bad to":     {target: "/contributions/total?to=yesterday", names: "to"},
		"undeclared": {target: "/contributions?sort=amount", names: "sort"},
		"bounded":    {target: "/employers?limit=5", names: "limit"},
		"over limit": {target: "/asset-values?limit=101", names: "limit"},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			reads := &heldReads{}
			recorder := get(t, readRoutes(reads, secretsHolding("")), setup.target)

			assert.Equal(t, http.StatusBadRequest, recorder.Code)
			assert.Equal(t, "/problems/http/request-invalid", problemTypeOf(t, recorder))
			assert.Contains(t, recorder.Body.String(), setup.names)
			assert.Empty(t, reads.filters)
		})
	}
}

// PENSION-SC-041: A share answers the English names, the ticker, both percents, the
// amount as a money object, the units, and the unit price.
func TestTheSharesAnswerTheEnglishNames(t *testing.T) {
	t.Parallel()

	recorder := get(t, readRoutes(&heldReads{}, secretsHolding("")), "/portfolio-shares")
	require.Equal(t, http.StatusOK, recorder.Code)

	var answer struct {
		Items []json.RawMessage `json:"items"`
	}

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answer))
	require.Len(t, answer.Items, 1)
	assert.JSONEq(t, `{
		"portfolio_id": 7,
		"portfolio_name": "Balanced",
		"asset_manager_name": "Agency",
		"ticker": "GPA-B",
		"risk_tier_name": "Medium",
		"target_share": 100,
		"current_share": 98.7,
		"amount": {"amount": 12480.55, "currency": "GEL"},
		"units": 1000.5,
		"unit_price": 12.47
	}`, string(answer.Items[0]))
}
