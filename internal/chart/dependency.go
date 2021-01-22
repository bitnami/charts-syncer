package chart

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/juju/errors"
	"github.com/mkmik/multierror"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/klog"
	"sigs.k8s.io/yaml"

	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"github.com/bitnami-labs/charts-syncer/pkg/client/core"
)

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

// BuildDependencies updates a local charts directory
//
// If reads the Chart.lock file to download the versions from the remote
// chart repository (it assumes all charts are stored in a single repo).
func BuildDependencies(chartPath string, r core.Reader) error {
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
