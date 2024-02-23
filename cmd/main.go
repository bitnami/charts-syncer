// Package main serves as an entrypoint to the chart-syncer command line tool
package main

import (
	"os"

	"k8s.io/klog"
)

func main() {
	defer klog.Flush()

	command := newRootCmd()

	if err := command.Execute(); err != nil {
		// No need to print the errors, Cobra does it for us already since SilenceErrors = false
		os.Exit(1)
	}
}
