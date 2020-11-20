package core

import (
	"fmt"

	"github.com/juju/errors"
	helmRepo "helm.sh/helm/v3/pkg/repo"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/utils"
)

// ClassicHelmClient implements Client for a Helm classic implementation.
type ClassicHelmClient struct {
	repo *api.Repo

	index *helmRepo.IndexFile
}

// NewClassicHelmClient creates a new `ClassicHelmClient`.
func NewClassicHelmClient(repo *api.Repo) *ClassicHelmClient {
	return &ClassicHelmClient{repo: repo}
}

// This allows test to replace the client index for testing.
var reloadHelmClassicIndex = func(c *ClassicHelmClient) error { return c.reloadIndex() }

func (c *ClassicHelmClient) reloadIndex() error {
	index, err := utils.LoadIndexFromRepo(c.repo)
	if err != nil {
		return errors.Trace(fmt.Errorf("error loading index.yaml: %w", err))
	}
	c.index = index
	return nil
}

// Push publishes a packaged chart to classic helm repository.
func (c *ClassicHelmClient) Push(filepath string) error {
	klog.V(3).Infof("Publishing %s to classic helm repo", filepath)
	return errors.Errorf("publishing to a Helm classic repository is not supported yet")
}

// Fetch downloads a packaged chart from a classic helm repository.
func (c *ClassicHelmClient) Fetch(filepath string, name string, version string) error {
	klog.V(3).Infof("Reloading index for %q repo", c.repo.GetUrl())
	if err := reloadHelmClassicIndex(c); err != nil {
		return errors.Trace(err)
	}

	klog.V(3).Infof("Downloading %s-%s from classic helm repo", name, version)
	downloadURL, err := utils.FindChartURL(name, version, c.index, c.repo.GetUrl())
	if err != nil {
		return errors.Trace(err)
	}
	if err := download(filepath, downloadURL, c.repo); err != nil {
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
func (c *ClassicHelmClient) ChartExists(name string, version string) (bool, error) {
	klog.V(3).Infof("Reloading index for %q repo", c.repo.GetUrl())
	if err := reloadHelmClassicIndex(c); err != nil {
		return false, errors.Trace(err)
	}

	klog.V(3).Infof("Checking if %s-%s chart exists", name, version)
	chartExists, err := utils.ChartExistInIndex(name, version, c.index)
	if err != nil {
		return false, errors.Trace(err)
	}
	return chartExists, nil
}
