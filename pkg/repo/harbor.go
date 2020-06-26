package repo

import (
	"fmt"
	"strings"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/utils"
	"github.com/juju/errors"
	helmRepo "helm.sh/helm/v3/pkg/repo"
	"k8s.io/klog"
)

// HarborClient implements ChartRepoAPI for a Harbor implementation.
type HarborClient struct {
	repo *api.Repo
}

// NewHarborClient creates a new `HarborClient`.
func NewHarborClient(repo *api.Repo) *HarborClient {
	return &HarborClient{repo: repo}
}

// PublishChart publishes a packaged chart to Harbor repository.
func (c *HarborClient) PublishChart(filepath string, targetRepo *api.Repo) error {
	klog.V(3).Infof("Publishing %s to Harbor repo", filepath)
	apiEndpoint := strings.Replace(targetRepo.Url, "/chartrepo/", "/api/chartrepo/", 1) + "/charts"
	if err := pushToChartMuseumLike(apiEndpoint, filepath, targetRepo); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// DownloadChart downloads a packaged chart from Harbor repository.
func (c *HarborClient) DownloadChart(filepath string, name string, version string, sourceRepo *api.Repo, index *helmRepo.IndexFile) error {
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
func (c *HarborClient) ChartExists(name string, version string, repo *api.Repo) (bool, error) {
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
