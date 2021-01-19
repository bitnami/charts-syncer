package helm

import (
	"context"
	"os/exec"
	"time"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/juju/errors"
	"k8s.io/klog"
)

const (
	helmTimeout = 1 * time.Minute
)

type ociCli interface {
	OciLogin(url string, auth *api.Auth) (func() error, error)
	OciLogout(url string) error
	SaveOciChart(chartPath, chartRef string) error
	PushToOci(chartRef string) error
}

// Cli structure with Helm cli methods
type Cli struct{}

// OciLogin login to OCI registry
func (c Cli) OciLogin(url string, auth *api.Auth) (func() error, error) {
	cleanup := func() error { return nil }
	klog.V(3).Info("Login to OCI registry")
	cmd := exec.Command("helm", "registry", "login", url, "--username", auth.Username, "--password", auth.Password)
	cmd.Env = []string{"HELM_EXPERIMENTAL_OCI=1"}
	if _, err := cmd.Output(); err != nil {
		return cleanup, errors.Annotate(err, "error login against OCI registry")
	}
	return func() error {
		return c.OciLogout(url)
	}, nil
}

// OciLogout logouts from OCI registry
func (c Cli) OciLogout(url string) error {
	klog.V(3).Info("Login out from OCI registry")
	cmd := exec.Command("helm", "registry", "logout")
	cmd.Env = []string{"HELM_EXPERIMENTAL_OCI=1"}
	if _, err := cmd.Output(); err != nil {
		return errors.Annotate(err, "error login out from OCI registry")
	}
	return nil
}

// SaveOciChart uses helm cli to save a chart to the local OCI cache
func (c Cli) SaveOciChart(chartPath, chartRef string) error {
	klog.V(3).Info(`Saving chart to local cache with "helm chart save"`)
	cmd := exec.Command("helm", "chart", "save", chartPath, chartRef)
	cmd.Env = []string{"HELM_EXPERIMENTAL_OCI=1"}
	if _, err := cmd.Output(); err != nil {
		return errors.Errorf("error saving chart for %s", chartPath)
	}
	return nil
}

// PushToOCI uses helm cli to push a local tgz to an OCI registry
func (c Cli) PushToOCI(chartRef string) error {
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
