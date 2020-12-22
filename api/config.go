package api

import (
	"net/url"

	"github.com/pkg/errors"
)

// Validate validates the config file is correct
func (c *Config) Validate() error {
	if _, err := url.ParseRequestURI(c.Source.Repo.Url); err != nil {
		return errors.Errorf(`"source.repo.url" should be a valid URL: %v`, err)
	}
	switch k := c.Target.Repo.Kind; k {
	case Kind_CHARTMUSEUM, Kind_HELM, Kind_HARBOR, Kind_OCI:
		if _, err := url.ParseRequestURI(c.Target.Repo.Url); err != nil {
			return errors.Errorf(`"target.repo.url" should be a valid URL: %v`, err)
		}
	case Kind_LOCAL:
	}
	if c.Target.ContainerRegistry == "" {
		return errors.Errorf(`"target.containerRegistry" cannot be empty`)
	}
	if c.Target.ContainerRepository == "" {
		return errors.Errorf(`"target.containerRepository" cannot be empty`)
	}
	return nil
}
