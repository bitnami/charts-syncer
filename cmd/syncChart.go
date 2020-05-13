package cmd

import (
	"os"

	"github.com/bitnami-labs/chart-repository-syncer/api"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/chart"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/config"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/helmcli"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/utils"
	"github.com/juju/errors"
	helm_repo "helm.sh/helm/v3/pkg/repo"
	"k8s.io/klog"

	"github.com/spf13/cobra"
)

var name, version string // used for flags
var syncAllVersions bool // used for flags

// syncChartCmd represents the syncChart command
var syncChartCmd = &cobra.Command{
	Use:   "syncChart",
	Short: "Syncronize a specific chart version between chart repos",
	Long: `Syncronize a specific chart version between chart repos:

	Example:
	$ c3tsyncer syncChart --name nginx --version 1.0.0 --config .c3tsyncer.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		syncChart()
	},
}

func init() {
	rootCmd.AddCommand(syncChartCmd)

	syncChartCmd.Flags().StringVarP(&name, "name", "", "", "Name of the chart to be synced")
	syncChartCmd.Flags().StringVarP(&version, "version", "", "", "Version of the chart to be synced")
	syncChartCmd.Flags().BoolVarP(&syncAllVersions, "all-versions", "", false, "Sync all versions of the provided chart")
	syncChartCmd.MarkFlagRequired("name")
}

func syncChart() {
	if !syncAllVersions && version == "" {
		klog.Fatal("Please specify a version to sync with --version VERSION or sync all versions with --all-versions")
	}

	// Load config file
	var syncConfig api.Config
	config.LoadConfig(&syncConfig)
	sourceRepo := syncConfig.Source
	targetRepo := syncConfig.Target

	// Parse index.yaml file to get all chart releases info
	indexFile, err := utils.DownloadIndex(sourceRepo)
	if err != nil {
		klog.Fatal(errors.ErrorStack(err))
	}
	sourceIndex, err := helm_repo.LoadIndexFile(indexFile)
	if err != nil {
		klog.Fatal(errors.ErrorStack(err))
	}
	defer os.Remove(indexFile)

	// Add target repo to helm CLI
	helmcli.AddRepoToHelm(targetRepo.Url, targetRepo.Auth)

	if syncAllVersions {
		if err := chart.SyncAllVersions(name, sourceRepo, targetRepo, false, sourceIndex, dryRun); err != nil {
			klog.Fatal(errors.ErrorStack(err))
		}
	} else {
		if chartExistsInSource, err := utils.ChartExistInIndex(name, version, sourceIndex); err == nil {
			if chartExistsInSource {
				if chartExistsInTarget, err := utils.ChartExistInTargetRepo(name, version, targetRepo); err == nil {
					if !chartExistsInTarget {
						if dryRun {
							klog.Infof("dry-run: Chart %s-%s pending to be synced", name, version)
						} else {
							if err := chart.Sync(name, version, sourceRepo, targetRepo, false); err != nil {
								klog.Fatal(errors.ErrorStack(err))
							}
						}
					} else {
						klog.Infof("Chart %s-%s already exists in target repo", name, version)
					}
				} else {
					klog.Fatal(errors.ErrorStack(err))
				}
			}
		} else {
			klog.Fatal(errors.ErrorStack(err))
		}
	}
}
