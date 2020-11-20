package core

import (
	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/utils"
	"github.com/juju/errors"
	helmRepo "helm.sh/helm/v3/pkg/repo"
	"k8s.io/klog"
)

// ClassicHelmClient implements Client for a Helm classic implementation.
type ClassicHelmClient struct {
	repo *api.Repo
}

// NewClassicHelmClient creates a new `ClassicHelmClient`.
func NewClassicHelmClient(repo *api.Repo) *ClassicHelmClient {
	return &ClassicHelmClient{repo: repo}
}

// Push publishes a packaged chart to classic helm repository.
func (c *ClassicHelmClient) Push(filepath string, targetRepo *api.Repo) error {
	klog.V(3).Infof("Publishing %s to classic helm repo", filepath)
	return errors.Errorf("publishing to a Helm classic repository is not supported yet")
}

// Fetch downloads a packaged chart from a classic helm repository.
func (c *ClassicHelmClient) Fetch(filepath string, name string, version string, sourceRepo *api.Repo, index *helmRepo.IndexFile) error {
	klog.V(3).Infof("Downloading %s-%s from classic helm repo", name, version)
	downloadURL, err := utils.FindChartURL(name, version, index, sourceRepo.Url)
	if err != nil {
		return errors.Trace(err)
	}
	if err := download(filepath, downloadURL, sourceRepo); err != nil {
		return errors.Trace(err)
	}
	// Check contentType
	contentType, err := utils.GetFileContentType(filepath)
	if err != nil {
		return errors.Trace(err)
	}
	if contentType != "application/x-gzip" {
		return errors.Errorf("the downloaded chart %s is not a gzipped tarball", filepath)
	}
	return nil
}

// ChartExists checks if a chart exists in the repo.
func (c *ClassicHelmClient) ChartExists(name string, version string, index *helmRepo.IndexFile) (bool, error) {
	klog.V(3).Infof("Checking if %s-%s chart exists", name, version)
	chartExists, err := utils.ChartExistInIndex(name, version, index)
	if err != nil {
		return false, errors.Trace(err)
	}
	return chartExists, nil
}
