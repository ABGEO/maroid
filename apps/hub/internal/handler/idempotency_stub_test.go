package handler_test

import (
	"context"

	"github.com/abgeo/maroid/libs/rest/idempotency"
)

// noIdempotency is a key cache that holds nothing. A route under test writes
// once, so the middleware passes every request through to the handler.
type noIdempotency struct{}

var _ idempotency.Store = noIdempotency{}

func (noIdempotency) Reserve(context.Context, string, string) (*idempotency.Answer, error) {
	return nil, nil //nolint:nilnil // every request claims its key and runs.
}

func (noIdempotency) Complete(context.Context, string, idempotency.Answer) error {
	return nil
}

func (noIdempotency) Release(context.Context, string) error {
	return nil
}
