package cmd

import (
	"github.com/juju/errors"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/config"
	"github.com/bitnami-labs/charts-syncer/pkg/syncer"
)

var (
	syncFromDate string
)

var (
	syncExample = `
  # Synchronizes all charts defined in the configuration file
  charts-syncer sync

  # Synchronizes all charts defined in the configuration file from May 1st, 2020
  charts-syncer sync --from-date 2020-05-01`
)

func newSyncCmd() *cobra.Command {
	var c api.Config

	cmd := &cobra.Command{
		Use:     "sync",
		Short:   "Syncronizes two chart repositories",
		Example: syncExample,
		PreRunE: func(cmd *cobra.Command, args []string) error {
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
			if err := viper.ReadInConfig(); err != nil {
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
			s := syncer.NewSyncer(c.GetSource(), c.GetTarget(),
				// TODO(jdrios): Some backends may not support discovery
				syncer.WithAutoDiscovery(true),
				syncer.WithDryRun(rootDryRun),
				syncer.WithFromDate(syncFromDate),
			)

			return errors.Trace(s.Sync(c.GetCharts()...))
		},
	}

	cmd.Flags().StringVar(&syncFromDate, "from-date", "", "Date you want to synchronize charts from. Format: YYYY-MM-DD")

	return cmd
}
