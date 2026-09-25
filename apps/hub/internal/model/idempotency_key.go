package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var errHeadersNotBytes = errors.New("the headers column did not answer bytes")

// IdempotencyKey is the answer of one write, kept under the key that a client
// picked so that repeating that write is safe.
type IdempotencyKey struct {
	ID          string    `db:"id"`
	UserID      string    `db:"user_id"`
	Key         string    `db:"key"`
	RequestHash string    `db:"request_hash"`
	Status      int       `db:"status"`
	Headers     Headers   `db:"headers"`
	Body        []byte    `db:"body"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// Headers is the header set of an answer, stored as one JSON object.
//
// Value takes the map and Scan fills it, which is the pair that database/sql
// asks for. Fields carries the same waiver.
//
//nolint:recvcheck
type Headers map[string][]string

// Value implements driver.Valuer.
func (h Headers) Value() (driver.Value, error) {
	encoded, err := json.Marshal(h)
	if err != nil {
		return nil, fmt.Errorf("encoding the headers: %w", err)
	}

	return encoded, nil
}

// Scan implements sql.Scanner.
func (h *Headers) Scan(value any) error {
	raw, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("%w: %T", errHeadersNotBytes, value)
	}

	if err := json.Unmarshal(raw, h); err != nil {
		return fmt.Errorf("decoding the headers: %w", err)
	}

	return nil
}
