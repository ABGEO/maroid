package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/parking/model"
	"github.com/abgeo/maroid/plugins/parking/permission"
)

const (
	filterFrom  = "from"
	filterTo    = "to"
	filterPlate = "plate"
	filterFree  = "free"
	filterKind  = "kind"

	maxPlateLength = 16
)

// period is the first and the last day of a filter, each one included.
type period struct {
	from *time.Time
	to   *time.Time
}

// readDay reads a filter that is a date YYYY-MM-DD. An absent filter answers nil.
func readDay(raw string, name string) (*time.Time, *problem.Problem) {
	if raw == "" {
		return nil, nil
	}

	day, err := time.Parse(time.DateOnly, raw)
	if err != nil {
		return nil, problem.NewRequestInvalid().
			WithDetail("The " + name + " parameter is not valid: it must be a date YYYY-MM-DD.")
	}

	return &day, nil
}

// readPeriod reads from and to. A period whose first date is after its last date is
// refused.
func readPeriod(filters map[string]string) (period, *problem.Problem) {
	from, failure := readDay(filters[filterFrom], filterFrom)
	if failure != nil {
		return period{}, failure
	}

	to, failure := readDay(filters[filterTo], filterTo)
	if failure != nil {
		return period{}, failure
	}

	if from != nil && to != nil && from.After(*to) {
		return period{}, problem.NewRequestInvalid().
			WithDetail("The " + filterFrom + " parameter is after the " + filterTo + " parameter.")
	}

	return period{from: from, to: to}, nil
}

// points answers the period as points in time: the start of the first day, and the start
// of the day after the last day.
//
// PARKHIST-DD-015: A day of a person begins at the midnight of their own time zone.
func (p period) points(ctx context.Context) (*time.Time, *time.Time) {
	zone := pluginapi.ActingTimeZone(ctx)

	return startOf(p.from, zone, 0), startOf(p.to, zone, 1)
}

func startOf(day *time.Time, zone *time.Location, offset int) *time.Time {
	if day == nil {
		return nil
	}

	start := time.Date(day.Year(), day.Month(), day.Day()+offset, 0, 0, 0, 0, zone)

	return &start
}

// readSessionFilter reads the period, the plate, and the free filter of the sessions.
func readSessionFilter(
	ctx context.Context,
	filters map[string]string,
) (model.SessionFilter, *problem.Problem) {
	within, failure := readPeriod(filters)
	if failure != nil {
		return model.SessionFilter{}, failure
	}

	var filter model.SessionFilter

	filter.From, filter.To = within.points(ctx)

	if plate := filters[filterPlate]; plate != "" {
		if utf8.RuneCountInString(plate) > maxPlateLength {
			return model.SessionFilter{}, problem.NewRequestInvalid().
				WithDetail("The " + filterPlate + " parameter is not valid: it holds at most 16 characters.")
		}

		filter.Plate = &plate
	}

	switch filters[filterFree] {
	case "":
	case "true":
		filter.Free = new(true)
	case "false":
		filter.Free = new(false)
	default:
		return model.SessionFilter{}, problem.NewRequestInvalid().
			WithDetail("The " + filterFree + " parameter is not valid: it must be true or false.")
	}

	return filter, nil
}

// readTransactionFilter reads the period and the kind of the transactions.
func readTransactionFilter(
	ctx context.Context,
	filters map[string]string,
) (model.TransactionFilter, *problem.Problem) {
	within, failure := readPeriod(filters)
	if failure != nil {
		return model.TransactionFilter{}, failure
	}

	var filter model.TransactionFilter

	filter.From, filter.To = within.points(ctx)

	if raw := filters[filterKind]; raw != "" {
		kind := model.TransactionKind(raw)

		switch kind {
		case model.TransactionKindTopUp, model.TransactionKindParkingPayment,
			model.TransactionKindPermitPurchase, model.TransactionKindOther:
			filter.Kind = &kind
		default:
			return model.TransactionFilter{}, problem.NewRequestInvalid().WithDetail(
				"The " + filterKind + " parameter is not valid: it must be top_up, " +
					"parking_payment, permit_purchase, or other.",
			)
		}
	}

	return filter, nil
}

// readFrom reads the filter from of a bounded collection as a point in time, and refuses
// every other parameter.
func readFrom(r *http.Request) (*time.Time, *problem.Problem) {
	asked, failure := page.ReadRequest(
		r,
		page.Options{Bounded: true, Filters: []string{filterFrom}},
	)
	if failure != nil {
		return nil, failure
	}

	within, failure := readPeriod(asked.Filters)
	if failure != nil {
		return nil, failure
	}

	from, _ := within.points(r.Context())

	return from, nil
}

// writePage answers one page of a collection with its links.
func writePage[T any](
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	items []T,
	next *page.Cursor,
	prev *page.Cursor,
) {
	answered, err := page.New(r, items, next, prev)
	if err != nil {
		logger.ErrorContext(r.Context(), "building a page failed", slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

		return
	}

	writeOK(w, r, logger, answered)
}

func writeOK(w http.ResponseWriter, r *http.Request, logger *slog.Logger, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		logger.ErrorContext(r.Context(), "failed to encode the answer", slog.Any("error", err))
	}
}

func loggerOf(logger *slog.Logger, name string) *slog.Logger {
	return logger.With(slog.String("component", "handler"), slog.String("handler", name))
}

// historyRoute answers a GET route that names history.read.
//
// PARKHIST-FR-045: Each route of the stored data names history.read.
func historyRoute(pattern string, handle http.HandlerFunc) pluginapi.Route {
	return pluginapi.Route{
		Method:     http.MethodGet,
		Pattern:    pattern,
		Handler:    handle,
		Permission: permission.HistoryRead,
	}
}
