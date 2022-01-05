package cmd

import (
	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/internal/config"
	"github.com/bitnami-labs/charts-syncer/pkg/syncer"
	"github.com/juju/errors"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/klog"
)

var (
	syncFromDate          string
	syncWorkdir           string
	syncSkipDependencies  bool
	syncLatestVersionOnly bool
)

var (
	syncExample = `
  # Synchronizes all charts defined in the configuration file
  charts-syncer sync

  # Synchronizes all charts defined in the configuration file from May 1st, 2020
  charts-syncer sync --from-date 2020-05-01`
)

func initConfigFile() error {
	// Use config file from the flag.
	if rootConfig != "" {
		viper.SetConfigFile(rootConfig)
		klog.Infof("Using config file: %q", rootConfig)
		return errors.Trace(viper.ReadInConfig())
	}

	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		return errors.Trace(err)
	}

	// Search config in home directory with name ".charts-syncer" (without extension).
	viper.AddConfigPath(home)
	viper.AddConfigPath(".")
	viper.SetConfigName(defaultCfgFile)
	viper.SetConfigType("yaml")
	klog.Infof("Looking for the default config %s", defaultCfgFile)
	return errors.Trace(viper.ReadInConfig())
}

func newSyncCmd() *cobra.Command {
	var c api.Config

	cmd := &cobra.Command{
		Use:     "sync",
		Short:   "Synchronizes two chart repositories",
		Example: syncExample,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := initConfigFile(); err != nil {
				return errors.Trace(err)
			}

			// Load config file relying on viper to find it
			if err := config.Load(&c); err != nil {
				return errors.Trace(err)
			}
			if err := c.Validate(); err != nil {
				return errors.Trace(err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			syncerOptions := []syncer.Option{
				// TODO(jdrios): Some backends may not support discovery
				syncer.WithAutoDiscovery(true),
				syncer.WithDryRun(rootDryRun),
				syncer.WithFromDate(syncFromDate),
				syncer.WithWorkdir(syncWorkdir),
				syncer.WithInsecure(rootInsecure),
				syncer.WithContainerImageRelocation(c.RelocateContainerImages),
				syncer.WithSkipDependencies(syncSkipDependencies),
				syncer.WithLatestVersionOnly(syncLatestVersionOnly),
			}
			s, err := syncer.New(c.GetSource(), c.GetTarget(), syncerOptions...)
			if err != nil {
				return errors.Trace(err)
			}

			return errors.Trace(s.SyncPendingCharts(c.GetCharts()...))
		},
	}

	cmd.Flags().StringVar(&syncFromDate, "from-date", "", "Date you want to synchronize charts from. Format: YYYY-MM-DD")
	cmd.Flags().StringVar(&syncWorkdir, "workdir", syncer.DefaultWorkdir(), "Working directory")
	cmd.Flags().BoolVar(&syncSkipDependencies, "skip-dependencies", false, "Skip syncing chart dependencies")
	cmd.Flags().BoolVar(&syncLatestVersionOnly, "latest-version-only", false, "Sync only latest version of each chart")

	return cmd
}
