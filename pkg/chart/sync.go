package chart

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/juju/errors"
	"github.com/mkmik/multierror"
	"k8s.io/klog"

	"github.com/bitnami-labs/chart-repository-syncer/api"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/helmcli"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/repo"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/utils"

	helmRepo "helm.sh/helm/v3/pkg/repo"
)

// SyncAllVersions will sync all versions of a specific chart.
func SyncAllVersions(name string, sourceRepo *api.Repo, target *api.TargetRepo, syncDependencies bool, index *helmRepo.IndexFile, dryRun bool) error {
	var errs error
	// Create client for target repo
	tc, err := repo.NewClient(target.Repo)
	if err != nil {
		return fmt.Errorf("could not create a client for the source repo: %w", err)
	}
	if index.Entries[name] != nil {
		for i := range index.Entries[name] {
			if chartExists, err := tc.ChartExists(name, index.Entries[name][i].Metadata.Version, target.Repo); !chartExists && err == nil {
				if dryRun {
					klog.Infof("dry-run: Chart %s-%s pending to be synced", name, index.Entries[name][i].Metadata.Version)
				} else {
					if err := Sync(name, index.Entries[name][i].Metadata.Version, sourceRepo, target, syncDependencies); err != nil {
						errs = multierror.Append(errs, errors.Trace(err))
					}
				}
			}
		}
	} else {
		return errors.Errorf("Chart %s not found in source repo", name)
	}
	return errs
}

// Sync is the main function. It downloads, transform, package and publish a chart.
func Sync(name string, version string, sourceRepo *api.Repo, target *api.TargetRepo, syncDependencies bool) error {
	// Create temporary working directory
	tmpDir, err := ioutil.TempDir("", "c3tsyncer")
	defer os.RemoveAll(tmpDir)
	if err != nil {
		return errors.Annotatef(err, "Error creating temporary: %s", tmpDir)
	}
	srcDir := path.Join(tmpDir, "src")
	destDir := path.Join(tmpDir, "dest")
	for _, path := range []string{srcDir, destDir} {
		os.MkdirAll(path, 0775)
	}

	// Download chart
	filepath := srcDir + "/" + name + "-" + version + ".tgz"
	klog.V(4).Infof("srcDir: %s", srcDir)
	klog.V(4).Infof("destDir: %s", destDir)
	klog.V(4).Infof("chartPath: %s", filepath)
	// Create client for source repo
	sc, err := repo.NewClient(sourceRepo)
	if err != nil {
		return fmt.Errorf("could not create a client for the source repo: %w", err)
	}
	if err := sc.DownloadChart(filepath, name, version, sourceRepo); err != nil {
		return errors.Annotatef(err, "Error downloading chart %s-%s from source repo", name, version)
	}

	// Uncompress chart
	if err := utils.Untar(filepath, destDir); err != nil {
		return errors.Annotate(err, "Error found in Untar function")
	}

	// If chart has dependencies, check that they are already in the target repo.
	chartPath := path.Join(destDir, name)
	if _, err := os.Stat(path.Join(chartPath, "requirements.lock")); err == nil {
		if err := manageDependencies(chartPath, sourceRepo, target, syncDependencies); err != nil {
			return errors.Annotatef(err, "Error updating dependencies for chart %s-%s", name, version)
		}
	}

	// Update values.yaml with new registry and repository info
	valuesFile := path.Join(chartPath, "values.yaml")
	valuesProductionFile := path.Join(chartPath, "values-production.yaml")
	if _, err := os.Stat(valuesFile); err == nil {
		klog.V(3).Infof("Chart %s-%s has values.yaml file...", name, version)
		updateValuesFile(valuesFile, target)
	}
	if _, err := os.Stat(valuesProductionFile); err == nil {
		klog.V(3).Infof("Chart %s-%s has values-production.yaml...", name, version)
		updateValuesFile(valuesProductionFile, target)
	}

	// Package chart
	packagedChartPath, err := helmcli.Package(chartPath, name, version, destDir)
	if err != nil {
		return errors.Annotate(err, "Error taring chart")
	}

	// Create client for target repo
	tc, err := repo.NewClient(target.Repo)
	if err != nil {
		return fmt.Errorf("could not create a client for the source repo: %w", err)
	}
	if err := tc.PublishChart(packagedChartPath, target.Repo); err != nil {
		return errors.Annotatef(err, "Error publishing chart %s-%s to target repo", name, version)
	}
	klog.Infof("Chart %s-%s published successfully", name, version)

	return errors.Trace(err)
}
