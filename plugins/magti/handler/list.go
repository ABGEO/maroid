package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/magti/repository"
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
