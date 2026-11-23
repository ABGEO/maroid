package secret

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
)

// TypeDestroy names the task of the outbox that destroys one key of the secret store.
const TypeDestroy = "secret_key.destroy"

// DestroyPayload is the payload of a task of TypeDestroy.
type DestroyPayload struct {
	Key Key `json:"key"`
}

// DestroyTask answers the task that destroys the key. It acts for no owner, and its
// dedupe key keeps one task for one key.
func DestroyTask(key Key) model.OutboxDraft {
	return model.OutboxDraft{
		Type:      TypeDestroy,
		Payload:   DestroyPayload{Key: key},
		DedupeKey: TypeDestroy + ":" + string(key),
	}
}

// DestroyHandler destroys the key that a task names. It serves the outbox, which
// retries a failed destruction.
type DestroyHandler struct {
	keys KeyDestroyer
}

// NewDestroyHandler creates a DestroyHandler over the store of the keys.
func NewDestroyHandler(keys KeyDestroyer) *DestroyHandler {
	return &DestroyHandler{keys: keys}
}

// Type answers TypeDestroy.
func (h *DestroyHandler) Type() string { return TypeDestroy }

// Handle destroys the key of the payload. It checks no prefix of the key, so a later
// change can destroy a key of a user with the same type.
func (h *DestroyHandler) Handle(ctx context.Context, task model.OutboxTask) error {
	var payload DestroyPayload
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return fmt.Errorf("decoding the payload: %w", err)
	}

	if payload.Key == "" {
		return errs.ErrSecretKeyMissing
	}

	if err := h.keys.DestroyKey(ctx, payload.Key); err != nil {
		return fmt.Errorf("destroying a key: %w", err)
	}

	return nil
}
