package chart

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/juju/errors"
	"helm.sh/helm/v3/pkg/chart"
	helmRepo "helm.sh/helm/v3/pkg/repo"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/client/core"
	"github.com/bitnami-labs/charts-syncer/pkg/helmcli"
	"github.com/bitnami-labs/charts-syncer/pkg/utils"
)

// Sync is the main function. It downloads, transform, package and publish a chart.
func Sync(name string, version string, sourceRepo *api.Repo, target *api.TargetRepo, sourceIndex *helmRepo.IndexFile, targetIndex *helmRepo.IndexFile, syncDeps bool) error {
	// Create temporary working directory
	tmpDir, err := ioutil.TempDir("", "charts-syncer")
	if err != nil {
		return errors.Annotatef(err, "error creating temporary: %s", tmpDir)
	}
	defer os.RemoveAll(tmpDir)
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
	sc, err := core.NewClient(sourceRepo)
	if err != nil {
		return fmt.Errorf("could not create a client for the source repo: %w", err)
	}
	if err := sc.Fetch(filepath, name, version, sourceRepo, sourceIndex); err != nil {
		return errors.Annotatef(err, "error downloading chart %s-%s from source repo", name, version)
	}

	// Uncompress chart
	if err := utils.Untar(filepath, destDir); err != nil {
		return errors.Annotate(err, "error found in Untar function")
	}

	// If chart has dependencies, check that they are already in the target repo.
	chartPath := path.Join(destDir, name)
	if _, err := os.Stat(path.Join(chartPath, RequirementsLockFilename)); err == nil {
		if err := syncDependencies(chartPath, sourceRepo, target, sourceIndex, targetIndex, APIV1, syncDeps); err != nil {
			return errors.Annotatef(err, "error updating dependencies for chart %s-%s", name, version)
		}
	}
	if _, err := os.Stat(path.Join(chartPath, ChartLockFilename)); err == nil {
		if err := syncDependencies(chartPath, sourceRepo, target, sourceIndex, targetIndex, APIV2, syncDeps); err != nil {
			return errors.Annotatef(err, "error updating dependencies for chart %s-%s", name, version)
		}
	}

	// Update values.yaml with new registry and repository info
	valuesFile := path.Join(chartPath, ValuesFilename)
	valuesProductionFile := path.Join(chartPath, ValuesProductionFilename)
	if _, err := os.Stat(valuesFile); err == nil {
		klog.V(3).Infof("Chart %s-%s has values.yaml file...", name, version)
		if err := updateValuesFile(valuesFile, target); err != nil {
			return errors.Trace(err)
		}
	}
	if _, err := os.Stat(valuesProductionFile); err == nil {
		klog.V(3).Infof("Chart %s-%s has values-production.yaml...", name, version)
		if err := updateValuesFile(valuesProductionFile, target); err != nil {
			return errors.Trace(err)
		}
	}
	readmeFile := path.Join(chartPath, ReadmeFilename)
	if _, err := os.Stat(readmeFile); err == nil {
		klog.V(3).Infof("Chart %s-%s has README.md...", name, version)
		if err := updateReadmeFile(readmeFile, sourceRepo.Url, target.Repo.Url, name, target.RepoName); err != nil {
			return errors.Trace(err)
		}
	}

	// Package chart
	packagedChartPath, err := helmcli.Package(chartPath, name, version, destDir)
	if err != nil {
		return errors.Annotate(err, "error taring chart")
	}

	// Create client for target repo
	tc, err := core.NewClient(target.Repo)
	if err != nil {
		return fmt.Errorf("could not create a client for the source repo: %w", err)
	}
	if err := tc.Push(packagedChartPath, target.Repo); err != nil {
		return errors.Annotatef(err, "error publishing chart %s-%s to target repo", name, version)
	}
	// Add just synced chart to our local target index so other charts that may have this as dependency
	// know it is already synced in the target repository.
	targetIndex.Add(&chart.Metadata{Name: name, Version: version}, "", "", "")
	klog.Infof("Chart %s-%s published successfully", name, version)

	return errors.Trace(err)
}

// ChangeReferences changes the references of a chart tgz file from the source
// repo to the target repo and returns a new tgz file.
func ChangeReferences(outDir string, filepath string, name string, version string, srcRepo *api.SourceRepo, tgtRepo *api.TargetRepo) (string, error) {
	tmpDir, err := ioutil.TempDir("", "charts-syncer")
	if err != nil {
		return "", errors.Trace(err)
	}
	defer os.RemoveAll(tmpDir)
	if err := utils.Untar(filepath, tmpDir); err != nil {
		return "", errors.Trace(err)
	}
	chartPath := path.Join(tmpDir, name)

	// Update values*.yaml
	for _, f := range []string{
		path.Join(chartPath, ValuesFilename),
		path.Join(chartPath, ValuesProductionFilename),
	} {
		if ok, err := utils.FileExists(f); err != nil {
			return "", errors.Trace(err)
		} else if ok {
			klog.V(5).Infof("Processing %q file...", f)
			if err := updateValuesFile(f, tgtRepo); err != nil {
				return "", errors.Trace(err)
			}
		}
	}

	// Update README.md
	readmeFile := path.Join(chartPath, ReadmeFilename)
	if ok, err := utils.FileExists(readmeFile); err != nil {
		return "", errors.Trace(err)
	} else if ok {
		klog.V(5).Infof("Processing %q file...", readmeFile)
		if err := updateReadmeFile(
			readmeFile,
			srcRepo.GetRepo().GetUrl(),
			tgtRepo.GetRepo().GetUrl(),
			name,
			tgtRepo.GetRepoName(),
		); err != nil {
			return "", errors.Trace(err)
		}
	}

	// Package chart again
	//
	// TODO(jdrios): This relies on the helm client to package the repo. It
	// does not take into account that the target repo could be out of sync yet
	// (for example, if we uploaded a dependency of the chart being packaged a
	// few seconds ago).
	newTgz, err := helmcli.Package(chartPath, name, version, outDir)
	if err != nil {
		return "", errors.Trace(err)
	}

	return newTgz, nil
}
