// Package common provides a Unwrapper implementation for remote chart targets
package common

import (
	"os"
	"regexp"

	"github.com/bitnami/charts-syncer/api"
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
	username           string
	password           string
	containersURL      string
	containersUsername string
	containersPassword string
	insecure           bool
	usePlainHTTP       bool
}

// New creates a Repo object from an api.Repo object.
func New(target *api.Target, chartWriter client.ChartsReaderWriter, insecure bool, usePlainHTTP bool) (*Target, error) {
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

func (t *Target) getContainersUploadURL() string {
	containersURL := t.containersURL
	if containersURL == "" {
		containersURL = t.GetUploadURL()
	}

	if schemeRE.MatchString(containersURL) {
		containersURL = schemeRE.ReplaceAllString(containersURL, "")
	}
	return containersURL
}

// Unwrap unwraps a chart
func (t *Target) Unwrap(file string, _ *chart.Metadata, opts ...config.Option) error {
	cfg := config.New(opts...)

	wrapWorkdir, err := os.MkdirTemp(cfg.WorkDir, "charts-syncer")

	if err != nil {
		return errors.Trace(err)
	}

	defer os.RemoveAll(wrapWorkdir)

	if _, err := unwrap.Chart(file, t.getContainersUploadURL(), t.GetUploadURL(), unwrap.WithSayYes(true),
		unwrap.WithTempDirectory(wrapWorkdir),
		unwrap.WithUsePlainHTTP(t.usePlainHTTP),
		unwrap.WithLogger(cfg.Logger),
		unwrap.WithAuth(t.username, t.password), unwrap.WithInsecure(t.insecure),
		unwrap.WithContainerRegistryAuth(t.containersUsername, t.containersPassword),
	); err != nil {
		return errors.Trace(err)
	}
	return nil
}
