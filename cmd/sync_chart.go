package main

import (
	"fmt"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/chart"
	"github.com/bitnami-labs/charts-syncer/pkg/config"
	"github.com/bitnami-labs/charts-syncer/pkg/helmcli"
	"github.com/bitnami-labs/charts-syncer/pkg/repo"
	"github.com/bitnami-labs/charts-syncer/pkg/utils"
	"github.com/juju/errors"
	"k8s.io/klog"

	"github.com/spf13/cobra"
)

var (
	name            string
	version         string
	syncAllVersions bool
	syncDeps        bool
)

func newSyncChart() *cobra.Command {
	var specFile string

	cmd := &cobra.Command{
		Use:   "syncChart",
		Short: "Syncronize a specific chart version between chart repos",
		Long: `Syncronize a specific chart version between chart repos:

	Example:
	$ charts-syncer syncChart --name nginx --version 1.0.0 --config .charts-syncer.yaml`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if !syncAllVersions && version == "" {
				return errors.Trace(fmt.Errorf(`missing "--version" flag: Please use either "--version VERSION" or "--all-versions"`))
			}
			if syncAllVersions && version != "" {
				klog.Warningf(`Ignoring "--version" flag: "--all-versions" is set`)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.Trace(syncChart())
		},
	}

	f := cmd.Flags()
	f.StringVar(&specFile, "spec", "", "spec file")
	f.StringVarP(&name, "name", "", "", "Name of the chart to be synced")
	f.StringVarP(&version, "version", "", "", "Version of the chart to be synced")
	f.BoolVarP(&syncAllVersions, "all-versions", "", false, "Sync all versions of the provided chart")
	f.BoolVarP(&syncDeps, "sync-deps", "", false, "Also sync dependencies of the chart if they are not found in the target repository")
	cmd.MarkFlagRequired("name")

	return cmd
}

func syncChart() error {
	// Load config file
	var syncConfig api.Config
	if err := config.Load(&syncConfig); err != nil {
		return errors.Trace(err)
	}
	if err := syncConfig.Validate(); err != nil {
		return errors.Trace(err)
	}
	source := syncConfig.Source
	target := syncConfig.Target

	// Load index.yaml info into index object
	sourceIndex, err := utils.LoadIndexFromRepo(source.Repo)
	if err != nil {
		return errors.Trace(fmt.Errorf("error loading index.yaml: %w", err))
	}

	// Add target repo to helm CLI
	helmcli.AddRepoToHelm(target.Repo.Url, target.Repo.Auth)
	// Create client for target repo
	tc, err := repo.NewClient(target.Repo)
	if err != nil {
		return fmt.Errorf("could not create a client for the source repo: %w", err)
	}

	if syncAllVersions {
		if err := chart.SyncAllVersions(name, source.Repo, target, syncDeps, sourceIndex, dryRun); err != nil {
			return errors.Trace(err)
		}
	} else {
		srcExists, err := utils.ChartExistInIndex(name, version, sourceIndex)
		if err != nil {
			return errors.Trace(err)
		}
		if !srcExists {
			return errors.Errorf("chart %s-%s not found in source index.yaml", name, version)
		}
		targetExists, err := tc.ChartExists(name, version, target.Repo)
		if err != nil {
			return errors.Trace(err)
		}
		if targetExists {
			klog.Infof("chart %s-%s already exists in target repo", name, version)
			return nil
		}
		if dryRun {
			klog.Infof("dry-run: Chart %s-%s pending to be synced", name, version)
			return nil
		}
		if err := chart.Sync(name, version, source.Repo, target, sourceIndex, syncDeps); err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}
