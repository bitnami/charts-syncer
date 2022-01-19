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
		inputFile string
		envVars   map[string]string
		// Helm Chart repo authentication
		expectedSourceAuth *api.Auth
		expectedTargetAuth *api.Auth
		// Container registry authentication
		expectedSourceContainerAuth *api.Containers_ContainerAuth
		expectedTargetContainerAuth *api.Containers_ContainerAuth
	}{
		"full-env-vars": {
			"example-config-no-auth.yaml",
			map[string]string{
				"SOURCE_REPO_AUTH_USERNAME":       "sUsername",
				"SOURCE_REPO_AUTH_PASSWORD":       "sPassword",
				"TARGET_REPO_AUTH_USERNAME":       "tUsername",
				"TARGET_REPO_AUTH_PASSWORD":       "tPassword",
				"SOURCE_CONTAINERS_AUTH_REGISTRY": "sRegistry",
				"SOURCE_CONTAINERS_AUTH_USERNAME": "sUsername",
				"SOURCE_CONTAINERS_AUTH_PASSWORD": "sPassword",
				"TARGET_CONTAINERS_AUTH_USERNAME": "tUsername",
				"TARGET_CONTAINERS_AUTH_PASSWORD": "tPassword",
			},
			&api.Auth{Username: "sUsername", Password: "sPassword"},
			&api.Auth{Username: "tUsername", Password: "tPassword"},
			&api.Containers_ContainerAuth{Username: "sUsername", Password: "sPassword", Registry: "sRegistry"},
			&api.Containers_ContainerAuth{Username: "tUsername", Password: "tPassword", Registry: "test.registry.io"},
		},
		"legacy-full-env-vars": {
			"example-config-no-auth.yaml",
			// Using old env variables, still compatible
			map[string]string{
				"SOURCE_AUTH_USERNAME": "sUsername",
				"SOURCE_AUTH_PASSWORD": "sPassword",
				"TARGET_AUTH_USERNAME": "tUsername",
				"TARGET_AUTH_PASSWORD": "tPassword",
			},
			&api.Auth{Username: "sUsername", Password: "sPassword"},
			&api.Auth{Username: "tUsername", Password: "tPassword"},
			nil, nil,
		},
		"full-file": {
			"example-config.yaml",
			map[string]string{},
			&api.Auth{Username: "user123", Password: "password123"},
			&api.Auth{Username: "user456", Password: "password456"},
			&api.Containers_ContainerAuth{Username: "user123", Password: "password123", Registry: "sRegistry"},
			&api.Containers_ContainerAuth{Username: "user456", Password: "password456", Registry: "test.registry.io"},
		},
		"user-file-pass-env": {
			"example-config-user-file.yaml",
			map[string]string{
				"SOURCE_REPO_AUTH_PASSWORD":       "sourcePassEnv",
				"TARGET_REPO_AUTH_PASSWORD":       "targetPassEnv",
				"SOURCE_CONTAINERS_AUTH_PASSWORD": "sPasswordEnv",
				"TARGET_CONTAINERS_AUTH_PASSWORD": "tPasswordEnv",
			},
			&api.Auth{Username: "sourceUserFile", Password: "sourcePassEnv"},
			&api.Auth{Username: "targetUserFile", Password: "targetPassEnv"},
			&api.Containers_ContainerAuth{Username: "user123", Password: "sPasswordEnv", Registry: "sRegistry"},
			&api.Containers_ContainerAuth{Username: "user456", Password: "tPasswordEnv", Registry: "test.registry.io"},
		},
		"full-file-existing-empty-env-vars": {
			"example-config.yaml",
			map[string]string{
				"SOURCE_REPO_AUTH_USERNAME":       "",
				"SOURCE_REPO_AUTH_PASSWORD":       "",
				"TARGET_REPO_AUTH_USERNAME":       "",
				"TARGET_REPO_AUTH_PASSWORD":       "",
				"SOURCE_CONTAINERS_AUTH_REGISTRY": "",
				"SOURCE_CONTAINERS_AUTH_USERNAME": "",
				"SOURCE_CONTAINERS_AUTH_PASSWORD": "",
				"TARGET_CONTAINERS_AUTH_USERNAME": "",
				"TARGET_CONTAINERS_AUTH_PASSWORD": "",
			},
			&api.Auth{Username: "user123", Password: "password123"},
			&api.Auth{Username: "user456", Password: "password456"},
			&api.Containers_ContainerAuth{Username: "user123", Password: "password123", Registry: "sRegistry"},
			&api.Containers_ContainerAuth{Username: "user456", Password: "password456", Registry: "test.registry.io"},
		},
		"overwrite-user-with-env-var": {
			"example-config.yaml",
			map[string]string{
				"SOURCE_REPO_AUTH_USERNAME":       "newSourceUserFromEnvVar",
				"TARGET_REPO_AUTH_USERNAME":       "newTargetUserFromEnvVar",
				"SOURCE_CONTAINERS_AUTH_USERNAME": "newSourceUserFromEnvVar",
				"TARGET_CONTAINERS_AUTH_USERNAME": "newSourceUserFromEnvVar",
			},
			&api.Auth{Username: "newSourceUserFromEnvVar", Password: "password123"},
			&api.Auth{Username: "newTargetUserFromEnvVar", Password: "password456"},
			&api.Containers_ContainerAuth{Username: "newSourceUserFromEnvVar", Password: "password123", Registry: "sRegistry"},
			&api.Containers_ContainerAuth{Username: "newSourceUserFromEnvVar", Password: "password456", Registry: "test.registry.io"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var syncConfig api.Config
			cfgFile := fmt.Sprintf("../../testdata/%s", tc.inputFile)
			viper.SetConfigFile(cfgFile)
			if err := InitEnvBindings(); err != nil {
				t.Fatal(err)
			}

			// This is the old method, TODO, remove once we move to viper bindings
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
			// Check Helm repository auth
			if got, want := source.GetRepo().GetAuth(), tc.expectedSourceAuth; !proto.Equal(got, want) {
				t.Errorf("got: %+v, want %+v", got, want)
			}
			if got, want := target.GetRepo().GetAuth(), tc.expectedTargetAuth; !proto.Equal(got, want) {
				t.Errorf("got: %+v, want %+v", got, want)
			}

			// Check container registry auth
			if got, want := source.GetContainers().GetAuth(), tc.expectedSourceContainerAuth; !proto.Equal(got, want) {
				t.Errorf("got: %+v, want %+v", got, want)
			}
			if got, want := target.GetContainers().GetAuth(), tc.expectedTargetContainerAuth; !proto.Equal(got, want) {
				t.Errorf("got: %+v, want %+v", got, want)
			}
		})
	}
}
