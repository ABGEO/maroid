package secret

import (
	"context"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
)

// UserKeyPrefix starts the name of the key that protects one user.
const UserKeyPrefix = "maroid-user-"

// WorkspaceKeyPrefix starts the name of the key that protects one workspace.
const WorkspaceKeyPrefix = "maroid-workspace-"

// Key names a key that a Cipher uses.
type Key string

// UserKey returns the Key that protects the secrets of one user.
func UserKey(userID string) Key {
	return Key(UserKeyPrefix + userID)
}

// WorkspaceKey returns the Key that protects the secrets of one workspace.
func WorkspaceKey(workspaceID string) Key {
	return Key(WorkspaceKeyPrefix + workspaceID)
}

// OwnerKind names the kind of record that one protection key protects.
type OwnerKind string

// The kinds of record that hold a protection key.
const (
	OwnerUser      OwnerKind = "user"
	OwnerWorkspace OwnerKind = "workspace"
)

// Owner is the user record or the workspace that one protection key protects.
type Owner struct {
	Kind OwnerKind `json:"kind"`
	ID   string    `json:"id"`
}

// Key answers the Key that protects the secrets of the owner.
func (o Owner) Key() (Key, error) {
	if o.ID == "" {
		return "", errs.ErrSecretKeyOwnerInvalid
	}

	switch o.Kind {
	case OwnerUser:
		return UserKey(o.ID), nil
	case OwnerWorkspace:
		return WorkspaceKey(o.ID), nil
	default:
		return "", errs.ErrSecretKeyOwnerInvalid
	}
}

// Cipher protects a value under a Key.
type Cipher interface {
	// Encrypt returns the protected form of the plaintext, under the given Key.
	Encrypt(ctx context.Context, key Key, plaintext string) (string, error)
	// Decrypt returns the plaintext of the protected form, under the given Key.
	Decrypt(ctx context.Context, key Key, ciphertext string) (string, error)
}
