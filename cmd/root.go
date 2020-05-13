package cmd

import (
	"flag"
	"os"

	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"k8s.io/klog"
)

var cfgFile string
var dryRun bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "c3tsyncer",
	Short: "tool to syncronize helm chart repositories",
	Long: `c3tsyncer (chart-repository-syncer) is a tool to syncronize
chart repositories from a source repository to a target repository.

You can sync a single chart or the whole repository`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		klog.Fatal(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.c3tsyncer.yaml)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "only shows the charts pending to be synced without syncing them")
	rootCmd.MarkPersistentFlagRequired("config")

	// Initialize klog. Override some flag defaults so they are shown in the help func.
	klog.InitFlags(flag.CommandLine)
	flag.CommandLine.Lookup("alsologtostderr").Value.Set("true")
	flag.CommandLine.Lookup("v").Value.Set("2")
	defer klog.Flush()
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
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
		// Search config in home directory with name ".c3tsyncer" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".c3tsyncer")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		klog.Info("Using config file:", viper.ConfigFileUsed())
	}
}
