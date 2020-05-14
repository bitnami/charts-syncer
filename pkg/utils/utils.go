package utils

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"github.com/bitnami-labs/chart-repository-syncer/api"
	"github.com/juju/errors"
	helmRepo "helm.sh/helm/v3/pkg/repo"

	"k8s.io/klog"
)

// ChartExistInIndex checks if a specific chart version is present in the index file
func ChartExistInIndex(name string, version string, index *helmRepo.IndexFile) (bool, error) {
	chartVersionFound := false
	var err error
	if index.Entries[name] != nil {
		klog.V(8).Infof("Chart %s exists in index.yaml file. Searching %s version", name, version)
		for i := range index.Entries[name] {
			// Check if chart exists in target repo
			//chartName := index.Entries[chart]
			if index.Entries[name][i].Metadata.Version == version {
				klog.V(8).Infof("Version %s found for chart %s in index.yaml file", index.Entries[name][i].Metadata.Version, name)
				chartVersionFound = true
				break
			}
		}
		if !chartVersionFound {
			return false, errors.Errorf("Chart version %s doesn't exist in index.yaml file", version)
		}
	} else {
		return false, errors.Errorf("%s chart doesn't exist in index.yaml", name)
	}

	return chartVersionFound, errors.Trace(err)
}

// ChartExistInTargetRepo checks if a chart exists in the target repo
// So far, targetRepo should be ChartMuseum-like as we are using its API for checking
func ChartExistInTargetRepo(name string, version string, targetRepo *api.Repo) (bool, error) {
	// Check if chart exists in target repo
	client := &http.Client{}
	req, err := http.NewRequest("GET", targetRepo.Url+"/api/charts/"+name+"/"+version, nil)
	if err != nil {
		return false, errors.Trace(err)
	}
	if targetRepo.Auth != nil && targetRepo.Auth.Username != "" && targetRepo.Auth.Password != "" {
		klog.V(12).Info("Target Repo uses basic authentication")
		req.SetBasicAuth(targetRepo.Auth.Username, targetRepo.Auth.Password)
	}
	res, err := client.Do(req)
	if err != nil {
		return false, errors.Trace(err)
	}
	//chartInfo, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return false, errors.Trace(err)
	}
	// Check error codes
	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		klog.V(4).Infof("Chart %s-%s already exists in target repo", name, version)
	} else {
		if res.StatusCode == 404 {
			klog.V(8).Infof("Chart %s-%s not found in target repo \n", name, version)
			return false, errors.Trace(err)
		}
		return false, errors.Annotatef(err, "Error checking if chart exists in repo: %s %d", http.StatusText(res.StatusCode), res.StatusCode)
	}

	return true, errors.Trace(err)
}

// DownloadIndex will download the index.yaml file of a chart repository and return
// the path to the downloaded file
func DownloadIndex(repo *api.Repo) (string, error) {
	downloadURL := repo.Url + "/index.yaml"

	// Get the data
	client := &http.Client{}
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return "", errors.Trace(err)
	}
	if repo.Auth != nil && repo.Auth.Username != "" && repo.Auth.Password != "" {
		klog.V(12).Info("Repo configures basic authentication. Downloading index.yaml...")
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

// Untar will uncompress a tarball
func Untar(filepath, destDir string) error {
	// Uncompress tarball
	klog.V(8).Info("Extracting source chart...")
	cmd := exec.Command("tar", "xzf", filepath, "--directory", destDir)
	_, err := cmd.Output()
	if err != nil {
		return errors.Annotatef(err, "Error untaring chart package %s", filepath)
	}
	return errors.Trace(err)
}

// GetFileContentType returns the content type of a file
func GetFileContentType(filepath string) (string, error) {
	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)
	// Open File
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
