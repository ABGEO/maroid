package secret

import "context"

// UserKeyPrefix starts the name of the key that protects one user.
// The owner creates each key under this prefix.
const UserKeyPrefix = "maroid-user-"

// WorkspaceKeyPrefix starts the name of the key that protects one workspace.
// The owner creates each key under this prefix.
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

// Cipher protects a value under a Key.
type Cipher interface {
	// Encrypt returns the protected form of the plaintext, under the given Key.
	Encrypt(ctx context.Context, key Key, plaintext string) (string, error)
	// Decrypt returns the plaintext of the protected form, under the given Key.
	Decrypt(ctx context.Context, key Key, ciphertext string) (string, error)
}
