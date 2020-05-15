package main

import (
	"fmt"
	"os"
	"time"

	"github.com/bitnami-labs/chart-repository-syncer/api"
	"github.com/juju/errors"
	"github.com/mkmik/multierror"

	"github.com/bitnami-labs/chart-repository-syncer/pkg/chart"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/config"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/helmcli"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/utils"

	"github.com/spf13/cobra"
	"k8s.io/klog"

	helmRepo "helm.sh/helm/v3/pkg/repo"
)

var (
	fromDate string // used for flags
)

func newSync() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Syncronize all the charts from a source repository to a target repository",
		Long:  `Syncronize all the charts from a source repository to a target repository`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.Trace(sync())
		},
	}

	f := cmd.Flags()
	f.StringVar(&fromDate, "from-date", "2001-01-01", "Date you want to synchronize charts from. Format: YYYY/MM/DD")

	return cmd
}

func sync() error {
	var errs error
	// Load config file
	var syncConfig api.Config
	if err := config.LoadConfig(&syncConfig); err != nil {
		return errors.Trace(fmt.Errorf("Error loading config file"))
	}
	source := syncConfig.Source
	target := syncConfig.Target

	// Create basic layout for date and parse flag to time type
	timeLayoutISO := "2006-01-02"
	dateThreshold, err := time.Parse(timeLayoutISO, fromDate)
	if err != nil {
		return errors.Trace(fmt.Errorf("Error parsing date: %w", err))
	}

	// Parse index.yaml file to get all chart releases info
	sourceIndexFile, err := utils.DownloadIndex(source.Repo)
	defer os.Remove(sourceIndexFile)
	if err != nil {
		return errors.Trace(fmt.Errorf("Error downloading index.yaml: %w", err))
	}
	sourceIndex, err := helmRepo.LoadIndexFile(sourceIndexFile)
	if err != nil {
		return errors.Trace(fmt.Errorf("Error loading index.yaml: %w", err))
	}
	// Add target repo to helm CLI
	helmcli.AddRepoToHelm(target.Repo.Url, target.Repo.Auth)

	// Iterate over charts in source index
	for chartName := range sourceIndex.Entries {
		// Iterate over charts versions
		for i := range sourceIndex.Entries[chartName] {
			// Get version and publishing date
			chartVersion := sourceIndex.Entries[chartName][i].Metadata.Version
			publishingTime := sourceIndex.Entries[chartName][i].Created
			// If publishing date before date threshold skip
			if publishingTime.Before(dateThreshold) {
				continue
			}
			// If chart-version already in target repo skip
			if chartExists, _ := utils.ChartExistInTargetRepo(chartName, chartVersion, target.Repo); chartExists {
				continue
			}
			// If dry-run mode enabled skip
			if dryRun {
				klog.Infof("dry-run: Chart %s-%s pending to be synced", chartName, chartVersion)
				continue
			}
			klog.Infof("Syncing %s-%s", chartName, chartVersion)
			if err := chart.Sync(chartName, chartVersion, source.Repo, target, true); err != nil {
				errs = multierror.Append(errs, errors.Trace(err))
			}
		}
	}
	return errs
}
