package handler

import (
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/drivvo/errs"
	"github.com/abgeo/maroid/plugins/drivvo/model"
	"github.com/abgeo/maroid/plugins/drivvo/repository"
)

const (
	paramVehicleID  = "vehicleId"
	paramLogEntryID = "logEntryId"
	filterFrom      = "from"
	filterTo        = "to"
	filterKind      = "kind"

	// canonicalUUIDLength is the length of a UUID in the form 8-4-4-4-12.
	canonicalUUIDLength = 36
)

// orderOf answers the order that the sort of the request names, or the default when it
// names none. A route declares one sort field, so the first field decides.
func orderOf(asked page.Request, fallback repository.Order) repository.Order {
	if len(asked.Sort) == 0 {
		return fallback
	}

	if strings.HasPrefix(strings.TrimSpace(asked.Sort[0]), "-") {
		return repository.Descending
	}

	return repository.Ascending
}

func refuseParameter(name, reason string) *problem.Problem {
	return problem.NewRequestInvalid().
		WithDetail("The " + name + " parameter is not valid: " + reason + ".")
}

// readPeriod reads the days from and to, each optional. A day that does not parse, and a
// from after the to, are refused.
func readPeriod(r *http.Request) (model.Period, *problem.Problem) {
	query := r.URL.Query()

	from, failure := readDay(query.Get(filterFrom), filterFrom)
	if failure != nil {
		return model.Period{}, failure
	}

	to, failure := readDay(query.Get(filterTo), filterTo)
	if failure != nil {
		return model.Period{}, failure
	}

	period, err := model.NewPeriod(from, to)
	if err != nil {
		return model.Period{}, refuseParameter(filterFrom, "it must be on or before the to day")
	}

	return period, nil
}

// readDay reads one day YYYY-MM-DD, nil when the parameter is absent.
func readDay(raw, name string) (*time.Time, *problem.Problem) {
	if raw == "" {
		return nil, nil
	}

	day, err := model.ParseDay(raw)
	if err != nil {
		return nil, refuseParameter(name, "it must be a day YYYY-MM-DD")
	}

	return &day, nil
}

// readKinds reads the repeated kind parameter. Absent names every kind.
func readKinds(r *http.Request) ([]model.Kind, *problem.Problem) {
	var kinds []model.Kind

	for _, raw := range r.URL.Query()[filterKind] {
		kind, ok := parseKind(raw)
		if !ok {
			return nil, refuseParameter(
				filterKind,
				"it must be one of "+strings.Join(kindNames(), ", "),
			)
		}

		if !slices.Contains(kinds, kind) {
			kinds = append(kinds, kind)
		}
	}

	return kinds, nil
}

func parseKind(raw string) (model.Kind, bool) {
	kind := model.Kind(raw)

	return kind, slices.Contains(model.Kinds(), kind)
}

func kindNames() []string {
	names := make([]string, 0, len(model.Kinds()))
	for _, kind := range model.Kinds() {
		names = append(names, string(kind))
	}

	return names
}

// pathID reads the identifier segment of the route that name names. A value that is no
// UUID names no row, so it answers the not found error before any query runs.
func pathID(r *http.Request, name string, notFound error) (string, error) {
	id := chi.URLParam(r, name)

	if _, err := uuid.Parse(id); err != nil || len(id) != canonicalUUIDLength {
		return "", notFound
	}

	return id, nil
}

// vehicleOf reads the vehicle segment of the route.
func vehicleOf(r *http.Request) (string, error) {
	return pathID(r, paramVehicleID, errs.ErrVehicleNotFound)
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
