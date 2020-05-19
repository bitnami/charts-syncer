package helmcli

import (
	"os/exec"
	"path"

	"github.com/bitnami-labs/chart-repository-syncer/api"
	"github.com/juju/errors"
	"k8s.io/klog"
)

// Package uses helm cli to package a chart and return the path to the packaged chart.
func Package(chartPath, name, version, destDir string) (string, error) {
	cmd := exec.Command("helm", "package", chartPath, "--destination", destDir)
	_, err := cmd.Output()
	if err != nil {
		return "", errors.Trace(err)
	}
	packagedChartPath := path.Join(destDir, name+"-"+version+".tgz")
	return packagedChartPath, errors.Trace(err)
}

// UpdateDependencies uses helm cli to update dependencies of a chart.
func UpdateDependencies(chartPath string) error {
	klog.V(3).Info(`Updating dependencies with "helm dependency update"`)
	cmd := exec.Command("helm", "dependency", "update", chartPath)
	if _, err := cmd.Output(); err != nil {
		return errors.Errorf("Error updading dependencies for %s", chartPath)
	}
	return nil
}

// AddRepoToHelm uses helm cli to add a repo to the helm CLI.
func AddRepoToHelm(url string, auth *api.Auth) error {
	var cmd *exec.Cmd
	if auth != nil && auth.Username != "" && auth.Password != "" {
		klog.V(3).Info("Adding target repository to helm cli with basic authentication")
		cmd = exec.Command("helm", "repo", "add", "target", url, "--username", auth.Username, "--password", auth.Password)
	} else {
		klog.V(3).Info("Adding target repository to helm cli")
		cmd = exec.Command("helm", "repo", "add", "target", url)
	}
	if _, err := cmd.Output(); err != nil {
		return errors.Annotate(err, "Error adding target repo to helm")
	}
	return nil
}
