package utils

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/juju/errors"
	helmRepo "helm.sh/helm/v3/pkg/repo"

	"k8s.io/klog"
)

const (
	timeLayoutISO = "2006-01-02"
)

var (
	// UnixEpoch is the number of seconds that have elapsed since January 1, 1970
	UnixEpoch = time.Unix(0, 0)
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
	downloadURL := repo.GetUrl() + "/index.yaml"

	// Get the data
	client := &http.Client{}
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return "", errors.Trace(err)
	}
	if repo.GetAuth() != nil && repo.GetAuth().GetUsername() != "" && repo.GetAuth().GetPassword() != "" {
		klog.V(4).Info("Repo configures basic authentication. Downloading index.yaml...")
		req.SetBasicAuth(repo.GetAuth().GetUsername(), repo.GetAuth().GetPassword())
	}
	res, err := client.Do(req)
	if err != nil {
		return "", errors.Trace(err)
	}
	defer res.Body.Close()
	// Check status code
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return "", errors.Errorf("error downloading index.yaml from %s. Status code is %d", repo.GetUrl(), res.StatusCode)
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

// Untar extracts compressed archives
//
// Based on Extract function from helm plugin installer. https://github.com/helm/helm/blob/master/pkg/plugin/installer/http_installer.go
func Untar(tarball, targetDir string) error {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	f, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer f.Close()
	gzipReader, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		path := filepath.Join(targetDir, header.Name)
		targetFolder := filepath.Join(targetDir, filepath.Dir(header.Name))
		// For some reason the for loop only iterates over files and not folders, so the switch below for folders is
		// never executed and so we are creating the target folder at this point.
		if _, err := os.Stat(targetFolder); err != nil {
			if err := os.MkdirAll(targetFolder, 0755); err != nil {
				return err
			}
		}
		switch header.Typeflag {
		// Related to previous comment. It seems this block of code is never executed.
		case tar.TypeDir:
			if err := os.Mkdir(path, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		// We don't want to process these extension header files.
		case tar.TypeXGlobalHeader, tar.TypeXHeader:
			continue
		default:
			return errors.Errorf("unknown type: %b in %s", header.Typeflag, header.Name)
		}
	}
	return nil
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
		return UnixEpoch, nil
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

// FileExists will test if a file exists
func FileExists(f string) (bool, error) {
	if _, err := os.Stat(f); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Trace(err)
	}
	return true, nil
}

// CopyFile copies a file from srcPath to destPath, ensuring the destPath directory exists.
func CopyFile(destPath string, srcPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return errors.Trace(err)
	}
	src, err := os.Open(srcPath)
	if err != nil {
		return errors.Trace(err)
	}
	defer src.Close()

	dest, err := os.Create(destPath)
	if err != nil {
		return errors.Trace(err)
	}
	defer dest.Close()

	if _, err := io.Copy(dest, src); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// HTTPResponseBody returns the body of an HTTP response
func HTTPResponseBody(res *http.Response) string {
	var s strings.Builder
	_, _ = io.Copy(&s, res.Body)
	return s.String()
}

// EncodeSha1 returns a SHA1 representation of the provided string
func EncodeSha1(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}
