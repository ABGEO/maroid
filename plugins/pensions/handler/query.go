package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/pensions/model"
)

const (
	filterEmployer = "employer"
	filterFrom     = "from"
	filterTo       = "to"
)

// readDate reads a filter that is a date YYYY-MM-DD. An absent filter answers nil.
func readDate(raw string, name string) (*time.Time, *problem.Problem) {
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

// readFrom reads the filter from of a bounded collection, and refuses every other
// parameter.
func readFrom(r *http.Request) (*time.Time, *problem.Problem) {
	asked, failure := page.ReadRequest(
		r,
		page.Options{Bounded: true, Filters: []string{filterFrom}},
	)
	if failure != nil {
		return nil, failure
	}

	return readDate(asked.Filters[filterFrom], filterFrom)
}

// readFromPage reads one page of a collection that grows over the years, and its filter
// from.
func readFromPage(r *http.Request) (page.Request, *time.Time, *problem.Problem) {
	asked, failure := page.ReadRequest(r, page.Options{Filters: []string{filterFrom}})
	if failure != nil {
		return page.Request{}, nil, failure
	}

	from, failure := readDate(asked.Filters[filterFrom], filterFrom)
	if failure != nil {
		return page.Request{}, nil, failure
	}

	return asked, from, nil
}

// readContributionFilter reads the employer and the period of the contributions. A
// period whose first date is after its last date is refused.
func readContributionFilter(
	filters map[string]string,
) (model.ContributionFilter, *problem.Problem) {
	var filter model.ContributionFilter

	if employer := filters[filterEmployer]; employer != "" {
		filter.EmployerCode = &employer
	}

	var failure *problem.Problem

	if filter.From, failure = readDate(filters[filterFrom], filterFrom); failure != nil {
		return model.ContributionFilter{}, failure
	}

	if filter.To, failure = readDate(filters[filterTo], filterTo); failure != nil {
		return model.ContributionFilter{}, failure
	}

	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return model.ContributionFilter{}, problem.NewRequestInvalid().
			WithDetail("The " + filterFrom + " parameter is after the " + filterTo + " parameter.")
	}

	return filter, nil
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

func loggerOf(logger *slog.Logger, name string) *slog.Logger {
	return logger.With(slog.String("component", "handler"), slog.String("handler", name))
}
