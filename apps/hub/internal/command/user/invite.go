package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/depresolver"
)

// ErrConflictingFlags indicates that the caller named a record and a name at once.
var ErrConflictingFlags = errors.New(
	"user invite: --user excludes --first-name and --last-name",
)

// InviteCommand issues the grant that binds the first identity of a user record.
type InviteCommand struct {
	depResolver depresolver.Resolver
	logger      *slog.Logger

	userID    string
	firstName string
	lastName  string
	ttl       time.Duration
}

// NewInviteCommand creates a new InviteCommand.
func NewInviteCommand(depResolver depresolver.Resolver) *InviteCommand {
	return &InviteCommand{
		depResolver: depResolver,
		logger: depResolver.Logger().With(
			slog.String("component", "command"),
			slog.String("command", "user invite"),
		),
	}
}

// Command initializes and returns the Cobra command.
func (c *InviteCommand) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invite",
		Short: "Create a user record and print the address that binds its first account",
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return c.validate()
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&c.userID, "user", "", "Issue for a record that exists")
	cmd.Flags().StringVar(&c.firstName, "first-name", "", "The first name of a new record")
	cmd.Flags().StringVar(&c.lastName, "last-name", "", "The last name of a new record")
	cmd.Flags().DurationVar(&c.ttl, "ttl", 0, "How long the invitation stays valid")

	return cmd
}

// validate refuses a request that names a record and a name at once.
func (c *InviteCommand) validate() error {
	if c.userID != "" && (c.firstName != "" || c.lastName != "") {
		return ErrConflictingFlags
	}

	return nil
}

func (c *InviteCommand) run(ctx context.Context) error {
	cfg := c.depResolver.Config()

	authSvc, err := c.depResolver.AuthService()
	if err != nil {
		return fmt.Errorf("resolving the auth service: %w", err)
	}

	ttl := c.ttl
	if ttl == 0 {
		ttl = cfg.Auth.InvitationTTL
	}

	result, err := authSvc.Invite(ctx, auth.InviteRequest{
		UserID:    c.userID,
		FirstName: c.firstName,
		LastName:  c.lastName,
	}, ttl)
	if err != nil {
		return fmt.Errorf("issuing the invitation: %w", err)
	}

	address, err := auth.InviteAddress(cfg.Auth.DeckURL, result.Token)
	if err != nil {
		return fmt.Errorf("building the address of the invitation: %w", err)
	}

	c.logger.InfoContext(
		ctx,
		"issued an invitation",
		slog.String("user_id", result.UserID),
		slog.Time("expires_at", time.Now().Add(ttl)),
	)

	fmt.Println(address) //nolint:forbidigo // the address is the result of the command.

	return nil
}
