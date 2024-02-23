// Package cmd implements the command line for the chart-syncer tool
package main

import (
	"flag"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/klog"
)

const (
	defaultCfgFile = "charts-syncer.yaml"
)

var (
	rootUsage = `charts-syncer is a tool to synchronize chart repositories from a source repository to a target repository

Find more information at: https://github.com/bitnami/charts-syncer`

	rootConfig   string
	rootDryRun   bool
	rootInsecure bool
)

func newRootCmd() *cobra.Command {
	// Klog flags
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)

	// Override some flag defaults so they are shown in the help func.
	klog.InitFlags(klogFlags)
	_ = klogFlags.Lookup("alsologtostderr").Value.Set("true")
	_ = klogFlags.Lookup("logtostderr").Value.Set("true")

	cmd := &cobra.Command{
		Use:   "charts-syncer",
		Short: "tool to synchronize helm chart repositories",
		Long:  rootUsage,
		// Do not show the Usage page on every raised error
		SilenceUsage: true,
	}
	cmd.PersistentFlags().BoolVar(&rootDryRun, "dry-run", false, "Only shows the charts pending to be synced without syncing them")
	cmd.PersistentFlags().StringVarP(&rootConfig, "config", "c", "", fmt.Sprintf("Config file. Defaults to ./%s or $HOME/%s)", defaultCfgFile, defaultCfgFile))
	cmd.PersistentFlags().BoolVar(&rootInsecure, "insecure", false, "Allow insecure SSL connections")

	// Register klog flags so they appear on the command's help
	cmd.PersistentFlags().AddGoFlagSet(klogFlags)

	// Add subcommands
	cmd.AddCommand(
		newSyncCmd(),
		newVersionCmd(),
	)

	// Workaround to disable help subcommand
	// https://github.com/spf13/cobra/issues/587
	cmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})

	return cmd
}
