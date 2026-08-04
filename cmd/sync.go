package main

import (
	goerrors "errors"
	"time"

	apiv1 "github.com/bitnami/charts-syncer/gen/proto/v1"
	"github.com/bitnami/charts-syncer/internal/config"
	klogLogger "github.com/bitnami/charts-syncer/internal/log"
	"github.com/bitnami/charts-syncer/internal/utils"
	"github.com/juju/errors"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	log "github.com/vmware-labs/distribution-tooling-for-helm/pkg/dtlog"
	"github.com/vmware-labs/distribution-tooling-for-helm/pkg/dtlog/pterm"
	"k8s.io/klog"
)

var (
	syncWorkdir           string
	syncFromDate          string
	syncLatestVersionOnly bool
	usePlainHTTP          bool
	usePlainLog           bool
	syncTimeout           time.Duration
)

var syncExample = `
  # Synchronizes charts and containers defined in the configuration file
  charts-syncer sync

  # Synchronizes only the latest version of each chart/container
  charts-syncer sync --latest-version-only`

func newSyncCmd() *cobra.Command {
	var c apiv1.Config

	cmd := &cobra.Command{
		Use:           "sync",
		Short:         "Sync charts and containers defined in the configuration file",
		Example:       syncExample,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			if !usePlainLog {
				_ = cmd.Flags().Lookup("alsologtostderr").Value.Set("false")
				_ = cmd.Flags().Lookup("logtostderr").Value.Set("false")
			}
			if err := initConfigFile(); err != nil {
				return errors.Trace(err)
			}
			if err := config.InitEnvBindings(); err != nil {
				return errors.Trace(err)
			}
			if err := config.Load(&c); err != nil {
				return errors.Trace(err)
			}
			if hasChartSource(&c) && hasChartTarget(&c) {
				if err := config.ValidateChartConfig(&c); err != nil {
					return errors.Trace(err)
				}
			}
			if hasContainerSource(&c) && hasContainerTarget(&c) {
				if err := config.ValidateContainerConfig(&c); err != nil {
					return errors.Trace(err)
				}
			}
			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			var parentLog log.SectionLogger
			if usePlainLog {
				parentLog = klogLogger.NewKlogSectionLogger()
			} else {
				parentLog = pterm.NewSectionLogger()
			}
			return runSync(parentLog, &c)
		},
	}

	cmd.Flags().StringVar(&syncWorkdir, "workdir", config.DefaultWorkdir(), "Working directory")
	cmd.Flags().BoolVar(&syncLatestVersionOnly, "latest-version-only", false, "Sync only latest version of each chart/container")
	cmd.Flags().BoolVar(&usePlainHTTP, "use-plain-http", false, "Use plain HTTP instead of HTTPS")
	cmd.Flags().BoolVar(&usePlainLog, "use-plain-log", false, "Use plain klog instead of the pretty logging")
	cmd.Flags().StringVar(&syncFromDate, "from-date", "", "Date you want to synchronize charts from. Format: YYYY-MM-DD")
	cmd.Flags().DurationVar(&syncTimeout, "timeout", 0, "Timeout for chart and container syncing operations (e.g. 5m, 300s)")

	return cmd
}

func hasChartSource(c *apiv1.Config) bool {
	return c.GetSource().GetRepo().GetUrl() != "" || (c.GetSource().GetRepo().GetPath() != "" && c.GetSource().GetRepo().GetKind() == apiv1.Kind_LOCAL)
}

func hasChartTarget(c *apiv1.Config) bool {
	return c.GetTarget().GetRepo().GetUrl() != "" || (c.GetTarget().GetRepo().GetPath() != "" && c.GetTarget().GetRepo().GetKind() == apiv1.Kind_LOCAL)
}

func hasContainerSource(c *apiv1.Config) bool {
	return c.GetSource().GetContainers().GetUrl() != "" || (c.GetSource().GetRepo().GetPath() != "" && c.GetSource().GetRepo().GetKind() == apiv1.Kind_LOCAL)
}

func hasContainerTarget(c *apiv1.Config) bool {
	return c.GetTarget().GetContainers().GetUrl() != "" || (c.GetTarget().GetRepo().GetPath() != "" && c.GetTarget().GetRepo().GetKind() == apiv1.Kind_LOCAL)
}

func runSync(parentLog log.SectionLogger, c *apiv1.Config) error {
	var errs error

	if syncTimeout > 0 {
		klog.V(3).Infof("Setting global HTTP client timeout to %v", syncTimeout)
		utils.DefaultClient.Timeout = syncTimeout
		utils.InsecureClient.Timeout = syncTimeout
	}

	if hasChartSource(c) && hasChartTarget(c) {
		if err := runChartsSyncer(parentLog, c); err != nil {
			errs = goerrors.Join(errs, err)
			// container sync still runs; chart errors reported at the end
			klog.Warningf("Chart sync failed, continuing with container sync: %v", err)
		}
	}
	if hasContainerSource(c) && hasContainerTarget(c) && len(c.GetContainers()) > 0 {
		if err := runContainersSyncer(parentLog, c); err != nil {
			errs = goerrors.Join(errs, err)
		}
	}
	return errs
}

func initConfigFile() error {
	if rootConfig != "" {
		viper.SetConfigFile(rootConfig)
		klog.Infof("Using config file: %q", rootConfig)
		return errors.Trace(viper.ReadInConfig())
	}

	home, err := homedir.Dir()
	if err != nil {
		return errors.Trace(err)
	}

	viper.AddConfigPath(home)
	viper.AddConfigPath(".")
	viper.SetConfigName(defaultCfgFile)
	viper.SetConfigType("yaml")
	klog.Infof("Looking for the default config %s", defaultCfgFile)
	return errors.Trace(viper.ReadInConfig())
}
