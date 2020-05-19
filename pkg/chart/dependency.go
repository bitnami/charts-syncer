package chart

import (
	"fmt"
	"io/ioutil"
	"path"

	"github.com/bitnami-labs/chart-repository-syncer/api"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/helmcli"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/repo"
	"github.com/juju/errors"
	"github.com/mkmik/multierror"
	"gopkg.in/yaml.v2"
	helmChart "helm.sh/helm/v3/pkg/chart"
	"k8s.io/klog"
)

// dependencies is the list of dependencies of a chart
type dependencies struct {
	Dependencies []*helmChart.Dependency `json:"dependencies"`
}

// manageDependencies takes care of updating dependencies to correct version and sync to target repo if necesary.
func manageDependencies(chartPath string, sourceRepo *api.Repo, target *api.TargetRepo, syncDependencies bool) error {
	var errs error
	var missingDependencies = false

	chartDependenciesMap := make(map[string]string)

	requirementsLockFile := path.Join(chartPath, "requirements.lock")
	requirementsFile := path.Join(chartPath, "requirements.yaml")
	klog.V(3).Info("Chart has dependencies...")

	requirementsLock, err := ioutil.ReadFile(requirementsLockFile)
	if err != nil {
		return errors.Annotatef(err, "Error reading %s file", requirementsLockFile)
	}

	lock := &helmChart.Lock{}
	err = yaml.Unmarshal(requirementsLock, lock)
	if err != nil {
		return errors.Annotatef(err, "Error unmarshaling %s file", requirementsLockFile)
	}
	// Create client for target repo
	tc, err := repo.NewClient(target.Repo)
	if err != nil {
		return fmt.Errorf("could not create a client for the source repo: %w", err)
	}

	// Check list of dependencies and missing dependencies
	dependenciesMap, missingDependenciesMap, err := getDependencies(lock, sourceRepo, target, tc)
	if err != nil {
		return errors.Trace(err)
	}

	// Sync missing dependencies
	if missingDependenciesMap != nil {
		if syncDependencies {
			if err := syncMissingDependencies(missingDependenciesMap, sourceRepo, target, tc); err != nil {
				return errors.Trace(err)
			}
		} else {
			for depName := range missingDependenciesMap {
				depVersion := missingDependenciesMap[depName]
				errs = multierror.Append(errs, errors.Errorf("Please sync %s-%s dependency first", depName, depVersion))
				missingDependencies = true
			}
		}
	}

	if !missingDependencies {
		klog.V(3).Info("Updating requirements.yaml file...")
		if err := updateRequirementsFile(chartPath, dependenciesMap, sourceRepo, target); err != nil {
			return errors.Trace(err)
		}
		if err := helmcli.UpdateDependencies(chartPath); err != nil {
			return errors.Trace(err)
		}
	}
	return errs
}

// getMissingDependencies returns the list of dependencies not synced yet.
func getDependencies(lock *helmChart.Lock, sourceRepo *api.Repo, target *api.TargetRepo, tc repo.ChartRepoAPI) (map[string]string, map[string]string, error) {
	dependenciesMap := make(map[string]string)
	missingDependenciesMap := make(map[string]string)
	for i := range lock.Dependencies {
		// Check if chart exists in target repo
		depName := lock.Dependencies[i].Name
		depVersion := lock.Dependencies[i].Version
		depRepository := lock.Dependencies[i].Repository
		chartDependenciesMap[depName] = depVersion
		// Only sync dependencies retrieved from source repo.
		if depRepository == sourceRepo.Url {
			if chartExists, _ := tc.ChartExists(depName, depVersion, target.Repo); chartExists {
				klog.Infof("Dependency %s-%s already synced\n", depName, depVersion)
			} else {
				if syncDependencies {
					klog.Infof("Dependency %s-%s not synced yet. Syncing now\n", depName, depVersion)
					Sync(depName, depVersion, sourceRepo, target, true)
					// Verify is already published in target repo
					if chartExists, _ := tc.ChartExists(depName, depVersion, target.Repo); chartExists {
						klog.Infof("Dependency %s-%s synced: Continuing with main chart\n", depName, depVersion)
					} else {
						klog.Infof("Dependency %s-%s not synced yet.\n", depName, depVersion)
					}
				} else {
					errs = multierror.Append(errs, errors.Errorf("Please sync %s-%s dependency first", depName, depVersion))
					missingDependencies = true
				}
			}
		} else {
			klog.Infof("Dependency %s-%s should exist in external repository %s \n", depName, depVersion, depRepository)
		}
	}
	return dependenciesMap, missingDependenciesMap, nil
}

// updateRequirementsFile returns the full list of dependencies and the list of missing dependencies.
func updateRequirementsFile(chartPath string, chartDependencies map[string]string, sourceRepo *api.Repo, target *api.TargetRepo) error {
	requirementsFile := path.Join(chartPath, "requirements.yaml")
	// Update requirements.yaml file to point to target repo
	requirements, err := ioutil.ReadFile(requirementsFile)
	if err != nil {
		return errors.Annotatef(err, "Error reading %s file", requirementsFile)
	}

	if !missingDependencies {
		klog.V(3).Info("Updating requirements.yaml file...")
		// Update requirements.yaml file to point to target repo
		requirements, err := ioutil.ReadFile(requirementsFile)
		if err != nil {
			return errors.Annotatef(err, "Error reading %s file", requirementsFile)
		}

		deps := &dependencies{}
		err = yaml.Unmarshal(requirements, deps)
		if err != nil {
			return errors.Annotatef(err, "Error unmarshaling %s file", requirementsFile)
		}
		for i := range deps.Dependencies {
			// Specify the exact dependencies versions used in the original requirements.lock file
			// so when running helm dep up we get the same versions resolved.
			deps.Dependencies[i].Version = chartDependenciesMap[deps.Dependencies[i].Name]
			// Maybe there are dependencies from other chart repos. In this case we don't want to replace
			// the repository.
			// For example, old dependencies obtained from helm/charts repo
			if deps.Dependencies[i].Repository == sourceRepo.Url {
				deps.Dependencies[i].Repository = target.Repo.Url
			}
		}
		// Write updated requirements.yaml file
		if err := writeRequirementsFile(chartPath, deps); err != nil {
			return errors.Trace(err)
		}
		if err := helmcli.UpdateDependencies(chartPath); err != nil {
			return errors.Trace(err)
		}
	}
	return errs
}

// writeRequirementsFile writes a requirements.yaml file to disk.
func writeRequirementsFile(chartPath string, deps *dependencies) error {
	data, err := yaml.Marshal(deps)
	if err != nil {
		return err
	}
	requirementsFileName := "requirements.yaml"
	dest := path.Join(chartPath, requirementsFileName)
	return ioutil.WriteFile(dest, data, 0644)
}
