package chart

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"regexp"

	"github.com/juju/errors"
	"github.com/mkmik/multierror"
	"k8s.io/klog"

	"github.com/bitnami-labs/chart-repository-syncer/api"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/helmcli"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/utils"

	"gopkg.in/yaml.v2"
	helm_chart "helm.sh/helm/v3/pkg/chart"
	helm_repo "helm.sh/helm/v3/pkg/repo"
)

// Dependencies is the list of dependencies of a chart
type Dependencies struct {
	// Dependencies is the list of dependencies
	Dependencies []*helm_chart.Dependency `json:"dependencies"`
}

// Download will download the .tgz of a chart to the given filepath
func Download(filepath string, name string, version string, sourceRepo *api.SourceRepo) error {
	var downloadURL string
	if sourceRepo.Kind == "chartmuseum" {
		downloadURL = sourceRepo.Url + "/charts/" + name + "-" + version + ".tgz" // For chartmuseum
	} else {
		downloadURL = sourceRepo.Url + "/" + name + "-" + version + ".tgz" // For generic helm repos
	}

	// Get the data
	client := &http.Client{}
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return errors.Annotate(err, "Error getting chart")
	}
	if sourceRepo.Auth != nil && sourceRepo.Auth.Username != "" && sourceRepo.Auth.Password != "" {
		klog.V(12).Info("Source repo configures basic authentication. Downloading chart...")
		req.SetBasicAuth(sourceRepo.Auth.Username, sourceRepo.Auth.Password)
	}
	res, err := client.Do(req)
	if err != nil {
		return errors.Annotate(err, "Error doing request")
	}
	defer res.Body.Close()

	// Check status code
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return errors.Errorf("Error downloading chart %s-%s. Status code is %d", name, version, res.StatusCode)
	}
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return errors.Annotatef(err, "Error creating %s file", filepath)
	}
	defer out.Close()

	// Write the body to file
	if _, err = io.Copy(out, res.Body); err != nil {
		return errors.Annotatef(err, "Error write to file %s", filepath)
	}

	// Check contentType
	contentType, err := utils.GetFileContentType(filepath)
	if err != nil {
		return errors.Annotatef(err, "Error checking contentType of %s file", filepath)
	}
	if contentType != "application/x-gzip" {
		return errors.Errorf("The downloaded chart %s is not a gzipped tarball", filepath)
	}
	return errors.Trace(err)
}

