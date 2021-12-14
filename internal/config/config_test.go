package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/proto"
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
	if source.GetRepo().GetKind() != api.Kind_HELM {
		t.Errorf("got: %s, want %s", source.GetRepo().GetKind(), "HELM")
	}
	if target.GetRepo().GetKind() != api.Kind_CHARTMUSEUM {
		t.Errorf("got: %s, want %s", target.GetRepo().GetKind(), "CHARTMUSEUM")
	}
	if target.GetContainerRegistry() != "test.registry.io" {
		t.Errorf("got: %s, want %s", target.GetContainerRegistry(), "test.registry.io")
	}
	if target.GetContainerRepository() != "user/demo" {
		t.Errorf("got: %s, want %s", target.GetContainerRepository(), "user/demo")
	}
}

// Get auth properties from env vars
func TestGetAuthFromEnvVar(t *testing.T) {
	tests := map[string]struct {
		inputFile          string
		envVars            map[string]string
		expectedSourceAuth *api.Auth
		expectedTargetAuth *api.Auth
	}{
		"full-env-vars": {
			"example-config-no-auth.yaml",
			map[string]string{
				"SOURCE_AUTH_USERNAME": "sUsername",
				"SOURCE_AUTH_PASSWORD": "sPassword",
				"TARGET_AUTH_USERNAME": "tUsername",
				"TARGET_AUTH_PASSWORD": "tPassword",
			},
			&api.Auth{Username: "sUsername", Password: "sPassword"},
			&api.Auth{Username: "tUsername", Password: "tPassword"},
		},
		"full-file": {
			"example-config.yaml",
			map[string]string{},
			&api.Auth{Username: "user123", Password: "password123"},
			&api.Auth{Username: "user456", Password: "password456"},
		},
		"user-file-pass-env": {
			"example-config-user-file.yaml",
			map[string]string{
				"SOURCE_AUTH_PASSWORD": "sourcePassEnv",
				"TARGET_AUTH_PASSWORD": "targetPassEnv",
			},
			&api.Auth{Username: "sourceUserFile", Password: "sourcePassEnv"},
			&api.Auth{Username: "targetUserFile", Password: "targetPassEnv"},
		},
		"full-file-existing-empty-env-vars": {
			"example-config.yaml",
			map[string]string{
				"SOURCE_AUTH_USERNAME": "",
				"SOURCE_AUTH_PASSWORD": "",
				"TARGET_AUTH_USERNAME": "",
				"TARGET_AUTH_PASSWORD": "",
			},
			&api.Auth{Username: "user123", Password: "password123"},
			&api.Auth{Username: "user456", Password: "password456"},
		},
		"overwrite-user-with-env-var": {
			"example-config.yaml",
			map[string]string{
				"SOURCE_AUTH_USERNAME": "newSourceUserFromEnvVar",
				"TARGET_AUTH_USERNAME": "newTargetUserFromEnvVar",
			},
			&api.Auth{Username: "newSourceUserFromEnvVar", Password: "password123"},
			&api.Auth{Username: "newTargetUserFromEnvVar", Password: "password456"},
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
			if got, want := source.GetRepo().GetAuth(), tc.expectedSourceAuth; !proto.Equal(got, want) {
				t.Errorf("got: %+v, want %+v", got, want)
			}
			if got, want := target.GetRepo().GetAuth(), tc.expectedTargetAuth; !proto.Equal(got, want) {
				t.Errorf("got: %+v, want %+v", got, want)
			}
		})
	}
}
