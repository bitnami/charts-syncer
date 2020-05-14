package repo

import (
	"github.com/bitnami-labs/chart-repository-syncer/api"
	"github.com/juju/errors"
	"k8s.io/klog"
)

// ClassicHelmClient is a client for publishing and downloading
// charts from/to a classic Helm repository
type ClassicHelmClient struct {
	repo *api.Repo
}

// NewClassicHelmClient creates a new `ClassicHelmClient`
func NewClassicHelmClient(repo *api.Repo) *ClassicHelmClient {
	return &ClassicHelmClient{repo: repo}
}

// PublishChart publishes a packaged chart to classic helm repository
func (c *ClassicHelmClient) PublishChart(filepath string, targetRepo *api.Repo) error {
	klog.V(8).Infof("Publishing %s to classic helm repo", filepath)
	return errors.Errorf("Publishing to a Helm classic repository is not supported yet")
}

// DownloadChart downloads a packaged chart from a classic helm repository
func (c *ClassicHelmClient) DownloadChart(filepath string, name string, version string, sourceRepo *api.Repo) error {
	klog.V(8).Infof("Downloading %s-%s from classic helm repo", name, version)
	downloadURL := sourceRepo.Url + "/" + name + "-" + version + ".tgz"
	if err := download(filepath, name, version, downloadURL, sourceRepo); err != nil {
		return errors.Trace(err)
	}
	return nil
}
