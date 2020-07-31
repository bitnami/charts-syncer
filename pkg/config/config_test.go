package config

import (
	"os"
	"testing"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/spf13/viper"
)

// Load unmarshall config file into Config struct
func TestLoad(t *testing.T) {
	var syncConfig api.Config
	cfgFile := "../../testdata/example-config.yaml"
	viper.SetConfigFile(cfgFile)
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("error reading config file: %+v", err)
	}
	if err := Load(&syncConfig); err != nil {
		t.Fatalf("error loading config file")
	}
	source := syncConfig.Source
	target := syncConfig.Target
	if source.Repo.Kind != api.Kind_HELM {
		t.Errorf("Got: %s, want %s", source.Repo.Kind, "HELM")
	}
	if target.Repo.Kind != api.Kind_CHARTMUSEUM {
		t.Errorf("Got: %s, want %s", target.Repo.Kind, "CHARTMUSEUM")
	}
	if target.ContainerRegistry != "test.registry.io" {
		t.Errorf("Got: %s, want %s", target.ContainerRegistry, "test.registry.io")
	}
	if target.ContainerRepository != "user/demo" {
		t.Errorf("Got: %s, want %s", target.ContainerRepository, "user/demo")
	}
}

// Get auth properties from env vars
func TestGetAuthFromEnvVar(t *testing.T) {
	var syncConfig api.Config
	cfgFile := "../../testdata/example-config-no-auth.yaml"
	viper.SetConfigFile(cfgFile)
	os.Setenv("SOURCE_AUTH_USERNAME", "sUsername")
	os.Setenv("SOURCE_AUTH_PASSWORD", "sPassword")
	os.Setenv("TARGET_AUTH_USERNAME", "tUsername")
	os.Setenv("TARGET_AUTH_PASSWORD", "tPassword")
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("error reading config file: %+v", err)
	}
	if err := Load(&syncConfig); err != nil {
		t.Fatalf("error loading config file")
	}
	source := syncConfig.Source
	target := syncConfig.Target
	if source.Repo.Auth.Username != "sUsername" {
		t.Errorf("Got: %s, want %s", source.Repo.Auth.Username, "sUsername")
	}
	if source.Repo.Auth.Password != "sPassword" {
		t.Errorf("Got: %s, want %s", source.Repo.Auth.Password, "sPassword")
	}
	if target.Repo.Auth.Username != "tUsername" {
		t.Errorf("Got: %s, want %s", target.Repo.Auth.Username, "tUsername")
	}
	if target.Repo.Auth.Password != "tPassword" {
		t.Errorf("Got: %s, want %s", target.Repo.Auth.Password, "tPassword")
	}
}
