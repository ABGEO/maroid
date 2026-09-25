package rest

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
)

const (
	// DefaultPageLimit is the count of items that a request with no limit takes.
	DefaultPageLimit = 20
	// MaxPageLimit is the ceiling that a request may not pass.
	MaxPageLimit = 100

	limitParameter = "limit"
	sortParameter  = "sort"
)

// PageOptions declares what one route takes. A route fills it once, and the
// reader refuses every parameter that the route does not declare.
type PageOptions struct {
	// Bounded marks a collection that the deployment bounds, which answers every
	// item in one page and takes no limit and no cursor.
	Bounded bool
	// SortFields names the fields that sort takes. A route that names none
	// sorts by the identifier, and refuses every field that a request names.
	SortFields []string
	// Filters names the query parameters that narrow the collection. A cursor
	// carries their values, so a request that changes one retires it.
	Filters []string
}

// PageRequest holds what one request asks of a collection.
type PageRequest struct {
	// Limit is the count of items to answer. It is zero for a bounded route.
	Limit int
	// Cursor names the position to read from, or nil for the first page.
	Cursor *Cursor
	// Sort holds the fields that the request named, each one optionally
	// prefixed with a sign.
	Sort []string
	// Filters holds the value of each declared filter that the request carries.
	Filters map[string]string
}

// ReadPageRequest reads limit, cursor and sort from the query and checks each
// one against what the route declares. It answers a problem that the caller
// writes, and a nil problem means the request is good.
func ReadPageRequest(r *http.Request, declared PageOptions) (PageRequest, *Problem) {
	query := r.URL.Query()
	asked := PageRequest{
		Filters: readFilters(r, declared),
	}

	if declared.Bounded {
		for _, name := range []string{limitParameter, cursorParameter} {
			if query.Has(name) {
				return PageRequest{}, refuse(name, "this collection answers every item in one page")
			}
		}
	} else {
		limit, problem := readLimit(query.Get(limitParameter))
		if problem != nil {
			return PageRequest{}, problem
		}

		asked.Limit = limit
	}

	sort, problem := readSort(query.Get(sortParameter), declared)
	if problem != nil {
		return PageRequest{}, problem
	}

	asked.Sort = sort

	if raw := query.Get(cursorParameter); raw != "" {
		cursor, err := DecodeCursor(raw)
		if err != nil {
			return PageRequest{}, refuse(
				cursorParameter, "the value is not a cursor that this API produced",
			)
		}

		if !cursor.Matches(asked.Sort, asked.Filters) {
			stale := NewCursorStale()

			return PageRequest{}, &stale
		}

		asked.Cursor = &cursor
	}

	return asked, nil
}

// readLimit answers the default when the request names none, and refuses a
// value outside the bounds that RES-005 gives.
func readLimit(raw string) (int, *Problem) {
	if raw == "" {
		return DefaultPageLimit, nil
	}

	limit, err := strconv.Atoi(raw)
	if err != nil {
		return 0, refuse(limitParameter, "the value is not a number")
	}

	if limit < 1 || limit > MaxPageLimit {
		return 0, refuse(limitParameter, "the value is outside 1 to "+strconv.Itoa(MaxPageLimit))
	}

	return limit, nil
}

// readSort refuses every field that the route does not declare, and the detail
// names that field so a client learns the set from the failure.
func readSort(raw string, declared PageOptions) ([]string, *Problem) {
	if raw == "" {
		return nil, nil
	}

	fields := strings.Split(raw, ",")
	for _, field := range fields {
		bare := strings.TrimLeft(strings.TrimSpace(field), "+-")
		if !slices.Contains(declared.SortFields, bare) {
			return nil, refuse(sortParameter, "this route does not sort by "+bare)
		}
	}

	return fields, nil
}

// readFilters keeps the declared filters that the request carries, so that a
// cursor and a request compare on the same set.
func readFilters(r *http.Request, declared PageOptions) map[string]string {
	query := r.URL.Query()
	filters := map[string]string{}

	for _, name := range declared.Filters {
		if value := query.Get(name); value != "" {
			filters[name] = value
		}
	}

	return filters
}

// refuse builds the failure of one parameter, naming it so a client learns
// which value to correct. ERR-005 bounds the detail.
func refuse(parameter string, reason string) *Problem {
	problem := NewRequestInvalid().
		WithDetail("The " + parameter + " parameter is not valid: " + reason + ".")

	return &problem
}
