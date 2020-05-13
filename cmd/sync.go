package cmd

import (
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

	helm_repo "helm.sh/helm/v3/pkg/repo"
)

var fromDate string // used for flags

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Syncronize all the charts from a source repository to a target repository",
	Long:  `Syncronize all the charts from a source repository to a target repository`,
	Run: func(cmd *cobra.Command, args []string) {
		sync()
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().StringVar(&fromDate, "from-date", "01/01/01", "Date you want to synchronize charts from. Format: MM/DD/YY")
	//syncCmd.Flags().DurationVar()
}

func sync() {
	var errs error
	// Load config file
	var syncConfig api.Config
	config.LoadConfig(&syncConfig)
	sourceRepo := syncConfig.Source
	targetRepo := syncConfig.Target

	// Create basic layout for date and parse flag to time type
	timeLayout := "01/02/06"
	dateThreshold, _ := time.Parse(timeLayout, fromDate)

	// Parse index.yaml file to get all chart releases info
	sourceIndexFile, err := utils.DownloadIndex(sourceRepo)
	if err != nil {
		klog.Fatalf("Error downloading index.yaml: %v ", err)
	}
	sourceIndex, err := helm_repo.LoadIndexFile(sourceIndexFile)
	if err != nil {
		klog.Fatalf("Error loading index.yaml: %v ", err)
	}
	defer os.Remove(sourceIndexFile)
	// Add target repo to helm CLI
	helmcli.AddRepoToHelm(targetRepo.Url, targetRepo.Auth)

	// Iterate over charts in source index
	for chartName := range sourceIndex.Entries {
		// Iterate over charts versions
		for i := range sourceIndex.Entries[chartName] {
			// Get version and publishing date
			chartVersion := sourceIndex.Entries[chartName][i].Metadata.Version
			publishingTime := sourceIndex.Entries[chartName][i].Created
			// Check if chart is already in target repo
			if chartExists, _ := utils.ChartExistInTargetRepo(chartName, chartVersion, targetRepo); !chartExists {
				if publishingTime.After(dateThreshold) {
					if dryRun {
						klog.Infof("dry-run: Chart %s-%s pending to be synced", chartName, chartVersion)
					} else {
						klog.Infof("Syncing %s-%s", chartName, chartVersion)
						if err := chart.Sync(chartName, chartVersion, sourceRepo, targetRepo, true); err != nil {
							errs = multierror.Append(errs, errors.Trace(err))
						}
					}
				}
			}
		}
	}
	if errs != nil {
		klog.Fatal(errors.ErrorStack(errs))
	}
}
