package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var Version = "dev"

func versionHelp() string {
	return fmt.Sprint("Print the version number of charts-syncer")
}

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Short:  versionHelp(),
		Long: versionHelp(),
		Example: versionHelp(),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("charts-syncer version: %s\n", Version)
		},
	}

	return cmd
}
