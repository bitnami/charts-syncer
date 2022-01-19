package api

import (
	"net/url"

	"github.com/pkg/errors"
)

// Validate validates the config file is correct
func (c *Config) Validate() error {
	if repo := c.GetSource().GetRepo(); repo != nil {
		if _, err := url.ParseRequestURI(repo.GetUrl()); err != nil {
			return errors.Errorf(`"source.repo.url" should be a valid URL: %v`, err)
		}
	}
	if repo := c.GetTarget().GetRepo(); repo != nil {
		switch k := repo.GetKind(); k {
		case Kind_CHARTMUSEUM, Kind_HELM, Kind_HARBOR, Kind_OCI:
			if _, err := url.ParseRequestURI(repo.GetUrl()); err != nil {
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

	// Authentication
	// Container images
	if auth := c.GetSource().GetContainers().GetAuth(); auth != nil {
		if auth.Username == "" || auth.Password == "" || auth.Registry == "" {
			return errors.Errorf(`"source.containers.auth" "registry", "username"" and "password" are required"`)
		}
	}
	if auth := c.GetTarget().GetContainers().GetAuth(); auth != nil {
		// NOTE: we do not indicate that the registry is empty because this one is set from target.containerRegistry
		// so the user does not need to set it up
		if auth.Username == "" || auth.Password == "" {
			return errors.Errorf(`"target.containers.auth" "username"" and "password" are required"`)
		}
	}

	return nil
}
