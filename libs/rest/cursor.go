package rest

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
)

// Direction says which way a cursor reads from its boundary row.
type Direction string

const (
	// DirectionForward reads the rows after the boundary. A next link carries it.
	DirectionForward Direction = "forward"
	// DirectionBackward reads the rows before it. A prev link carries it.
	DirectionBackward Direction = "backward"
)

var errCursorShape = errors.New("the value is not a cursor")

// Cursor names one position in one collection.
type Cursor struct {
	// Sort holds the fields that order the collection, in the order that the
	// request named them.
	Sort []string `json:"sort"`
	// Direction reads forward from the boundary, or backward.
	Direction Direction `json:"direction"`
	// Filters holds the filters that produced the collection. A request that
	// changes one retires the cursor.
	Filters map[string]string `json:"filters"`
	// Boundary holds the value of each sort field at the boundary row.
	Boundary map[string]string `json:"boundary"`
	// ID is the identifier of the boundary row, which breaks a tie between two
	// rows that share every sort value.
	ID string `json:"id"`
}

// Matches reports whether this cursor belongs to a request with this sort and
// these filters. A cursor that does not match is stale.
func (c Cursor) Matches(sort []string, filters map[string]string) bool {
	if !slices.Equal(c.Sort, sort) {
		return false
	}

	held := c.Filters
	if held == nil {
		held = map[string]string{}
	}

	asked := filters
	if asked == nil {
		asked = map[string]string{}
	}

	return maps.Equal(held, asked)
}

// EncodeCursor answers the opaque value that a link carries. The value carries
// no signature, because it names a position in a collection that the caller
// already reads.
func EncodeCursor(cursor Cursor) (string, error) {
	encoded, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("encoding the cursor: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

// DecodeCursor reads the value that a client passed back. It fails for a value
// that Maroid did not produce, and the caller answers TypeRequestInvalid.
func DecodeCursor(value string) (Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Cursor{}, fmt.Errorf("decoding the cursor: %w", err)
	}

	var cursor Cursor

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	if err = decoder.Decode(&cursor); err != nil {
		return Cursor{}, fmt.Errorf("reading the cursor: %w", err)
	}

	if cursor.ID == "" || cursor.Direction == "" {
		return Cursor{}, fmt.Errorf("reading the cursor: %w", errCursorShape)
	}

	return cursor, nil
}
