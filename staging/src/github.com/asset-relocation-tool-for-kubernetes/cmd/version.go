// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.SetOut(os.Stdout)
}

var Version = "dev"

func versionHelp() string {
	return fmt.Sprintf("Print the version number of %s", appName)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: versionHelp(),
	Long:  versionHelp(),
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("%s version: %s\n", appName, Version)
	},
}
