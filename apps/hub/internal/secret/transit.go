package secret

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/openbao/openbao/api/v2"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
)

// Transit protects a secret with the transit engine of OpenBao.
type Transit struct {
	client *api.Client
	mount  string
}

var _ Cipher = (*Transit)(nil)

// NewTransit creates a new Transit cipher over a client that already logged in.
func NewTransit(client *api.Client, mount string) *Transit {
	return &Transit{
		client: client,
		mount:  mount,
	}
}

// Encrypt returns the protected form of the plaintext, under the named key.
func (c *Transit) Encrypt(
	ctx context.Context,
	key Key,
	plaintext string,
) (string, error) {
	answer, err := c.client.Logical().WriteWithContext(
		ctx,
		c.path("encrypt", key),
		map[string]any{
			"plaintext": base64.StdEncoding.EncodeToString([]byte(plaintext)),
		},
	)
	if err != nil {
		return "", fmt.Errorf("encrypting under the key %q: %w", key, err)
	}

	return readField(answer, "ciphertext")
}

// Decrypt returns the plaintext of the protected form, under the named key.
func (c *Transit) Decrypt(
	ctx context.Context,
	key Key,
	ciphertext string,
) (string, error) {
	answer, err := c.client.Logical().WriteWithContext(
		ctx,
		c.path("decrypt", key),
		map[string]any{"ciphertext": ciphertext},
	)
	if err != nil {
		return "", fmt.Errorf("decrypting under the key %q: %w", key, err)
	}

	encoded, err := readField(answer, "plaintext")
	if err != nil {
		return "", err
	}

	plaintext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decoding the plaintext: %w", err)
	}

	return string(plaintext), nil
}

func (c *Transit) path(operation string, key Key) string {
	return fmt.Sprintf("%s/%s/%s", c.mount, operation, key)
}

// readField reads one field of the answer. An answer with no data is an error,
// never an empty result.
func readField(answer *api.Secret, field string) (string, error) {
	if answer == nil || answer.Data == nil {
		return "", fmt.Errorf("%w: the answer holds no data", errs.ErrProtectionUnavailable)
	}

	value, ok := answer.Data[field].(string)
	if !ok {
		return "", fmt.Errorf(
			"%w: the answer holds no %s",
			errs.ErrProtectionUnavailable,
			field,
		)
	}

	return value, nil
}
