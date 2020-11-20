package core

import (
	"fmt"

	"github.com/juju/errors"
	helmRepo "helm.sh/helm/v3/pkg/repo"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/utils"
)

// ChartMuseumClient implements Client for a ChartMuseum implementation.
type ChartMuseumClient struct {
	repo *api.Repo

	index *helmRepo.IndexFile
}

// NewChartMuseumClient creates a new `ChartMuseumClient`.
func NewChartMuseumClient(repo *api.Repo) (*ChartMuseumClient, error) {
	c := &ChartMuseumClient{repo: repo}
	if err := c.reloadIndex(); err != nil {
		return c, err
	}
	return c, nil
}

// Push publishes a packaged chart to ChartsMuseum repository.
func (c *ChartMuseumClient) Push(filepath string) error {
	klog.V(3).Infof("Publishing %s to chartmuseum repo", filepath)
	apiEndpoint := c.repo.GetUrl() + "/api/charts"
	if err := pushToChartMuseumLike(apiEndpoint, filepath, c.repo); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// Fetch downloads a packaged chart from ChartsMuseum repository.
func (c *ChartMuseumClient) Fetch(filepath string, name string, version string) error {
	klog.V(3).Infof("Downloading %s-%s from Chartmuseum repo", name, version)
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
func (c *ChartMuseumClient) ChartExists(name string, version string) (bool, error) {
	klog.V(3).Infof("Reloading index for %q repo", c.repo.GetUrl())
	if err := c.reloadIndex(); err != nil {
		return false, errors.Trace(err)
	}

	klog.V(3).Infof("Checking if %s-%s chart exists", name, version)
	chartExists, err := utils.ChartExistInIndex(name, version, c.index)
	if err != nil {
		return false, errors.Trace(err)
	}
	return chartExists, nil
}

func (c *ChartMuseumClient) reloadIndex() error {
	index, err := utils.LoadIndexFromRepo(c.repo)
	if err != nil {
		return errors.Trace(fmt.Errorf("error loading index.yaml: %w", err))
	}
	c.index = index
	return nil
}
