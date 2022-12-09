package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	defaultCfgFile         = "charts-syncer.yaml"
	defaultGenerateCfgFile = "charts-generator-syncer.yaml"
)

var (
	rootUsage = `charts-syncer is a tool to synchronize chart repositories from a source repository to a target repository

Find more information at: https://github.com/bitnami-labs/charts-syncer`

	rootConfig         string
	rootGenerateConfig string
	rootDryRun         bool
	rootInsecure       bool
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "charts-syncer",
		Short: "tool to synchronize helm chart repositories",
		Long:  rootUsage,
		// Do not show the Usage page on every raised error
		SilenceUsage: true,
	}

	cmd.PersistentFlags().BoolVar(&rootDryRun, "dry-run", false, "Only shows the charts pending to be synced without syncing them")
	cmd.PersistentFlags().StringVarP(&rootConfig, "config", "c", "", fmt.Sprintf("Config file. Defaults to ./%s or $HOME/%s)", defaultCfgFile, defaultCfgFile))
	cmd.PersistentFlags().StringVarP(&rootGenerateConfig, "generate-config", "g", "", fmt.Sprintf("Generate Config file. Defaults to ./%s or $HOME/%s)", defaultGenerateCfgFile, defaultGenerateCfgFile))
	cmd.PersistentFlags().BoolVar(&rootInsecure, "insecure", false, "Allow insecure SSL connections")

	// Add subcommands
	cmd.AddCommand(
		newSyncCmd(),
		newGenerateCmd(),
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
