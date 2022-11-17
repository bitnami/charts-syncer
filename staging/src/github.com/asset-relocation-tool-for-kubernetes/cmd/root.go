// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

const appName = "relok8s"

var rootCmd = &cobra.Command{
	Use: appName,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// rootCmd may return an error, but Cobra is already displaying it
		// so here we just swallow but still exit with an error code
		os.Exit(1)
	}
}
