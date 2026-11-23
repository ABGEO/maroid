package outbox_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/outbox"
)

const probeRun = "probe.run"

// probeHandler serves one type with a function of the test.
type probeHandler struct {
	taskType string
	handle   func(ctx context.Context, task outbox.Task) error
}

func (p probeHandler) Type() string { return p.taskType }

func (p probeHandler) Handle(ctx context.Context, task outbox.Task) error {
	return p.handle(ctx, task)
}

func noop(context.Context, outbox.Task) error { return nil }

// JOB-010: One handler serves one type. A second handler of a type refuses the whole
// call, and the registry keeps what it held.
func TestTheRegistryRejectsASecondHandlerOfAType(t *testing.T) {
	t.Parallel()

	registry := outbox.NewRegistry()
	require.NoError(t, registry.Register(probeHandler{probeRun, noop}))

	err := registry.Register(probeHandler{"probe.other", noop}, probeHandler{probeRun, noop})
	require.ErrorIs(t, err, outbox.ErrHandlerRegistered)

	_, found := registry.Get("probe.other")
	require.False(t, found)

	err = registry.Register(probeHandler{"probe.twice", noop}, probeHandler{"probe.twice", noop})
	require.ErrorIs(t, err, outbox.ErrHandlerRegistered)

	require.Len(t, registry.All(), 1)
}
