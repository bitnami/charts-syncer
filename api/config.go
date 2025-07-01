package api

import (
	"net/url"

	"github.com/pkg/errors"
)

// Validate validates the config file is correct
func (c *Config) Validate() error {
	// Ensure backward compatibility: if using old 'source' field, migrate it to 'sources'
	if err := c.migrateToMultipleSources(); err != nil {
		return err
	}

	// Validate that we have at least one source
	sources := c.GetEffectiveSources()
	if len(sources) == 0 {
		return errors.Errorf("at least one source must be specified")
	}

	// Validate each source
	for i, source := range sources {
		if err := c.validateSource(source, i); err != nil {
			return err
		}
	}

	// Validate target
	if repo := c.GetTarget().GetRepo(); repo != nil {
		switch k := repo.GetKind(); k {
		case Kind_CHARTMUSEUM, Kind_HELM, Kind_HARBOR, Kind_OCI:
			if _, err := url.ParseRequestURI(repo.GetUrl()); err != nil {
				return errors.Errorf(`"target.repo.url" should be a valid URL: %v`, err)
			}
		}
	}

	// Authentication
	// Container images
	for i, source := range sources {
		if auth := source.GetContainers().GetAuth(); auth != nil {
			if auth.Username == "" || auth.Password == "" || auth.Registry == "" {
				return errors.Errorf(`"sources[%d].containers.auth" "registry", "username" and "password" are required"`, i)
			}
		}
	}

	if auth := c.GetTarget().GetContainers().GetAuth(); auth != nil {
		// NOTE: we do not indicate that the registry is empty because this one is set from target.containerRegistry
		// so the user does not need to set it up
		if auth.Username == "" || auth.Password == "" {
			return errors.Errorf(`"target.containers.auth" "username" and "password" are required"`)
		}
	}
	if repo := c.GetTarget().GetRepo(); repo != nil {
		if repo.GetKind() != Kind_OCI && repo.GetKind() != Kind_LOCAL {
			return errors.Errorf(`"target.repo.kind" should be "OCI" or "LOCAL"`)
		}
	}

	return nil
}

// migrateToMultipleSources migrates the old single source configuration to the new multiple sources format
func (c *Config) migrateToMultipleSources() error {
	if c.GetSource() != nil && len(c.GetSources()) > 0 {
		return errors.Errorf(`cannot use both "source" and "sources" fields. Please use only "sources" for multiple source support`)
	}

	// If using the old format, migrate to new format
	if c.GetSource() != nil && len(c.GetSources()) == 0 {
		c.Sources = []*Source{c.GetSource()}
		// Keep the old source field for backward compatibility but mark it as migrated
	}

	return nil
}

// GetEffectiveSources returns the list of sources, handling backward compatibility
func (c *Config) GetEffectiveSources() []*Source {
	if len(c.GetSources()) > 0 {
		return c.GetSources()
	}
	if c.GetSource() != nil {
		return []*Source{c.GetSource()}
	}
	return nil
}

// validateSource validates a single source configuration
func (c *Config) validateSource(source *Source, index int) error {
	if repo := source.GetRepo(); repo != nil {
		switch k := repo.GetKind(); k {
		case Kind_CHARTMUSEUM, Kind_HELM, Kind_HARBOR, Kind_OCI:
			if _, err := url.ParseRequestURI(repo.GetUrl()); err != nil {
				return errors.Errorf(`"sources[%d].repo.url" should be a valid URL: %v`, index, err)
			}
		}
	}

	// Validate that charts and skip_charts are not both specified for this source
	if len(source.GetCharts()) > 0 && len(source.GetSkipCharts()) > 0 {
		return errors.Errorf(`"sources[%d].charts" and "sources[%d].skip_charts" properties cannot be set at the same time`, index, index)
	}

	return nil
}

// GetEffectiveChartsForSource returns the effective charts list for a specific source,
// considering both source-specific and global chart configurations
func (c *Config) GetEffectiveChartsForSource(source *Source) []string {
	// If source has specific charts defined, use those
	if len(source.GetCharts()) > 0 {
		return source.GetCharts()
	}

	// Fall back to global charts if no source-specific charts
	return c.GetCharts()
}

// GetEffectiveSkipChartsForSource returns the effective skip charts list for a specific source,
// considering both source-specific and global skip chart configurations
func (c *Config) GetEffectiveSkipChartsForSource(source *Source) []string {
	// If source has specific skip_charts defined, use those
	if len(source.GetSkipCharts()) > 0 {
		return source.GetSkipCharts()
	}

	// Fall back to global skip_charts if no source-specific skip_charts
	return c.GetSkipCharts()
}
