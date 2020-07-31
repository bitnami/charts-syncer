package config

import (
	"fmt"
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
	tests := map[string]struct {
		inputFile string
		envVars   map[string]string
		expected  map[string]string
	}{
		"full-env-vars": {
			"example-config-no-auth.yaml",
			map[string]string{
				"SOURCE_AUTH_USERNAME": "sUsername",
				"SOURCE_AUTH_PASSWORD": "sPassword",
				"TARGET_AUTH_USERNAME": "tUsername",
				"TARGET_AUTH_PASSWORD": "tPassword",
			},
			map[string]string{
				"su": "sUsername",
				"sp": "sPassword",
				"tu": "tUsername",
				"tp": "tPassword",
			},
		},
		"full-file": {
			"example-config.yaml",
			map[string]string{},
			map[string]string{
				"su": "user123",
				"sp": "password123",
				"tu": "user456",
				"tp": "password456",
			},
		},
		"user-file-pass-env": {
			"example-config-user-file.yaml",
			map[string]string{
				"SOURCE_AUTH_PASSWORD": "sourcePassEnv",
				"TARGET_AUTH_PASSWORD": "targetPassEnv",
			},
			map[string]string{
				"su": "sourceUserFile",
				"sp": "sourcePassEnv",
				"tu": "targetUserFile",
				"tp": "targetPassEnv",
			},
		},
		"full-file-existing-empty-env-vars": {
			"example-config.yaml",
			map[string]string{
				"SOURCE_AUTH_USERNAME": "",
				"SOURCE_AUTH_PASSWORD": "",
				"TARGET_AUTH_USERNAME": "",
				"TARGET_AUTH_PASSWORD": "",
			},
			map[string]string{
				"su": "user123",
				"sp": "password123",
				"tu": "user456",
				"tp": "password456",
			},
		},
		"no-overwrite-user-with-env-var": {
			"example-config.yaml",
			map[string]string{
				"SOURCE_AUTH_USERNAME": "newSourceUserFromEnvVar",
				"TARGET_AUTH_USERNAME": "newTargetUserFromEnvVar",
			},
			map[string]string{
				"su": "user123",
				"sp": "password123",
				"tu": "user456",
				"tp": "password456",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var syncConfig api.Config
			cfgFile := fmt.Sprintf("../../testdata/%s", tc.inputFile)
			viper.SetConfigFile(cfgFile)
			for k, v := range tc.envVars {
				os.Setenv(k, v)
			}
			// If a config file is found, read it in.
			if err := viper.ReadInConfig(); err != nil {
				t.Fatalf("error reading config file: %+v", err)
			}
			if err := Load(&syncConfig); err != nil {
				t.Fatalf("error loading config file")
			}
			source := syncConfig.Source
			target := syncConfig.Target
			for k := range tc.envVars {
				os.Unsetenv(k)
			}
			if source.Repo.Auth.Username != tc.expected["su"] {
				t.Errorf("Got: %s, want %s", source.Repo.Auth.Username, tc.expected["su"])
			}
			if source.Repo.Auth.Password != tc.expected["sp"] {
				t.Errorf("Got: %s, want %s", source.Repo.Auth.Password, tc.expected["sp"])
			}
			if target.Repo.Auth.Username != tc.expected["tu"] {
				t.Errorf("Got: %s, want %s", target.Repo.Auth.Username, tc.expected["tu"])
			}
			if target.Repo.Auth.Password != tc.expected["tp"] {
				t.Errorf("Got: %s, want %s", target.Repo.Auth.Password, tc.expected["tp"])
			}

		})
	}
}
