package chart

import (
	"fmt"
	"io/ioutil"
	"path"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/client/core"
	"github.com/bitnami-labs/charts-syncer/pkg/helmcli"
	"github.com/juju/errors"
	"github.com/mkmik/multierror"
	helmChart "helm.sh/helm/v3/pkg/chart"
	helmRepo "helm.sh/helm/v3/pkg/repo"
	"k8s.io/klog"
	"sigs.k8s.io/yaml"
)

// dependencies is the list of dependencies of a chart
type dependencies struct {
	Dependencies []*helmChart.Dependency `json:"dependencies"`
}

// syncDependencies takes care of updating dependencies to correct version and sync to target repo if necesary.
func syncDependencies(chartPath string, sourceRepo *api.Repo, target *api.TargetRepo, sourceIndex *helmRepo.IndexFile, targetIndex *helmRepo.IndexFile, apiVersion string, syncDeps bool) error {
	klog.V(3).Info("Chart has dependencies...")
	var errs error
	var missingDependencies = false

	lockFilePath, err := lockFilePath(chartPath, apiVersion)
	if err != nil {
		return errors.Trace(err)
	}
	lockContent, err := ioutil.ReadFile(lockFilePath)
	if err != nil {
		return errors.Trace(err)
	}
	lock := &helmChart.Lock{}
	err = yaml.Unmarshal(lockContent, lock)
	if err != nil {
		return errors.Annotatef(err, "error unmarshaling %s file", lockFilePath)
	}

	tc, err := core.NewClient(target.Repo)
	if err != nil {
		return fmt.Errorf("could not create a client for the source repo: %w", err)
	}

	if err != nil {
		return errors.Trace(err)
	}
	for _, lockDep := range lock.Dependencies {
		depName := lockDep.Name
		depVersion := lockDep.Version
		depRepository := lockDep.Repository
		if depRepository != sourceRepo.Url {
			continue
		}
		if chartExists, _ := tc.ChartExists(depName, depVersion, targetIndex); chartExists {
			klog.V(3).Infof("Dependency %s-%s already synced", depName, depVersion)
			continue
		}
		if !syncDeps {
			missingDependencies = true
			errs = multierror.Append(errs, errors.Errorf("please sync %s-%s dependency first", depName, depVersion))
			continue
		}
		klog.Infof("Dependency %s-%s not synced yet. Syncing now", depName, depVersion)
		if err := Sync(depName, depVersion, sourceRepo, target, sourceIndex, targetIndex, true); err != nil {
			return errors.Trace(err)
		}
		chartExists, err := tc.ChartExists(depName, depVersion, targetIndex)
		if err != nil {
			return errors.Trace(err)
		}
		if !chartExists {
			return errors.Errorf("dependency %s-%s not available yet", depName, depVersion)
		}
		klog.Infof("Dependency %s-%s synced", depName, depVersion)
	}

	if !missingDependencies {
		klog.V(3).Info("Updating dependencies file...")
		switch apiVersion {
		case APIV1:
			if err := updateRequirementsFile(chartPath, lock, sourceRepo, target); err != nil {
				return errors.Trace(err)
			}
		case APIV2:
			if err := updateChartMetadataFile(chartPath, lock, sourceRepo, target); err != nil {
				return errors.Trace(err)
			}
		default:
			return errors.Errorf("unrecognised apiVersion %s", apiVersion)
		}
		if err := helmcli.UpdateDependencies(chartPath); err != nil {
			return errors.Trace(err)
		}
	}
	return errs
}

// updateChartMetadataFile updates the dependencies in Chart.yaml
// For helm v3 dependency management
func updateChartMetadataFile(chartPath string, lock *helmChart.Lock, sourceRepo *api.Repo, target *api.TargetRepo) error {
	chartFile := path.Join(chartPath, ChartFilename)
	chart, err := ioutil.ReadFile(chartFile)
	if err != nil {
		return errors.Trace(err)
	}
	chartMetadata := &helmChart.Metadata{}
	err = yaml.Unmarshal(chart, chartMetadata)
	if err != nil {
		return errors.Annotatef(err, "error unmarshaling %s file", chartFile)
	}
	for _, dep := range chartMetadata.Dependencies {
		// Specify the exact dependencies versions used in the original Chart.lock file
		// so when running helm dep up we get the same versions resolved.
		dep.Version = findDepByName(lock.Dependencies, dep.Name).Version
		// Maybe there are dependencies from other chart repos. In this case we don't want to replace
		// the repository.
		// For example, old charts pointing to helm/charts repo
		if dep.Repository == sourceRepo.Url {
			dep.Repository = target.Repo.Url
		}
	}
	// Write updated requirements yamls file
	writeChartMetadataFile(chartPath, chartMetadata)
	return nil
}

// updateRequirementsFile returns the full list of dependencies and the list of missing dependencies.
// For helm v2 dependency management
func updateRequirementsFile(chartPath string, lock *helmChart.Lock, sourceRepo *api.Repo, target *api.TargetRepo) error {
	requirementsFile := path.Join(chartPath, RequirementsFilename)
	requirements, err := ioutil.ReadFile(requirementsFile)
	if err != nil {
		return errors.Trace(err)
	}

	deps := &dependencies{}
	err = yaml.Unmarshal(requirements, deps)
	if err != nil {
		return errors.Annotatef(err, "error unmarshaling %s file", requirementsFile)
	}
	for _, dep := range deps.Dependencies {
		// Specify the exact dependencies versions used in the original requirements.lock file
		// so when running helm dep up we get the same versions resolved.
		dep.Version = findDepByName(lock.Dependencies, dep.Name).Version
		// Maybe there are dependencies from other chart repos. In this case we don't want to replace
		// the repository.
		// For example, old charts pointing to helm/charts repo
		if dep.Repository == sourceRepo.Url {
			dep.Repository = target.Repo.Url
		}
	}
	// Write updated requirements yamls file
	writeRequirementsFile(chartPath, deps)
	return nil
}

// findDepByName returns the dependency that matches a provided name from a list of dependencies.
func findDepByName(dependencies []*helmChart.Dependency, name string) *helmChart.Dependency {
	for _, dep := range dependencies {
		if dep.Name == name {
			return dep
		}
	}
	return nil
}

// writeRequirementsFile writes a requirements.yaml file to disk.
// For helm v2 dependency management
func writeRequirementsFile(chartPath string, deps *dependencies) error {
	data, err := yaml.Marshal(deps)
	if err != nil {
		return err
	}
	requirementsFileName := RequirementsFilename
	dest := path.Join(chartPath, requirementsFileName)
	return ioutil.WriteFile(dest, data, 0644)
}

// writeChartMetadataFile writes a Chart.yaml file to disk.
// For helm v3 dependency management
func writeChartMetadataFile(chartPath string, chartMetadata *helmChart.Metadata) error {
	data, err := yaml.Marshal(chartMetadata)
	if err != nil {
		return err
	}
	chartMetadataFileName := ChartFilename
	dest := path.Join(chartPath, chartMetadataFileName)
	return ioutil.WriteFile(dest, data, 0644)
}

// lockFilePath returns the path to the lock file according to provided Api version
func lockFilePath(chartPath, apiVersion string) (string, error) {
	switch apiVersion {
	case APIV1:
		return path.Join(chartPath, RequirementsLockFilename), nil
	case APIV2:
		return path.Join(chartPath, ChartLockFilename), nil
	default:
		return "", errors.Errorf("unrecognised apiVersion %q", apiVersion)
	}
}
