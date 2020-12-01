package helmcli

import (
	"context"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/juju/errors"
	"k8s.io/klog"
)

const (
	helmTimeout = 1 * time.Minute
)

// Package uses helm cli to package a chart and return the path to the packaged chart.
func Package(chartPath, name, version, destDir string) (string, error) {
	cmd := exec.Command("helm", "package", chartPath, "--destination", destDir)
	_, err := cmd.Output()
	if err != nil {
		return "", errors.Errorf("error packaging chart for %s", chartPath)
	}
	packagedChartPath := path.Join(destDir, name+"-"+version+".tgz")
	return packagedChartPath, nil
}

// UpdateDependencies uses helm cli to update dependencies of a chart.
func UpdateDependencies(chartPath string) error {
	klog.V(3).Info(`Updating dependencies with "helm dependency update"`)
	// Delete the charts folder so helm dep up update packages dependencies from target repo
	if err := os.RemoveAll(path.Join(chartPath, "charts")); err != nil {
		return err
	}
	cmd := exec.Command("helm", "dependency", "update", chartPath)
	if _, err := cmd.Output(); err != nil {
		return errors.Errorf("error updading dependencies for %s", chartPath)
	}
	return nil
}

// AddRepoToHelm uses helm cli to add a repo to the helm CLI.
func AddRepoToHelm(name string, url string, auth *api.Auth) (func() error, error) {
	cleanup := func() error { return nil }
	var cmd *exec.Cmd
	if auth != nil && auth.Username != "" && auth.Password != "" {
		klog.V(3).Info("Adding target repository to helm cli with basic authentication")
		cmd = exec.Command("helm", "repo", "add", name, url, "--username", auth.Username, "--password", auth.Password)
	} else {
		klog.V(3).Info("Adding target repository to helm cli")
		cmd = exec.Command("helm", "repo", "add", name, url)
	}
	if _, err := cmd.Output(); err != nil {
		return cleanup, errors.Annotate(err, "error adding target repo to helm")
	}
	return func() error {
		return DeleteHelmRepo(name)
	}, nil
}

// DeleteHelmRepo uses helm cli to delete a repo from the helm CLI.
func DeleteHelmRepo(name string) error {
	klog.V(3).Info("Removing target repository from helm cli")
	cmd := exec.Command("helm", "repo", "remove", name)
	if _, err := cmd.Output(); err != nil {
		return errors.Annotate(err, "error removing target repo from helm")
	}
	return nil
}

// OciLogin login to OCI registry
func OciLogin(url string, auth *api.Auth) (func() error, error) {
	cleanup := func() error { return nil }
	klog.V(3).Info("Login to OCI registry")
	cmd := exec.Command("helm", "registry", "login", url, "--username", auth.Username, "--password", auth.Password)
	cmd.Env = []string{"HELM_EXPERIMENTAL_OCI=1"}
	if _, err := cmd.Output(); err != nil {
		return cleanup, errors.Annotate(err, "error login against OCI registry")
	}
	return func() error {
		return OciLogout(url)
	}, nil
}

// OciLogout logouts from OCI registry
func OciLogout(url string) error {
	klog.V(3).Info("Login out from OCI registry")
	cmd := exec.Command("helm", "registry", "logout")
	cmd.Env = []string{"HELM_EXPERIMENTAL_OCI=1"}
	if _, err := cmd.Output(); err != nil {
		return errors.Annotate(err, "error login out from OCI registry")
	}
	return nil
}

// SaveOciChart uses helm cli to save a chart to the local OCI cache
func SaveOciChart(chartPath, chartRef string) error {
	klog.V(3).Info(`Saving chart to local cache with "helm chart save"`)
	cmd := exec.Command("helm", "chart", "save", chartPath, chartRef)
	cmd.Env = []string{"HELM_EXPERIMENTAL_OCI=1"}
	if _, err := cmd.Output(); err != nil {
		return errors.Errorf("error saving chart for %s", chartPath)
	}
	return nil
}

// PushToOCI uses helm cli to push a local tgz to an OCI registry
func PushToOCI(chartRef string) error {
	ctx, cancel := context.WithTimeout(context.Background(), helmTimeout)
	defer cancel()
	klog.V(3).Info(`Pushing chart to OCI registry with "helm chart push"`)
	cmd := exec.CommandContext(ctx, "helm", "chart", "push", chartRef)
	cmd.Env = []string{"HELM_EXPERIMENTAL_OCI=1"}
	if _, err := cmd.Output(); err != nil {
		return errors.Errorf("error pushing chart for %s", chartRef)
	}
	return nil
}
