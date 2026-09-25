package handler_test

import (
	"context"

	"github.com/abgeo/maroid/libs/rest"
)

// noIdempotency is a key cache that holds nothing. A route under test writes
// once, so the middleware passes every request through to the handler.
type noIdempotency struct{}

var _ rest.IdempotencyStore = noIdempotency{}

func (noIdempotency) Answer(context.Context, string) (rest.IdempotentAnswer, error) {
	return rest.IdempotentAnswer{}, rest.ErrNoIdempotentAnswer
}

func (noIdempotency) Keep(context.Context, string, rest.IdempotentAnswer) error {
	return nil
}
