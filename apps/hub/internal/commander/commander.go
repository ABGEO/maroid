// Package commander defines the Commander interface for CLI commands.
package commander

import "github.com/spf13/cobra"

// Commander represents a CLI command with its associated functionality.
type Commander interface {
	// Command initializes and returns the Cobra command.
	Command() *cobra.Command
}