// Publish takes a .tgz chart and publish to a target repo
// Currently only works for chartmuseum repos
func Publish(filepath string, targetRepo *api.TargetRepo) error {
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

	client := &http.Client{}
	req, err := http.NewRequest("POST", publishURL, bodyBuf)
	req.Header.Add("content-type", contentType)
	if err != nil {
		return errors.Annotatef(err, "Error creating POST request to %s", publishURL)
	}
	if targetRepo.Auth != nil && targetRepo.Auth.Username != "" && targetRepo.Auth.Password != "" {
		klog.V(12).Info("Target repo uses basic authentication...")
		req.SetBasicAuth(targetRepo.Auth.Username, targetRepo.Auth.Password)
	}
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

// SyncAllVersions will sync all versions of a specific char.
func SyncAllVersions(name string, sourceRepo *api.SourceRepo, targetRepo *api.TargetRepo, syncDependencies bool, index *helm_repo.IndexFile, dryRun bool) error {
	var errs error
	if index.Entries[name] != nil {
		for i := range index.Entries[name] {
			if chartExists, err := utils.ChartExistInTargetRepo(name, index.Entries[name][i].Metadata.Version, targetRepo); !chartExists && err == nil {
				if dryRun {
					klog.Infof("dry-run: Chart %s-%s pending to be synced", name, index.Entries[name][i].Metadata.Version)
				} else {
					if err := Sync(name, index.Entries[name][i].Metadata.Version, sourceRepo, targetRepo, syncDependencies); err != nil {
						errs = multierror.Append(errs, errors.Trace(err))
					}
				}
			}
		}
	} else {
		return errors.Errorf("Chart %s not found in source repo", name)
	}
	return errs
}

// Sync is the main function.  It downloads, transform, package and publish a chart
func Sync(name string, version string, sourceRepo *api.SourceRepo, targetRepo *api.TargetRepo, syncDependencies bool) error {
	// Create temporary working directory
	tmpDir, err := ioutil.TempDir("", "c3tsyncer")
	if err != nil {
		return errors.Annotatef(err, "Error creating temporary: %s", tmpDir)
	}
	defer os.RemoveAll(tmpDir)
	srcDir := path.Join(tmpDir, "src")
	destDir := path.Join(tmpDir, "dest")
	for _, path := range []string{srcDir, destDir} {
		os.MkdirAll(path, 0775)
	}

	// Download chart
	filepath := srcDir + "/" + name + "-" + version + ".tgz"
	klog.V(12).Infof("srcDir: %s", srcDir)
	klog.V(12).Infof("destDir: %s", destDir)
	klog.V(12).Infof("chartPath: %s", filepath)
	if err := Download(filepath, name, version, sourceRepo); err != nil {
		return errors.Annotatef(err, "Error downloading chart %s-%s from source repo", name, version)
	}

	// Uncompress chart
	if err := utils.Untar(filepath, destDir); err != nil {
		return errors.Annotate(err, "Error found in Untar function")
	}

	// If chart has dependencies, check that they are already in the target repo.
	chartPath := path.Join(destDir, name)
	if _, err := os.Stat(path.Join(chartPath, "requirements.lock")); err == nil {
		if err := manageDependencies(chartPath, sourceRepo, targetRepo, syncDependencies); err != nil {
			return errors.Annotatef(err, "Error updating dependencies for chart %s-%s", name, version)
		}
	}

	// Update values.yaml with new registry and repository info
	valuesFile := path.Join(chartPath, "values.yaml")
	valuesProductionFile := path.Join(chartPath, "values-production.yaml")
	if _, err := os.Stat(valuesFile); err == nil {
		klog.V(8).Infof("Chart %s-%s has values.yaml file...", name, version)
		updateValuesFile(valuesFile, targetRepo)
	}
	if _, err := os.Stat(valuesProductionFile); err == nil {
		klog.V(8).Infof("Chart %s-%s has values-production.yaml...", name, version)
		updateValuesFile(valuesProductionFile, targetRepo)
	}

	// Package chart
	packagedChartPath, err := helmcli.Package(chartPath, name, version, destDir)
	if err != nil {
		return errors.Annotate(err, "Error taring chart")
	}

	// Publish to target repo
	if err := Publish(packagedChartPath, targetRepo); err != nil {
		return errors.Annotatef(err, "Error publishing chart %s", filepath)
	}
	klog.Infof("Chart %s-%s published successfully", name, version)

	return errors.Trace(err)
}

// manageDependencies takes care of updating dependencies to correct version and sync to target repo if necesary
func manageDependencies(chartPath string, sourceRepo *api.SourceRepo, targetRepo *api.TargetRepo, syncDependencies bool) error {
	var errs error
	var missingDependencies = false
	dependencies := make(map[string]string)

	requirementsLockFile := path.Join(chartPath, "requirements.lock")
	requirementsFile := path.Join(chartPath, "requirements.yaml")
	klog.V(4).Info("Chart has dependencies...")

	requirementsLock, err := ioutil.ReadFile(requirementsLockFile)
	if err != nil {
		return errors.Annotatef(err, "Error reading %s file", requirementsLockFile)
	}

	lock := &helm_chart.Lock{}
	err = yaml.Unmarshal(requirementsLock, lock)
	if err != nil {
		return errors.Annotatef(err, "Error unmarshaling %s file", requirementsLockFile)
	}
	for i := range lock.Dependencies {
		// Check if chart exists in target repo
		depName := lock.Dependencies[i].Name
		depVersion := lock.Dependencies[i].Version
		depRepository := lock.Dependencies[i].Repository
		dependencies[depName] = depVersion
		// Only sync dependencies retrieved from source repo.
		if depRepository == sourceRepo.Url {
			if chartExists, _ := utils.ChartExistInTargetRepo(depName, depVersion, targetRepo); chartExists {
				klog.Infof("Dependency %s-%s already synced\n", depName, depVersion)
			} else {
				if syncDependencies {
					klog.Infof("Dependency %s-%s not synced yet. Syncing now\n", depName, depVersion)
					Sync(depName, depVersion, sourceRepo, targetRepo, true)
					// Verify is already published in target repo
					if chartExists, _ := utils.ChartExistInTargetRepo(depName, depVersion, targetRepo); chartExists {
						klog.Infof("Dependency %s-%s synced: Continuing with main chart\n", depName, depVersion)
					} else {
						klog.Infof("Dependency %s-%s not synced yet.\n", depName, depVersion)
					}
				} else {
					errs = multierror.Append(errs, errors.Errorf("Please sync %s-%s dependency first", depName, depVersion))
					missingDependencies = true
				}
			}
		} else {
			klog.Infof("Dependency %s-%s should exist in external repository %s \n", depName, depVersion, depRepository)
		}
	}

	if !missingDependencies {
		klog.V(8).Info("Updating requirements.yaml file...")
		// Update requirements.yaml file to point to target repo
		requirements, err := ioutil.ReadFile(requirementsFile)
		if err != nil {
			return errors.Annotatef(err, "Error reading %s file", requirementsFile)
		}

		deps := &Dependencies{}
		err = yaml.Unmarshal(requirements, deps)
		if err != nil {
			return errors.Annotatef(err, "Error unmarshaling %s file", requirementsFile)
		}
		for i := range deps.Dependencies {
			// Specify the exact dependencies versions used in the original requirements.lock file
			// so when running helm dep up we get the same versions resolved.
			deps.Dependencies[i].Version = dependencies[deps.Dependencies[i].Name]
			// Maybe there are dependencies from other chart repos. In this case we don't want to replace
			// the repository.
			// For example, old charts pointing to helm/charts repo
			if deps.Dependencies[i].Repository == sourceRepo.Url {
				deps.Dependencies[i].Repository = targetRepo.Url
			}
		}
		// Write updated requirements yamls file
		writeRequirementsFile(chartPath, deps)
		if err := helmcli.UpdateDependencies(chartPath); err != nil {
			return errors.Trace(err)
		}
	}
	return errs
}

// updateValuesFile performs some substitutions to a given values.yaml file
func updateValuesFile(valuesFile string, targetRepo *api.TargetRepo) error {
	if err := updateContainerImageRegistry(valuesFile, targetRepo); err != nil {
		return errors.Annotatef(err, "Error updating %s file", valuesFile)
	}
	if err := updateContainerImageRepository(valuesFile, targetRepo); err != nil {
		return errors.Annotatef(err, "Error updating %s file", valuesFile)
	}
	return nil
}

// updateContainerImageRepository updates the container repository in a values.yaml file
func updateContainerImageRepository(valuesFile string, targetRepo *api.TargetRepo) error {
	regex := regexp.MustCompile(`(?m)(repository:[[:blank:]])(.*)(/)`)
	values, err := ioutil.ReadFile(valuesFile)
	if err != nil {
		return errors.Trace(err)
	}
	submatch := regex.FindStringSubmatch(string(values))
	if len(submatch) > 0 {
		replaceLine := fmt.Sprintf("%s%s%s", submatch[1], targetRepo.ContainerRepository, submatch[3])
		newContents := regex.ReplaceAllString(string(values), replaceLine)
		err = ioutil.WriteFile(valuesFile, []byte(newContents), 0)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return errors.Trace(err)
}

// updateContainerImageRegistry updates the container registry in a values.yaml file
func updateContainerImageRegistry(valuesFile string, targetRepo *api.TargetRepo) error {
	regex := regexp.MustCompile(`(?m)(registry:[[:blank:]])(.*)(.*$)`)
	values, err := ioutil.ReadFile(valuesFile)
	if err != nil {
		return errors.Trace(err)
	}
	submatch := regex.FindStringSubmatch(string(values))
	if len(submatch) > 0 {
		replaceLine := fmt.Sprintf("%s%s%s", submatch[1], targetRepo.ContainerRegistry, submatch[3])
		newContents := regex.ReplaceAllString(string(values), replaceLine)
		err = ioutil.WriteFile(valuesFile, []byte(newContents), 0)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return errors.Trace(err)
}

// writeLock writes a lockfile to disk
func writeRequirementsFile(chartPath string, deps *Dependencies) error {
	data, err := yaml.Marshal(deps)
	if err != nil {
		return err
	}
	requirementsFileName := "requirements.yaml"
	dest := path.Join(chartPath, requirementsFileName)
	return ioutil.WriteFile(dest, data, 0644)
}
