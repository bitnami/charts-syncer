package utils

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/juju/errors"
	helmRepo "helm.sh/helm/v3/pkg/repo"

	"k8s.io/klog"
)

const (
	timeLayoutISO = "2006-01-02"
)

// LoadIndexFromRepo get the index.yaml from a Helm repo and returns an index object
func LoadIndexFromRepo(repo *api.Repo) (*helmRepo.IndexFile, error) {
	indexFile, err := downloadIndex(repo)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer os.Remove(indexFile)
	index, err := helmRepo.LoadIndexFile(indexFile)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return index, errors.Trace(err)
}

// ChartExistInIndex checks if a specific chart version is present in the index file.
func ChartExistInIndex(name string, version string, index *helmRepo.IndexFile) (bool, error) {
	chartVersionFound := false
	if index.Entries[name] != nil {
		klog.V(3).Infof("Chart %q exists in index.yaml file. Searching %q version", name, version)
		for i := range index.Entries[name] {
			if index.Entries[name][i].Metadata.Version == version {
				klog.V(3).Infof("Version %q found for chart %q in index.yaml file", index.Entries[name][i].Metadata.Version, name)
				chartVersionFound = true
				break
			}
		}
		if !chartVersionFound {
			return false, nil
		}
	} else {
		return false, nil
	}

	return chartVersionFound, nil
}

// downloadIndex will download the index.yaml file of a chart repository and return
// the path to the downloaded file.
func downloadIndex(repo *api.Repo) (string, error) {
	downloadURL := repo.Url + "/index.yaml"

	// Get the data
	client := &http.Client{}
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return "", errors.Trace(err)
	}
	if repo.Auth != nil && repo.Auth.Username != "" && repo.Auth.Password != "" {
		klog.V(4).Info("Repo configures basic authentication. Downloading index.yaml...")
		req.SetBasicAuth(repo.Auth.Username, repo.Auth.Password)
	}
	res, err := client.Do(req)
	if err != nil {
		return "", errors.Trace(err)
	}
	defer res.Body.Close()
	// Check status code
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return "", errors.Errorf("Error downloading index.yaml from %s. Status code is %d", repo.Url, res.StatusCode)
	}

	// Create the file
	out, err := ioutil.TempFile("", "index.*.yaml")
	if err != nil {
		klog.Fatal(err)
	}

	// Write the body to file
	_, err = io.Copy(out, res.Body)
	return out.Name(), errors.Trace(err)
}

// Untar will uncompress a tarball.
func Untar(filepath, destDir string) error {
	// Uncompress tarball
	klog.V(3).Info("Extracting source chart...")
	cmd := exec.Command("tar", "xzf", filepath, "--directory", destDir)
	_, err := cmd.Output()
	if err != nil {
		return errors.Annotatef(err, "Error untaring chart package %s", filepath)
	}
	return errors.Trace(err)
}

// GetFileContentType returns the content type of a file.
func GetFileContentType(filepath string) (string, error) {
	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)
	file, err := os.Open(filepath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	n, err := file.Read(buffer)
	if err != nil {
		return "", err
	}
	// Use the net/http package's handy DectectContentType function. Always returns a valid
	// content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer[:n])
	return contentType, err
}

// GetDateThreshold will parse a string date agains a fixed layout and return a time.Date value
func GetDateThreshold(date string) (time.Time, error) {
	if date == "" {
		return time.Unix(0, 0), nil
	}
	dateThreshold, err := time.Parse(timeLayoutISO, date)
	if err != nil {
		return dateThreshold, errors.Trace(err)
	}
	return dateThreshold, nil
}

// FindChartURL will return the chart url
func FindChartURL(name string, version string, index *helmRepo.IndexFile, sourceURL string) (string, error) {
	chart := findChartByVersion(index.Entries[name], version)
	if chart != nil && len(chart.URLs) > 0 {
		if isValidURL(chart.URLs[0]) {
			return chart.URLs[0], nil
		}
		return fmt.Sprintf("%s/%s", sourceURL, chart.URLs[0]), nil
	}
	return "", fmt.Errorf("unable to find chart url in index")
}

// findChartByVersion returns the chart that matches a provided version from a list of charts.
func findChartByVersion(chartVersions []*helmRepo.ChartVersion, version string) *helmRepo.ChartVersion {
	for _, chart := range chartVersions {
		if chart.Version == version {
			return chart
		}
	}
	return nil
}

// isValidUrl tests a string to determine if it is a well-structured url or not.
func isValidURL(text string) bool {
	_, err := url.ParseRequestURI(text)
	if err != nil {
		return false
	}
	return true
}
