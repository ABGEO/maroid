package secret

import "context"

// UserKeyPrefix starts the name of the key that protects one user.
// The owner creates each key under this prefix.
const UserKeyPrefix = "maroid-user-"

// Key names a key that a Cipher uses.
type Key string

// UserKey returns the Key that protects the secrets of one user.
func UserKey(userID string) Key {
	return Key(UserKeyPrefix + userID)
}

// Cipher protects a value under a Key.
type Cipher interface {
	// Encrypt returns the protected form of the plaintext, under the given Key.
	Encrypt(ctx context.Context, key Key, plaintext string) (string, error)
	// Decrypt returns the plaintext of the protected form, under the given Key.
	Decrypt(ctx context.Context, key Key, ciphertext string) (string, error)
}
