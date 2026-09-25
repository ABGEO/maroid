package rest

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

const cursorParameter = "cursor"

var errNoBaseURL = errors.New("the context carries no external address")

// Page is one answer of a collection.
type Page[T any] struct {
	// Items holds the rows of this page. It is never null, and an empty
	// collection answers an empty array.
	Items []T `json:"items"`
	// Self is the address of this page.
	Self string `json:"self"`
	// First is the address of the first page of the same collection.
	First string `json:"first"`
	// Next is absent on the last page, which is how a client stops.
	Next *string `json:"next,omitempty"`
	// Prev is absent on the first page.
	Prev *string `json:"prev,omitempty"`
}

// NewPage builds one page and its links. Every link repeats the sort and the
// filters of the request, because a cursor that meets another set is stale.
//
// next and prev carry the boundary rows of this page, and a nil leaves the link
// absent, which is what tells a client that no further page exists.
func NewPage[T any](r *http.Request, items []T, next *Cursor, prev *Cursor) (Page[T], error) {
	if items == nil {
		items = []T{}
	}

	self, err := link(r, r.URL.Query().Get(cursorParameter))
	if err != nil {
		return Page[T]{}, err
	}

	first, err := link(r, "")
	if err != nil {
		return Page[T]{}, err
	}

	page := Page[T]{
		Items: items,
		Self:  self,
		First: first,
	}

	if page.Next, err = cursorLink(r, next); err != nil {
		return Page[T]{}, err
	}

	if page.Prev, err = cursorLink(r, prev); err != nil {
		return Page[T]{}, err
	}

	return page, nil
}

// cursorLink answers the address of one neighbouring page, or nil when no such
// page exists.
func cursorLink(r *http.Request, cursor *Cursor) (*string, error) {
	if cursor == nil {
		return nil, nil //nolint:nilnil // An absent link carries a meaning. RES-005.
	}

	encoded, err := EncodeCursor(*cursor)
	if err != nil {
		return nil, err
	}

	built, err := link(r, encoded)
	if err != nil {
		return nil, err
	}

	return &built, nil
}

// link builds one absolute address for this collection, carrying the query of
// the request with the cursor replaced.
func link(r *http.Request, cursor string) (string, error) {
	base := BaseURLFromContext(r.Context())
	if base == "" {
		return "", fmt.Errorf("building a link: %w", errNoBaseURL)
	}

	query := cloneQuery(r.URL.Query())
	if cursor == "" {
		query.Del(cursorParameter)
	} else {
		query.Set(cursorParameter, cursor)
	}

	address := base + r.URL.EscapedPath()
	if encoded := query.Encode(); encoded != "" {
		address += "?" + encoded
	}

	return address, nil
}

func cloneQuery(values url.Values) url.Values {
	cloned := make(url.Values, len(values))
	for key, held := range values {
		cloned[key] = append([]string(nil), held...)
	}

	return cloned
}
