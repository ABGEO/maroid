// Package user holds the commands that act on a user record.
package user

import (
	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/depresolver"
)

// Command represents the `user` branch of the command tree.
type Command struct {
	depResolver depresolver.Resolver
}

// New creates a new Command.
func New(depResolver depresolver.Resolver) *Command {
	return &Command{depResolver: depResolver}
}

// Command initializes and returns the Cobra command.
func (c *Command) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Act on a user record",
	}

	cmd.AddCommand(NewInviteCommand(c.depResolver).Command())

	return cmd
}
