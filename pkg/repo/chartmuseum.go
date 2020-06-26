package repo

import (
	"fmt"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/utils"
	"github.com/juju/errors"
	helmRepo "helm.sh/helm/v3/pkg/repo"
	"k8s.io/klog"
)

// ChartMuseumClient implements ChartRepoAPI for a ChartMuseum implementation.
type ChartMuseumClient struct {
	repo *api.Repo
}

// NewChartMuseumClient creates a new `ChartMuseumClient`.
func NewChartMuseumClient(repo *api.Repo) *ChartMuseumClient {
	return &ChartMuseumClient{repo: repo}
}

// PublishChart publishes a packaged chart to ChartsMuseum repository.
func (c *ChartMuseumClient) PublishChart(filepath string, targetRepo *api.Repo) error {
	klog.V(3).Infof("Publishing %s to chartmuseum repo", filepath)
	apiEndpoint := targetRepo.Url + "/api/charts"
	if err := pushToChartMuseumLike(apiEndpoint, filepath, targetRepo); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// DownloadChart downloads a packaged chart from ChartsMuseum repository.
func (c *ChartMuseumClient) DownloadChart(filepath string, name string, version string, sourceRepo *api.Repo, index *helmRepo.IndexFile) error {
	klog.V(3).Infof("Downloading %s-%s from Harbor repo", name, version)
	apiEndpoint, err := utils.FindChartURL(name, version, index)
	if err != nil {
		return errors.Trace(err)
	}
	if err := downloadFromChartMuseumLike(apiEndpoint, filepath, sourceRepo); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// ChartExists checks if a chart exists in the repo.
func (c *ChartMuseumClient) ChartExists(name string, version string, repo *api.Repo) (bool, error) {
	klog.V(3).Infof("Checking if %s-%s chart exists in %q", name, version, repo.Url)
	index, err := utils.LoadIndexFromRepo(repo)
	if err != nil {
		return false, errors.Trace(fmt.Errorf("Error loading index.yaml: %w", err))
	}
	chartExists, err := utils.ChartExistInIndex(name, version, index)
	if err != nil {
		return false, errors.Trace(err)
	}
	return chartExists, nil
}
