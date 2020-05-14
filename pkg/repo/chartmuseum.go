package repo

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/bitnami-labs/chart-repository-syncer/api"
	"github.com/juju/errors"
	"k8s.io/klog"
)

// ChartMuseumClient is a client for publishing and downloading
// charts from/to a ChartMuseumRepository
type ChartMuseumClient struct {
	repo *api.Repo
}

// NewChartMuseumClient creates a new `ChartMuseumClient`
func NewChartMuseumClient(repo *api.Repo) *ChartMuseumClient {
	return &ChartMuseumClient{repo: repo}
}

// PublishChart publishes a packaged chart to ChartsMuseum repository
func (c *ChartMuseumClient) PublishChart(filepath string, targetRepo *api.Repo) error {
	klog.V(8).Infof("Publishing %s to chartmuseum repo", filepath)
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
	req.Header.Add("content-type", contentType)
	if err != nil {
		return errors.Annotatef(err, "Error creating POST request to %s", publishURL)
	}
	if targetRepo.Auth != nil && targetRepo.Auth.Username != "" && targetRepo.Auth.Password != "" {
		klog.V(12).Info("Target repo uses basic authentication...")
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
	klog.V(12).Infof("POST chart status Code: %d, Message: %s", res.StatusCode, string(respBody))
	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		klog.V(8).Infof("Chart %s uploaded successfully", filepath)
	} else {
		errors.Annotatef(err, "Error publishing %s chart", filepath)
		return errors.New("Post status code is not 2xx")
	}
	return errors.Trace(err)
}

// DownloadChart downloads a packaged chart from ChartsMuseum repository
func (c *ChartMuseumClient) DownloadChart(filepath string, name string, version string, sourceRepo *api.Repo) error {
	klog.V(8).Infof("Downloading %s-%s from chartmuseum repo", name, version)
	downloadURL := sourceRepo.Url + "/charts/" + name + "-" + version + ".tgz"
	if err := download(filepath, name, version, downloadURL, sourceRepo); err != nil {
		return errors.Trace(err)
	}
	return nil
}
