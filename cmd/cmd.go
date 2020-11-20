package cmd

import (
	"github.com/spf13/cobra"
)

// New creates a new cobra Command
func New() *cobra.Command {
	return newRootCmd()
}
