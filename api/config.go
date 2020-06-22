package api

import (
	"net/url"

	"github.com/pkg/errors"
)

// Validate validates the config file is correct
func (c *Config) Validate() error {
	if _, err := url.ParseRequestURI(c.Source.Repo.Url); err != nil {
		return errors.Errorf("source repo URL should be a valid URL")
	}
	if _, err := url.ParseRequestURI(c.Target.Repo.Url); err != nil {
		return errors.Errorf("target repo URL should be a valid URL")
	}
	if c.Target.ContainerRegistry == "" {
		return errors.Errorf("container Registry cannot be empty")
	}
	if c.Target.ContainerRepository == "" {
		return errors.Errorf("container Repository cannot be empty")
	}
	return nil
}
