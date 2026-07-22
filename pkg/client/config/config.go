// Package config provides a configuration object for the client/repo package.
package config

import (
	log "github.com/vmware-labs/distribution-tooling-for-helm/pkg/dtlog"
	"github.com/vmware-labs/distribution-tooling-for-helm/pkg/dtlog/silent"
)

// Config is the configuration object for the client package
type Config struct {
	Logger             log.SectionLogger
	WorkDir            string
	ContainerPlatforms []string
	SkipArtifacts      bool
	SkipImages         bool
	PreserveDigest     bool
	PreservedSourceRef string
}

// Option is a function that modifies the Config
type Option func(*Config)

// WithWorkDir sets the workdir
func WithWorkDir(workdir string) func(*Config) {
	return func(c *Config) {
		c.WorkDir = workdir
	}
}

// WithSkipArtifacts sets the skip artifacts flag
func WithSkipArtifacts(skipArtifacts bool) func(*Config) {
	return func(c *Config) {
		c.SkipArtifacts = skipArtifacts
	}
}

// WithSkipImages sets the skip image flag
func WithSkipImages(skipImages bool) func(*Config) {
	return func(c *Config) {
		c.SkipImages = skipImages
	}
}

// WithContainerPlatforms sets the container platforms to sync
func WithContainerPlatforms(containerPlatforms []string) func(*Config) {
	return func(c *Config) {
		c.ContainerPlatforms = containerPlatforms
	}
}

// WithPreserveDigest sets the preserve digest flag
func WithPreserveDigest(preserveDigest bool) func(*Config) {
	return func(c *Config) {
		c.PreserveDigest = preserveDigest
	}
}

// WithPreservedSourceRef sets the original oci:// reference of the chart
// being wrapped, so PreserveDigest can capture the pristine source manifest
// even though the chart was already fetched to a local .tgz for indexing
func WithPreservedSourceRef(ref string) func(*Config) {
	return func(c *Config) {
		c.PreservedSourceRef = ref
	}
}

// WithLogger sets the logger
func WithLogger(logger log.SectionLogger) func(*Config) {
	return func(c *Config) {
		c.Logger = logger
	}
}

// New creates a new Config object
func New(options ...Option) *Config {
	c := &Config{
		Logger: silent.NewSectionLogger(),
	}
	for _, option := range options {
		option(c)
	}
	return c
}
