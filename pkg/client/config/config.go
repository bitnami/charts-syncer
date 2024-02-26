// Package config provides a configuration object for the client/repo package.
package config

import (
	"github.com/vmware-labs/distribution-tooling-for-helm/pkg/log"
	"github.com/vmware-labs/distribution-tooling-for-helm/pkg/log/silent"
)

// Config is the configuration object for the client package
type Config struct {
	Logger             log.SectionLogger
	WorkDir            string
	ContainerPlatforms []string
	SkipArtifacts      bool
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

// WithContainerPlatforms sets the container platforms to sync
func WithContainerPlatforms(containerPlatforms []string) func(*Config) {
	return func(c *Config) {
		c.ContainerPlatforms = containerPlatforms
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
	c := &Config{Logger: silent.NewSectionLogger()}
	for _, option := range options {
		option(c)
	}
	return c
}
