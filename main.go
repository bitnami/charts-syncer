package main

import (
	"flag"
	"os"

	"k8s.io/klog"

	"github.com/bitnami/charts-syncer/cmd"
)

func main() {
	defer klog.Flush()
	// Klog flags
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)

	// Override some flag defaults so they are shown in the help func.
	klog.InitFlags(klogFlags)
	_ = klogFlags.Lookup("alsologtostderr").Value.Set("true")
	_ = klogFlags.Lookup("logtostderr").Value.Set("true")

	command := cmd.New()

	// Register klog flags so they appear on the command's help
	command.PersistentFlags().AddGoFlagSet(klogFlags)

	if err := command.Execute(); err != nil {
		// No need to print the errors, Cobra does it for us already since SilenceErrors = false
		os.Exit(1)
	}
}
