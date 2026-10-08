package user

import (
	"context"
	"io"

	"github.com/abgeo/maroid/apps/hub/internal/provider"
)

// The tests of package user_test reach the unexported functions through these.

func SetPassword(
	ctx context.Context,
	records recordReader,
	accounts provider.LocalAccounts,
	input passwordInput,
	userID string,
	email string,
) error {
	return setPassword(ctx, records, accounts, input, userID, email)
}

func PromptTwice(read func(prompt string) ([]byte, error)) ([]byte, error) {
	return promptTwice(read)
}

func FirstLine(reader io.Reader) ([]byte, error) {
	return firstLine(reader)
}

func WriteInvitation(out io.Writer, userID string, address string) error {
	return writeInvitation(out, userID, address)
}
