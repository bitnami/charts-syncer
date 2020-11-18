package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/juju/errors"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/config"
	"github.com/bitnami-labs/charts-syncer/pkg/syncer"
)

const (
	defaultCfgFile = "charts-syncer.yaml"
)

var (
	dryRun   = flag.Bool("dry-run", false, "Only shows the charts pending to be synced without syncing them")
	cfgFile  = flag.String("config", "", fmt.Sprintf("Config file (default is $HOME/%s)", defaultCfgFile))
	fromDate = flag.String("from-date", "", "Date you want to synchronize charts from. Format: YYYY-MM-DD")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n\nFlags:\n", os.Args[0])
	flag.PrintDefaults()
}

func init() {
	// Initialize klog. Override some flag defaults so they are shown in the help func.
	klog.InitFlags(flag.CommandLine)
	flag.CommandLine.Lookup("alsologtostderr").Value.Set("true")
	flag.CommandLine.Lookup("v").Value.Set("2")

	flag.Usage = usage
}

// parseFlags reads in config file and ENV variables if set.
func parseFlags() error {
	// Use config file from the flag.
	if *cfgFile != "" {
		viper.SetConfigFile(*cfgFile)
		klog.Infof("Using config file: %q", *cfgFile)
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

func main() {
	defer klog.Flush()

	flag.Parse()

	if err := parseFlags(); err != nil {
		klog.Errorf("%+v", err)
		usage()
		os.Exit(1)
	}

	if err := mainE(); err != nil {
		klog.Errorf("%+v", err)
		usage()
		os.Exit(1)
	}
}

func mainE() error {
	// Load config file
	var c api.Config
	if err := config.Load(&c); err != nil {
		return errors.Trace(err)
	}
	if err := c.Validate(); err != nil {
		return errors.Trace(err)
	}

	s := syncer.NewSyncer(c.GetSource(), c.GetTarget(),
		// TODO(jdrios): Some backends may not support discovery
		syncer.WithAutoDiscovery(true),
		syncer.WithDryRun(*dryRun),
		syncer.WithFromDate(*fromDate),
	)

	return errors.Trace(s.Sync(c.GetCharts()...))
}
