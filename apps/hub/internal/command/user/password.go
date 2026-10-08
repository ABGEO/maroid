package user

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/abgeo/maroid/apps/hub/internal/depresolver"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/provider"
)

// PasswordCommand gives a user record a local account, or a new password for the one
// it holds. It is the way to the first sign in, and the way back in after a lockout.
type PasswordCommand struct {
	depResolver depresolver.Resolver
	logger      *slog.Logger
	input       passwordInput

	userID string
	email  string
}

// recordReader reads a user record, active or blocked.
type recordReader interface {
	Get(ctx context.Context, userID string) (*model.User, error)
}

// passwordInput reads the new password.
type passwordInput interface {
	Password() ([]byte, error)
}

// NewPasswordCommand creates a new PasswordCommand that reads the password from the
// standard input.
func NewPasswordCommand(depResolver depresolver.Resolver) *PasswordCommand {
	command := &PasswordCommand{
		depResolver: depResolver,
		input:       consoleInput{in: os.Stdin, prompts: os.Stderr},
	}

	if depResolver != nil {
		command.logger = depResolver.Logger().With(
			slog.String("component", "command"),
			slog.String("command", "user password"),
		)
	}

	return command
}

// Command initializes and returns the Cobra command. No flag takes the password,
// because a command line reaches the shell history and the process list.
func (c *PasswordCommand) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "password",
		Short: "Give a user record a local account, or a new password for the one it holds",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&c.userID, "user", "", "The identifier of the user record")
	cmd.Flags().StringVar(&c.email, "email", "", "The email address of a new local account")
	_ = cmd.MarkFlagRequired("user")

	return cmd
}

func (c *PasswordCommand) run(ctx context.Context) error {
	records, err := c.depResolver.UserService()
	if err != nil {
		return fmt.Errorf("resolving the user service: %w", err)
	}

	accounts, err := c.depResolver.LocalAccounts()
	if err != nil {
		return fmt.Errorf("resolving the local accounts: %w", err)
	}

	if err = setPassword(ctx, records, accounts, c.input, c.userID, c.email); err != nil {
		return err
	}

	c.logger.InfoContext(ctx, "set the local account", slog.String("user_id", c.userID))

	return nil
}

// setPassword runs every check and reads the password before the first write, so a
// refused run changes nothing.
func setPassword(
	ctx context.Context,
	records recordReader,
	accounts provider.LocalAccounts,
	input passwordInput,
	userID string,
	email string,
) error {
	record, err := records.Get(ctx, userID)
	if err != nil {
		return fmt.Errorf("reading the user record: %w", err)
	}

	held, err := holdsAccount(ctx, accounts, record.ID, email)
	if err != nil {
		return err
	}

	password, err := input.Password()
	if err != nil {
		return fmt.Errorf("reading the password: %w", err)
	}

	if err = accounts.EnsureProvider(ctx); err != nil {
		return fmt.Errorf("adding the local provider: %w", err)
	}

	if held {
		err = accounts.Reset(ctx, record.ID, password)
	} else {
		err = accounts.Give(ctx, record.ID, email, password)
	}

	if err != nil {
		return fmt.Errorf("setting the local account: %w", err)
	}

	return nil
}

// holdsAccount reports whether the record holds a local account, and refuses an email
// address that does not match that answer.
func holdsAccount(
	ctx context.Context,
	accounts provider.LocalAccounts,
	userID string,
	email string,
) (bool, error) {
	held, err := accounts.HasAccount(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("reading the local account: %w", err)
	}

	switch {
	case held && email != "":
		return false, errs.ErrLocalAccountEmail
	case !held && email == "":
		return false, errs.ErrLocalAccountEmailMissing
	default:
		return held, nil
	}
}

// consoleInput reads the password from a terminal with no echo, or from the first line
// of a standard input that is no terminal.
type consoleInput struct {
	in      *os.File
	prompts io.Writer
}

func (c consoleInput) Password() ([]byte, error) {
	descriptor := int(c.in.Fd())

	if !term.IsTerminal(descriptor) {
		return firstLine(c.in)
	}

	return promptTwice(func(prompt string) ([]byte, error) {
		_, _ = fmt.Fprint(c.prompts, prompt)

		password, err := term.ReadPassword(descriptor)

		_, _ = fmt.Fprintln(c.prompts)

		if err != nil {
			return nil, fmt.Errorf("reading from the terminal: %w", err)
		}

		return password, nil
	})
}

// promptTwice reads the password twice and refuses two different entries, because no
// echo shows a typing mistake.
func promptTwice(read func(prompt string) ([]byte, error)) ([]byte, error) {
	password, err := read("Password: ")
	if err != nil {
		return nil, err
	}

	repeated, err := read("Repeat the password: ")
	if err != nil {
		return nil, err
	}

	if string(password) != string(repeated) {
		return nil, errs.ErrPasswordMismatch
	}

	return password, nil
}

// firstLine answers the first line of the reader, without its line break.
func firstLine(reader io.Reader) ([]byte, error) {
	line, err := bufio.NewReader(reader).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("reading the standard input: %w", err)
	}

	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	if line == "" {
		return nil, errs.ErrPasswordMissing
	}

	return []byte(line), nil
}
