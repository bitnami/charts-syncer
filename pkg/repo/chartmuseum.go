package repo

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/bitnami-labs/chart-repository-syncer/api"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/utils"
	"github.com/juju/errors"
	"k8s.io/klog"
)

// ChartMuseumClient implements ChartRepoAPI for a ChartMuseum implementation.
type ChartMuseumClient struct {
	repo *api.Repo
}

// NewChartMuseumClient creates a new `ChartMuseumClient`
func NewChartMuseumClient(repo *api.Repo) *ChartMuseumClient {
	return &ChartMuseumClient{repo: repo}
}

// PublishChart publishes a packaged chart to ChartsMuseum repository
func (c *ChartMuseumClient) PublishChart(filepath string, targetRepo *api.Repo) error {
	klog.V(3).Infof("Publishing %s to chartmuseum repo", filepath)
	publishURL := targetRepo.Url + "/api/charts"
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	fileWriter, err := bodyWriter.CreateFormFile("chart", filepath)
	if err != nil {
		return errors.Annotate(err, "Error writing to buffer")
	}

	fh, err := os.Open(filepath)
	if err != nil {
		return errors.Annotatef(err, "Error opening file %s", filepath)
	}
	defer fh.Close()

	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		return errors.Trace(err)
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	req, err := http.NewRequest("POST", publishURL, bodyBuf)
	klog.V(4).Infof("POST %q", publishURL)
	req.Header.Add("content-type", contentType)
	if err != nil {
		return errors.Annotatef(err, "Error creating POST request to %s", publishURL)
	}
	if targetRepo.Auth != nil && targetRepo.Auth.Username != "" && targetRepo.Auth.Password != "" {
		klog.V(4).Info("Target repo uses basic authentication...")
		req.SetBasicAuth(targetRepo.Auth.Username, targetRepo.Auth.Password)
	}
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return errors.Annotatef(err, "Error doing POST request to %s", publishURL)
	}
	defer res.Body.Close()
	respBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.Annotatef(err, "Error reading POST response from %s", publishURL)
	}
	klog.V(4).Infof("POST chart status Code: %d, Message: %s", res.StatusCode, string(respBody))
	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		klog.V(3).Infof("Chart %s uploaded successfully", filepath)
	} else {
		return errors.Errorf("POST chart status Code: %d, Message: %s", res.StatusCode, string(respBody))
	}
	return errors.Trace(err)
}

// DownloadChart downloads a packaged chart from ChartsMuseum repository
func (c *ChartMuseumClient) DownloadChart(filepath string, name string, version string, sourceRepo *api.Repo) error {
	klog.V(3).Infof("Downloading %s-%s from chartmuseum repo", name, version)
	downloadURL := sourceRepo.Url + "/charts/" + name + "-" + version + ".tgz"
	if err := download(filepath, name, version, downloadURL, sourceRepo); err != nil {
		return errors.Trace(err)
	}
	// Check contentType
	contentType, err := utils.GetFileContentType(filepath)
	if err != nil {
		return errors.Annotatef(err, "Error checking contentType of %s file", filepath)
	}
	if contentType != "application/x-gzip" {
		return errors.Errorf("The downloaded chart %s is not a gzipped tarball", filepath)
	}
	return nil
}

// ChartExists checks if a chart exists in the repo
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
