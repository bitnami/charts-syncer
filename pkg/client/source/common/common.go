// Package common provides a Wrapper implementation for remote chart sources
package common

import (
	"fmt"
	"os"

	"github.com/bitnami/charts-syncer/api"
	"github.com/bitnami/charts-syncer/pkg/client"
	"github.com/bitnami/charts-syncer/pkg/client/config"
	"github.com/vmware-labs/distribution-tooling-for-helm/cmd/dt/wrap"
)

// Source allows to operate a chart source.
type Source struct {
	client.ChartsReader
	username           string
	password           string
	containersUsername string
	containersPassword string
	insecure           bool
	usePlainHTTP       bool
}

// New creates a Repo object from an api.Repo object.
func New(source *api.Source, chartReader client.ChartsReader, insecure bool, usePlainHTTP bool) (*Source, error) {
	containers := source.GetContainers()
	repo := source.GetRepo()
	s := &Source{ChartsReader: chartReader, insecure: insecure, usePlainHTTP: usePlainHTTP}
	if repo.GetAuth() != nil {
		s.username = repo.GetAuth().GetUsername()
		s.password = repo.GetAuth().GetPassword()
	}
	if containers != nil && containers.GetAuth() != nil {
		s.containersUsername = containers.GetAuth().GetUsername()
		s.containersPassword = containers.GetAuth().GetPassword()
	}
	return s, nil
}

// Wrap wraps a chart.
func (t *Source) Wrap(tgz, destWrap string, opts ...config.Option) (string, error) {
	cfg := config.New(opts...)
	l := cfg.Logger

	wrapWorkdir, err := os.MkdirTemp(cfg.WorkDir, "charts-syncer")

	if err != nil {
		return "", fmt.Errorf("unable to create work directory for chart: %v", err)
	}
	defer os.RemoveAll(wrapWorkdir)

	outputFile, err := wrap.Chart(tgz, wrap.WithFetchArtifacts(true),
		wrap.WithInsecure(t.insecure), wrap.WithTempDirectory(wrapWorkdir),
		wrap.WithAuth(t.username, t.password),
		wrap.WithContainerRegistryAuth(t.containersUsername, t.containersPassword),
		wrap.WithOutputFile(destWrap),
		wrap.WithLogger(l))
	if err != nil {
		return "", fmt.Errorf("failed to wrap chart %q: %w", tgz, err)
	}
	return outputFile, nil
}
