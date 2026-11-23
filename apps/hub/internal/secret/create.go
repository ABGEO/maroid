package secret

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
)

// TypeCreate names the task of the outbox that creates one protection key.
const TypeCreate = "secret_key.create"

// CreatePayload is the payload of a task of TypeCreate.
type CreatePayload = Owner

// CreateTask answers the task that creates the key of the owner. It acts for no
// owner, and its dedupe key keeps one task for one key.
func CreateTask(owner Owner) model.OutboxDraft {
	draft := model.OutboxDraft{
		Type:    TypeCreate,
		Payload: owner,
	}

	if key, err := owner.Key(); err == nil {
		draft.DedupeKey = TypeCreate + ":" + string(key)
	}

	return draft
}

// CreateHandler creates the key of the owner that a task names. It serves the outbox,
// which retries a failed creation.
type CreateHandler struct {
	logger *slog.Logger
	db     *sqlx.DB
	keys   Keys
}

// NewCreateHandler creates a CreateHandler over the database and the store of the keys.
func NewCreateHandler(logger *slog.Logger, db *sqlx.DB, keys Keys) *CreateHandler {
	return &CreateHandler{
		logger: logger.With(slog.String("component", "secret_key")),
		db:     db,
		keys:   keys,
	}
}

// Type answers TypeCreate.
func (h *CreateHandler) Type() string { return TypeCreate }

// Handle creates the key of the owner. It reads the owner before and after the
// creation, because a removal that commits in between queues its destruction before
// the key exists. The handler then destroys the key itself.
func (h *CreateHandler) Handle(ctx context.Context, task model.OutboxTask) error {
	var owner CreatePayload
	if err := json.Unmarshal(task.Payload, &owner); err != nil {
		return fmt.Errorf("decoding the payload: %w", err)
	}

	key, err := owner.Key()
	if err != nil {
		return fmt.Errorf("naming the key: %w", err)
	}

	logger := h.logger.With(slog.String("key", string(key)))

	present, err := h.holds(ctx, owner)
	if err != nil {
		return err
	}

	if !present {
		logger.InfoContext(ctx, "skipped the key of an absent owner")

		return nil
	}

	if err = h.keys.CreateKey(ctx, key); err != nil {
		return fmt.Errorf("creating a key: %w", err)
	}

	present, err = h.holds(ctx, owner)
	if err != nil {
		return err
	}

	if present {
		logger.InfoContext(ctx, "created a protection key")

		return nil
	}

	if err = h.keys.DestroyKey(ctx, key); err != nil {
		return fmt.Errorf("destroying the key of an absent owner: %w", err)
	}

	logger.InfoContext(ctx, "destroyed the key of an owner removed during its creation")

	return nil
}

// holds answers whether the record of the owner exists.
func (h *CreateHandler) holds(ctx context.Context, owner Owner) (bool, error) {
	err := database.WithTx(ctx, h.db, func(tx *sqlx.Tx) error {
		if owner.Kind == OwnerUser {
			if _, err := repository.NewUser(tx).GetByID(ctx, owner.ID); err != nil {
				return fmt.Errorf("reading the user record: %w", err)
			}

			return nil
		}

		if _, err := repository.NewWorkspace(tx).GetByID(ctx, owner.ID); err != nil {
			return fmt.Errorf("reading the workspace: %w", err)
		}

		return nil
	})

	switch {
	case errors.Is(err, errs.ErrUserNotFound), errors.Is(err, errs.ErrWorkspaceNotFound):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("reading the owner of the key: %w", err)
	default:
		return true, nil
	}
}
