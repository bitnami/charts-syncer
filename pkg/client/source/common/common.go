// Package common provides a Wrapper implementation for remote chart sources
package common

import (
	"fmt"
	"os"

	apiv1 "github.com/bitnami/charts-syncer/gen/proto/v1"
	"github.com/bitnami/charts-syncer/pkg/client"
	"github.com/bitnami/charts-syncer/pkg/client/config"
	"github.com/vmware-labs/distribution-tooling-for-helm/cmd/dt/wrap"
)

// Source allows to operate a chart source.
type Source struct {
	client.ChartsReader
	client.ContainersReader
	username           string
	password           string
	containersUsername string
	containersPassword string
	insecure           bool
	usePlainHTTP       bool
}

// New creates a Repo object from an apiv1.Repo object.
func New(source *apiv1.Source, chartReader client.ChartsReader, insecure bool, usePlainHTTP bool) (*Source, error) {
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

// NewContainer creates a Repo object from an api.Repo object.
func NewContainer(source *apiv1.Source, containersReader client.ContainersReader, insecure bool, usePlainHTTP bool) (*Source, error) {
	containers := source.GetContainers()
	s := &Source{ContainersReader: containersReader, insecure: insecure, usePlainHTTP: usePlainHTTP}
	if containers != nil && containers.GetAuth() != nil {
		s.containersUsername = containers.GetAuth().GetUsername()
		s.containersPassword = containers.GetAuth().GetPassword()
	}
	return s, nil
}

// WrapChart wraps a chart.
func (t *Source) WrapChart(tgz, destWrap string, opts ...config.Option) (string, error) {
	cfg := config.New(opts...)
	l := cfg.Logger

	wrapWorkdir, err := os.MkdirTemp(cfg.WorkDir, "charts-syncer")

	if err != nil {
		return "", fmt.Errorf("unable to create work directory for chart: %v", err)
	}
	defer os.RemoveAll(wrapWorkdir)

	fetchArtifacts := !cfg.SkipArtifacts

	outputFile, err := wrap.Chart(tgz, wrap.WithFetchArtifacts(fetchArtifacts),
		wrap.WithSkipPullImages(cfg.SkipImages),
		wrap.WithInsecure(t.insecure), wrap.WithTempDirectory(wrapWorkdir),
		wrap.WithUsePlainHTTP(t.usePlainHTTP),
		wrap.WithAuth(t.username, t.password),
		wrap.WithPlatforms(cfg.ContainerPlatforms),
		wrap.WithContainerRegistryAuth(t.containersUsername, t.containersPassword),
		wrap.WithOutputFile(destWrap),
		wrap.WithPreserveDigest(cfg.PreserveDigest),
		wrap.WithPreservedSourceRef(cfg.PreservedSourceRef),
		wrap.WithLogger(l))
	if err != nil {
		return "", fmt.Errorf("failed to wrap chart %q: %w", tgz, err)
	}
	return outputFile, nil
}

// WrapContainer wraps a container image.
func (t *Source) WrapContainer(imageRef string, destination string, opts ...config.Option) (string, error) {
	cfg := config.New(opts...)
	l := cfg.Logger

	wrapWorkdir, err := os.MkdirTemp(cfg.WorkDir, "charts-syncer")
	if err != nil {
		return "", fmt.Errorf("unable to create work directory for container: %v", err)
	}
	defer os.RemoveAll(wrapWorkdir)

	outputFile, err := wrap.Container(imageRef,
		wrap.WithFetchArtifacts(!cfg.SkipArtifacts),
		wrap.WithSkipPullImages(cfg.SkipImages),
		wrap.WithInsecure(t.insecure),
		wrap.WithUsePlainHTTP(t.usePlainHTTP),
		wrap.WithTempDirectory(wrapWorkdir),
		wrap.WithContainerRegistryAuth(t.containersUsername, t.containersPassword),
		wrap.WithPlatforms(cfg.ContainerPlatforms),
		wrap.WithOutputFile(destination),
		wrap.WithPreserveDigest(cfg.PreserveDigest),
		wrap.WithLogger(l),
	)
	if err != nil {
		return "", fmt.Errorf("failed to wrap container %q: %w", imageRef, err)
	}
	return outputFile, nil
}
