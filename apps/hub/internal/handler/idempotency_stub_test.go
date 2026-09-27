package handler_test

import (
	"context"

	"github.com/abgeo/maroid/libs/rest/idempotency"
)

// noIdempotency is a key cache that holds nothing. A route under test writes
// once, so the middleware passes every request through to the handler.
type noIdempotency struct{}

var _ idempotency.Store = noIdempotency{}

func (noIdempotency) Answer(context.Context, string) (idempotency.Answer, error) {
	return idempotency.Answer{}, idempotency.ErrNoAnswer
}

func (noIdempotency) Keep(context.Context, string, idempotency.Answer) error {
	return nil
}
