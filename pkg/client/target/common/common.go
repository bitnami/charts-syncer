// Package common provides a Unwrapper implementation for remote chart targets
package common

import (
	"context"
	"os"
	"regexp"

	apiv1 "github.com/bitnami/charts-syncer/gen/proto/v1"
	"github.com/bitnami/charts-syncer/pkg/client"
	"github.com/bitnami/charts-syncer/pkg/client/config"
	"github.com/juju/errors"
	"github.com/vmware-labs/distribution-tooling-for-helm/cmd/dt/unwrap"
	"helm.sh/helm/v3/pkg/chart"
)

var (
	schemeRE = regexp.MustCompile(`^([a-z]+)://`)
)

// Target allows to operate a remote chart target
type Target struct {
	client.ChartsReaderWriter
	client.ContainersReaderWriter
	username           string
	password           string
	containersURL      string
	containersUsername string
	containersPassword string
	insecure           bool
	usePlainHTTP       bool
}

// New creates a Repo object from an apiv1.Repo object.
func New(target *apiv1.Target, chartWriter client.ChartsReaderWriter, insecure bool, usePlainHTTP bool) (*Target, error) {
	containers := target.GetContainers()
	repo := target.GetRepo()
	s := &Target{ChartsReaderWriter: chartWriter, insecure: insecure, usePlainHTTP: usePlainHTTP}
	if repo.GetAuth() != nil {
		s.username = repo.GetAuth().GetUsername()
		s.password = repo.GetAuth().GetPassword()
	}
	if containers != nil {
		s.containersURL = containers.GetUrl()
		if containers.GetAuth() != nil {
			s.containersUsername = containers.GetAuth().GetUsername()
			s.containersPassword = containers.GetAuth().GetPassword()
		}
	}
	return s, nil
}

// NewContainer creates a Repo object from an api.Repo object.
func NewContainer(target *apiv1.Target, containersReaderWriter client.ContainersReaderWriter, insecure bool, usePlainHTTP bool) (*Target, error) {
	containers := target.GetContainers()
	s := &Target{ContainersReaderWriter: containersReaderWriter, insecure: insecure, usePlainHTTP: usePlainHTTP}
	if containers != nil {
		s.containersURL = containers.GetUrl()
		if containers.GetAuth() != nil {
			s.username = containers.GetAuth().GetUsername()
			s.password = containers.GetAuth().GetPassword()
			s.containersUsername = containers.GetAuth().GetUsername()
			s.containersPassword = containers.GetAuth().GetPassword()
		}
	}
	return s, nil
}

func (t *Target) getContainersUploadURL() string {
	containersURL := t.containersURL
	if containersURL == "" {
		// When containers URL is not specified, append /containers to charts URL
		// to avoid collision between charts and containers at the same path
		containersURL = t.GetUploadURL() + "/containers"
	}

	if schemeRE.MatchString(containersURL) {
		containersURL = schemeRE.ReplaceAllString(containersURL, "")
	}
	return containersURL
}

// UnwrapChart unwraps a chart
func (t *Target) UnwrapChart(file string, _ *chart.Metadata, opts ...config.Option) error {
	cfg := config.New(opts...)

	wrapWorkdir, err := os.MkdirTemp(cfg.WorkDir, "charts-syncer")

	if err != nil {
		return errors.Trace(err)
	}

	defer os.RemoveAll(wrapWorkdir)

	unwrapOpts := []unwrap.Option{
		unwrap.WithSayYes(true),
		unwrap.WithTempDirectory(wrapWorkdir),
		unwrap.WithUsePlainHTTP(t.usePlainHTTP),
		unwrap.WithLogger(cfg.Logger),
		unwrap.WithAuth(t.username, t.password),
		unwrap.WithInsecure(t.insecure),
		unwrap.WithContainerRegistryAuth(t.containersUsername, t.containersPassword),
		unwrap.WithSkipImageRelocation(cfg.SkipImages),
		unwrap.WithSkipPullImages(cfg.SkipImages),
		unwrap.WithPreserveRepository(false),
		unwrap.WithPreserveDigest(cfg.PreserveDigest),
	}

	if cfg.Timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel()
		unwrapOpts = append(unwrapOpts, unwrap.WithContext(ctx))
	}

	if _, err := unwrap.Chart(file, t.getContainersUploadURL(), t.GetUploadURL(), unwrapOpts...); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// UnwrapContainer unwraps a container
func (t *Target) UnwrapContainer(file string, opts ...config.Option) error {
	cfg := config.New(opts...)

	wrapWorkdir, err := os.MkdirTemp(cfg.WorkDir, "charts-syncer")

	if err != nil {
		return errors.Trace(err)
	}

	defer os.RemoveAll(wrapWorkdir)

	unwrapOpts := []unwrap.Option{
		unwrap.WithSayYes(true),
		unwrap.WithTempDirectory(wrapWorkdir),
		unwrap.WithUsePlainHTTP(t.usePlainHTTP),
		unwrap.WithLogger(cfg.Logger),
		unwrap.WithAuth(t.username, t.password),
		unwrap.WithInsecure(t.insecure),
		unwrap.WithContainerRegistryAuth(t.containersUsername, t.containersPassword),
		unwrap.WithSkipImageRelocation(cfg.SkipImages),
		unwrap.WithSkipPullImages(cfg.SkipImages),
		unwrap.WithFetchArtifacts(!cfg.SkipArtifacts),
		unwrap.WithPreserveRepository(false),
		unwrap.WithPreserveDigest(cfg.PreserveDigest),
	}

	if cfg.Timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel()
		unwrapOpts = append(unwrapOpts, unwrap.WithContext(ctx))
	}

	if _, err := unwrap.Container(file, t.getContainersUploadURL(), unwrapOpts...); err != nil {
		return errors.Trace(err)
	}
	return nil
}
