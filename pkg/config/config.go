package config

import (
	"fmt"

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
