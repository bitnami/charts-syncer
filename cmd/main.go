package main

import (
	"flag"
	"os"

	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"k8s.io/klog"
)

var (
	cfgFile string
	dryRun  bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "charts-syncer",
	Short: "tool to syncronize helm chart repositories",
	Long: `charts-syncer is a tool to syncronize
chart repositories from a source repository to a target repository.

You can sync a single chart or the whole repository`,
}

func init() {
	cobra.OnInitialize(initConfig)

	// Initialize klog. Override some flag defaults so they are shown in the help func.
	klog.InitFlags(flag.CommandLine)
	flag.CommandLine.Lookup("alsologtostderr").Value.Set("true")
	flag.CommandLine.Lookup("v").Value.Set("2")

	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.charts-syncer.yaml)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "only shows the charts pending to be synced without syncing them")

	flag.Usage = func() {
		if err := rootCmd.Help(); err != nil {
			klog.Fatalf("%+v", err)
		}
	}

	rootCmd.AddCommand(newSync())
	rootCmd.AddCommand(newSyncChart())
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			klog.Info(err)
			os.Exit(1)
		}
		// Search config in home directory with name ".charts-syncer" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".charts-syncer")
	}
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		klog.Fatalf("error reading config file: %+v", err)
	} else {
		klog.Info("Using config file:", viper.ConfigFileUsed())
	}
}

func main() {
	defer klog.Flush()
	if err := rootCmd.Execute(); err != nil {
		klog.Fatalf("%+v", err)
	}
}
