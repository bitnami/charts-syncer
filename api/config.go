package api

import (
	"net/url"

	"github.com/pkg/errors"
)

// Validate validates the config file is correct
func (c *Config) Validate() error {
	if c.GetSource().GetRepo() != nil {
		if _, err := url.ParseRequestURI(c.GetSource().GetRepo().GetUrl()); err != nil {
			return errors.Errorf(`"source.repo.url" should be a valid URL: %v`, err)
		}
	}
	if c.GetTarget().GetRepo() != nil {
		switch k := c.GetTarget().GetRepo().GetKind(); k {
		case Kind_CHARTMUSEUM, Kind_HELM, Kind_HARBOR, Kind_OCI:
			if _, err := url.ParseRequestURI(c.GetTarget().GetRepo().GetUrl()); err != nil {
				return errors.Errorf(`"target.repo.url" should be a valid URL: %v`, err)
			}
		case Kind_LOCAL:
		}
		if c.GetTarget().GetContainerRegistry() == "" {
			return errors.Errorf(`"target.containerRegistry" cannot be empty`)
		}
		if c.GetTarget().GetContainerRepository() == "" {
			return errors.Errorf(`"target.containerRepository" cannot be empty`)
		}
	}

	return nil
}
