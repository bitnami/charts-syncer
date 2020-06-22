package config

import (
	"fmt"
	"net/url"

	"github.com/bitnami-labs/chart-repository-syncer/api"
	"github.com/juju/errors"
	"github.com/spf13/viper"
)

// LoadConfig unmarshall config file into Config struct.
func LoadConfig(Config *api.Config) error {
	viper.BindEnv("source.auth.username", "SOURCE_AUTH_USERNAME")
	viper.BindEnv("source.auth.password", "SOURCE_AUTH_PASSWORD")
	viper.BindEnv("target.auth.username", "TARGET_AUTH_USERNAME")
	viper.BindEnv("target.auth.password", "TARGET_AUTH_PASSWORD")

	err := viper.Unmarshal(&Config)
	if err != nil {
		return errors.Trace(fmt.Errorf("Error unmarshalling config file: %w", err))
	}
	return nil
}

// ValidateConfig validates the config file is correct
func ValidateConfig(Config *api.Config) error {
	if _, err := url.ParseRequestURI(Config.Source.Repo.Url); err != nil {
		return errors.Errorf("Source repo URL should be a valid URL")
	}
	if _, err := url.ParseRequestURI(Config.Target.Repo.Url); err != nil {
		return errors.Errorf("Target repo URL should be a valid URL")
	}
	if Config.Target.ContainerRegistry == "" {
		return errors.Errorf("Container Registry cannot be empty")
	}
	if Config.Target.ContainerRepository == "" {
		return errors.Errorf("Container Repository cannot be empty")
	}
	switch Config.Source.Repo.Kind {
	case api.Kind_HELM.String():
	case api.Kind_CHARTMUSEUM.String():
	default:
		return errors.Errorf("Repo kind %q is not supported for source repo", Config.Source.Repo.Kind)
	}
	switch Config.Target.Repo.Kind {
	case api.Kind_HELM.String():
	case api.Kind_CHARTMUSEUM.String():
	default:
		return errors.Errorf("Repo kind %q is not supported for target repo", Config.Target.Repo.Kind)
	}
	return nil
}
