package main

import (
	"github.com/spf13/cobra"
)

var version = "dev"

func versionHelp() string {
	return "Print the version number of charts-syncer"
}

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: versionHelp(),
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Printf("%s\n", version)
		},
	}

	return cmd
}
