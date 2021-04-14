package chart

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"

	"github.com/juju/errors"
	"github.com/mkmik/multierror"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/provenance"
	"k8s.io/klog"
	"sigs.k8s.io/yaml"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"github.com/bitnami-labs/charts-syncer/pkg/client/core"
)

// dependencies is the list of dependencies of a chart
type dependencies struct {
	Dependencies []*chart.Dependency `json:"dependencies"`
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

// GetChartLock returns the chart.Lock from an uncompressed chart
func GetChartLock(chartPath string) (*chart.Lock, error) {
	// If the API version is not set, there is not a lock file. Hence, this
	// chart has no dependencies.
	apiVersion, err := GetLockAPIVersion(chartPath)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if apiVersion == "" {
		return nil, nil
	}

	lockFilePath, err := lockFilePath(chartPath, apiVersion)
	if err != nil {
		return nil, errors.Trace(err)
	}
	lockContent, err := ioutil.ReadFile(lockFilePath)
	if err != nil {
		return nil, errors.Trace(err)
	}
	lock := &chart.Lock{}
	if err = yaml.Unmarshal(lockContent, lock); err != nil {
		return nil, errors.Annotatef(err, "unmarshaling %q file", lockFilePath)
	}
	return lock, nil
}

// GetChartDependencies returns the chart chart.Dependencies from a chart in tgz format.
func GetChartDependencies(filepath string, name string) ([]*chart.Dependency, error) {
	// Create temporary working directory
	chartPath, err := ioutil.TempDir("", "charts-syncer")
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer os.RemoveAll(chartPath)

	// Uncompress chart
	if err := utils.Untar(filepath, chartPath); err != nil {
		return nil, errors.Annotatef(err, "uncompressing %q", filepath)
	}
	// Untar uncompress the chart in a subfolder
	chartPath = path.Join(chartPath, name)

	lock, err := GetChartLock(chartPath)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// No dependencies found
	if lock == nil {
		return nil, nil
	}

	return lock.Dependencies, nil
}

// GetLockAPIVersion returns the apiVersion field of a chart's lock file
func GetLockAPIVersion(chartPath string) (string, error) {
	if ok, err := utils.FileExists(path.Join(chartPath, RequirementsLockFilename)); err != nil {
		return "", errors.Trace(err)
	} else if ok {
		return APIV1, nil
	}
	if ok, err := utils.FileExists(path.Join(chartPath, ChartLockFilename)); err != nil {
		return "", errors.Trace(err)
	} else if ok {
		return APIV2, nil
	}

	return "", nil
}

// BuildDependencies updates a local charts directory and update repository references in Chart.yaml/Chart.lock/requirements.yaml/requirements.lock
//
// If reads the Chart.lock/requirements.lock file to download the versions from the target
// chart repository (it assumes all charts are stored in a single repo).
func BuildDependencies(chartPath string, r core.Reader, sourceRepo, targetRepo *api.Repo) error {
	// Build deps manually for OCI as helm does not support it yet
	if err := os.RemoveAll(path.Join(chartPath, "charts")); err != nil {
		return errors.Trace(err)
	}
	// Re-create empty charts folder
	err := os.Mkdir(path.Join(chartPath, "charts"), 0755)
	if err != nil {
		return errors.Trace(err)
	}
	lock, err := GetChartLock(chartPath)
	if err != nil {
		return errors.Trace(err)
	}
	// Step 1. Update references in Chart.yaml/Chart.lock or Requirements.yaml/Requirements.lock
	// If the API version is not set, there is not a lock file. Hence, this
	// chart has no dependencies.
	apiVersion, err := GetLockAPIVersion(chartPath)
	if err != nil {
		return errors.Trace(err)
	}
	if apiVersion == "" {
		return nil
	}
	switch apiVersion {
	case APIV1:
		if err := updateRequirementsFile(chartPath, lock, sourceRepo, targetRepo); err != nil {
			return errors.Trace(err)
		}
	case APIV2:
		if err := updateChartMetadataFile(chartPath, lock, sourceRepo, targetRepo); err != nil {
			return errors.Trace(err)
		}
	default:
		return errors.Errorf("unrecognised apiVersion %s", apiVersion)
	}

	// Step 2. Build charts/ folder
	var errs error
	if lock != nil {
		for _, dep := range lock.Dependencies {
			id := fmt.Sprintf("%s-%s", dep.Name, dep.Version)
			klog.V(4).Infof("Building %q chart dependency", id)

			depTgz, err := r.Fetch(dep.Name, dep.Version)
			if err != nil {
				errs = multierror.Append(errs, errors.Annotatef(err, "fetching %q chart", id))
				continue
			}

			depFile := path.Join(chartPath, "charts", fmt.Sprintf("%s.tgz", id))
			if err := utils.CopyFile(depFile, depTgz); err != nil {
				errs = multierror.Append(errs, errors.Annotatef(err, "copying %q chart to %q", id, depFile))
				continue
			}
		}
	}

	return errs
}

// updateChartMetadataFile updates the dependencies in Chart.yaml
// For helm v3 dependency management
func updateChartMetadataFile(chartPath string, lock *chart.Lock, sourceRepo, targetRepo *api.Repo) error {
	chartFile := path.Join(chartPath, ChartFilename)
	chartYamlContent, err := ioutil.ReadFile(chartFile)
	if err != nil {
		return errors.Trace(err)
	}
	chartMetadata := &chart.Metadata{}
	err = yaml.Unmarshal(chartYamlContent, chartMetadata)
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
		if dep.Repository == sourceRepo.GetUrl() {
			repoUrl, err := getDependencyRepoURL(targetRepo)
			if err != nil {
				return errors.Trace(err)
			}
			dep.Repository = repoUrl
		}
	}
	// Write updated requirements yamls file
	writeChartMetadataFile(chartPath, chartMetadata)
	if err := updateLockFile(chartPath, lock, chartMetadata.Dependencies, sourceRepo, targetRepo, false); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// updateRequirementsFile returns the full list of dependencies and the list of missing dependencies.
// For helm v2 dependency management
func updateRequirementsFile(chartPath string, lock *chart.Lock, sourceRepo, targetRepo *api.Repo) error {
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
		if dep.Repository == sourceRepo.GetUrl() {
			repoUrl, err := getDependencyRepoURL(targetRepo)
			if err != nil {
				return errors.Trace(err)
			}
			dep.Repository = repoUrl
		}
	}
	// Write updated requirements yamls file
	writeRequirementsFile(chartPath, deps)
	if err := updateLockFile(chartPath, lock, deps.Dependencies, sourceRepo, targetRepo, true); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// updateLockFile updates the lock file with the new registry
func updateLockFile(chartPath string, lock *chart.Lock, deps []*chart.Dependency, sourceRepo *api.Repo, targetRepo *api.Repo, legacyLockfile bool) error {
	for _, dep := range lock.Dependencies {
		if dep.Repository == sourceRepo.GetUrl() {
			repoUrl, err := getDependencyRepoURL(targetRepo)
			if err != nil {
				return errors.Trace(err)
			}
			dep.Repository = repoUrl
		}
	}
	newDigest, err := hashDeps(deps, lock.Dependencies)
	if err != nil {
		return errors.Trace(err)
	}
	lock.Digest = newDigest

	// Write updated requirements yamls file
	writeLockFile(chartPath, lock, legacyLockfile)
	return nil
}

// findDepByName returns the dependency that matches a provided name from a list of dependencies.
func findDepByName(dependencies []*chart.Dependency, name string) *chart.Dependency {
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
func writeChartMetadataFile(chartPath string, chartMetadata *chart.Metadata) error {
	data, err := yaml.Marshal(chartMetadata)
	if err != nil {
		return err
	}
	chartMetadataFileName := ChartFilename
	dest := path.Join(chartPath, chartMetadataFileName)
	return ioutil.WriteFile(dest, data, 0644)
}

// writeLockFile writes a lockfile to disk
func writeLockFile(chartPath string, lock *chart.Lock, legacyLockfile bool) error {
	data, err := yaml.Marshal(lock)
	if err != nil {
		return err
	}
	lockFileName := "Chart.lock"
	if legacyLockfile {
		lockFileName = "requirements.lock"
	}
	dest := path.Join(chartPath, lockFileName)
	return ioutil.WriteFile(dest, data, 0644)
}

// hashDeps generates a hash of the dependencies.
//
// This should be used only to compare against another hash generated by this
// function.
func hashDeps(req, lock []*chart.Dependency) (string, error) {
	data, err := json.Marshal([2][]*chart.Dependency{req, lock})
	if err != nil {
		return "", err
	}
	s, err := provenance.Digest(bytes.NewBuffer(data))
	return "sha256:" + s, err
}

// getDependencyRepoURL calculates and return the proper URL to be used in dependencies files
func getDependencyRepoURL(targetRepo *api.Repo) (string, error) {
	repoUrl := targetRepo.GetUrl()
	if targetRepo.GetKind() == api.Kind_OCI {
		parseUrl, err := url.Parse(repoUrl)
		if err != nil {
			return "", errors.Trace(err)
		}
		parseUrl.Scheme = "oci"
		repoUrl = parseUrl.String()
	}
	return repoUrl, nil
}
