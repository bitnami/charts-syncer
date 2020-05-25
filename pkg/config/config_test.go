package config

import (
	"testing"

	"github.com/bitnami-labs/chart-repository-syncer/api"
	"github.com/spf13/viper"
)

// LoadConfig unmarshall config file into Config struct
func TestLoadConfig(t *testing.T) {
	var syncConfig api.Config
	cfgFile := "../../testdata/example-config.yaml"
	viper.SetConfigFile(cfgFile)
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		t.Errorf("Error reading config file: %+v", err)
	}
	if err := LoadConfig(&syncConfig); err != nil {
		t.Errorf("Error loading config file")
	}
	source := syncConfig.Source
	target := syncConfig.Target
	if source.Repo.Kind != "HELM" {
		t.Errorf("Got: %s, want %s", source.Repo.Kind, "HELM")
	}
	if target.Repo.Kind != "CHARTMUSEUM" {
		t.Errorf("Got: %s, want %s", target.Repo.Kind, "CHARTMUSEUM")
	}
	if target.ContainerRegistry != "test.registry.io" {
		t.Errorf("Got: %s, want %s", target.ContainerRegistry, "test.registry.io")
	}
	if target.ContainerRepository != "user/demo" {
		t.Errorf("Got: %s, want %s", target.ContainerRepository, "user/demo")
	}
}
