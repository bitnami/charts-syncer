package cmd

import (
	"github.com/juju/errors"
	"github.com/spf13/cobra"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/config"
	"github.com/bitnami-labs/charts-syncer/pkg/syncer"
)

var (
	syncPendingFromDate string
)

var (
	syncPendingExample = `
  # Synchronizes all charts defined in the configuration file
  charts-syncer sync

  # Synchronizes all charts defined in the configuration file from May 1st, 2020
  charts-syncer sync --from-date 2020-05-01`
)

func newSyncPendingCmd() *cobra.Command {
	var c api.Config

	cmd := &cobra.Command{
		Use:     "sync-pending",
		Short:   "[EXPERIMENTAL] Synchronizes two chart repositories using an experimental feature",
		Hidden:  true,
		Example: syncPendingExample,
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
			s, err := syncer.New(c.GetSource(), c.GetTarget(),
				// TODO(jdrios): Some backends may not support discovery
				syncer.WithAutoDiscovery(true),
				syncer.WithDryRun(rootDryRun),
				syncer.WithFromDate(syncPendingFromDate),
			)
			if err != nil {
				return errors.Trace(err)
			}

			return errors.Trace(s.SyncPendingCharts(c.GetCharts()...))
		},
	}

	cmd.Flags().StringVar(&syncPendingFromDate, "from-date", "", "Date you want to synchronize charts from. Format: YYYY-MM-DD")

	return cmd
}
