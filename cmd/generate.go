package cmd

import (
	"github.com/juju/errors"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/internal/manifest"
	"github.com/bitnami-labs/charts-syncer/pkg/generator"
)

func initGenerateConfigFile() error {
	// Use config file from the flag.
	if rootGenerateConfig != "" {
		viper.SetConfigFile(rootGenerateConfig)
		klog.Infof("Using generator config file: %q", rootGenerateConfig)
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
	viper.SetConfigName(defaultGenerateCfgFile)
	viper.SetConfigType("yaml")
	klog.Infof("Looking for the default generator config %s", defaultGenerateCfgFile)
	return errors.Trace(viper.ReadInConfig())
}

func newGenerateCmd() *cobra.Command {
	m := api.Manifest{}
	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generate charts-syncer config",
		Example: syncExample,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := initGenerateConfigFile(); err != nil {
				return errors.Trace(err)
			}

			// Env variables bindings for viper
			if err := manifest.InitEnvBindings(); err != nil {
				return errors.Trace(err)
			}

			// Load manifest config file relying on viper to find it
			if err := manifest.Load(&m); err != nil {
				return errors.Trace(err)
			}

			if err := m.Validate(); err != nil {
				return errors.Trace(err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			genOptions := []generator.Option{
				generator.WithDryRun(rootDryRun),
			}

			g, err := generator.New(&m, genOptions...)
			if err != nil {
				return errors.Trace(err)
			}
			return errors.Trace(g.Generator())
		},
	}
	return cmd
}
