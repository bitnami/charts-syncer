package core

import (
	"fmt"
	"strings"

	"github.com/juju/errors"
	helmRepo "helm.sh/helm/v3/pkg/repo"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/utils"
)

// HarborClient implements Client for a Harbor implementation.
type HarborClient struct {
	repo *api.Repo

	index *helmRepo.IndexFile
}

// NewHarborClient creates a new `HarborClient`.
func NewHarborClient(repo *api.Repo) *HarborClient {
	return &HarborClient{repo: repo}
}

// This allows test to replace the client index for testing.
var reloadHarborIndex = func(c *HarborClient) error { return c.reloadIndex() }

func (c *HarborClient) reloadIndex() error {
	index, err := utils.LoadIndexFromRepo(c.repo)
	if err != nil {
		return errors.Trace(fmt.Errorf("error loading index.yaml: %w", err))
	}
	c.index = index
	return nil
}

// Push publishes a packaged chart to Harbor repository.
func (c *HarborClient) Push(filepath string) error {
	klog.V(3).Infof("Publishing %s to Harbor repo", filepath)
	apiEndpoint := strings.Replace(c.repo.GetUrl(), "/chartrepo/", "/api/chartrepo/", 1) + "/charts"
	if err := pushToChartMuseumLike(apiEndpoint, filepath, c.repo); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// Fetch downloads a packaged chart from Harbor repository.
func (c *HarborClient) Fetch(filepath string, name string, version string) error {
	klog.V(3).Infof("Reloading index for %q repo", c.repo.GetUrl())
	if err := reloadHarborIndex(c); err != nil {
		return errors.Trace(err)
	}

	klog.V(3).Infof("Downloading %s-%s from Harbor repo", name, version)
	apiEndpoint, err := utils.FindChartURL(name, version, c.index, c.repo.GetUrl())
	if err != nil {
		return errors.Trace(err)
	}
	if err := downloadFromChartMuseumLike(apiEndpoint, filepath, c.repo); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// ChartExists checks if a chart exists in the repo.
func (c *HarborClient) ChartExists(name string, version string) (bool, error) {
	klog.V(3).Infof("Reloading index for %q repo", c.repo.GetUrl())
	if err := reloadHarborIndex(c); err != nil {
		return false, errors.Trace(err)
	}

	klog.V(3).Infof("Checking if %s-%s chart exists", name, version)
	chartExists, err := utils.ChartExistInIndex(name, version, c.index)
	if err != nil {
		return false, errors.Trace(err)
	}
	return chartExists, nil
}
