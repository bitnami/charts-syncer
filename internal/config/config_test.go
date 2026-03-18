package config

import (
	"fmt"
	"os"
	"testing"

	apiv1 "github.com/bitnami/charts-syncer/gen/proto/v1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

// Load unmarshall config file into Config struct
func TestLoad(t *testing.T) {
	var syncConfig apiv1.Config
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
	if source.GetRepo().GetKind() != apiv1.Kind_HELM {
		t.Errorf("got: %s, want %s", source.GetRepo().GetKind(), "HELM")
	}
	if target.GetRepo().GetKind() != apiv1.Kind_CHARTMUSEUM {
		t.Errorf("got: %s, want %s", target.GetRepo().GetKind(), "CHARTMUSEUM")
	}
}

// Get auth properties from env vars
func TestGetAuthFromEnvVar(t *testing.T) {
	tests := map[string]struct {
		inputFile string
		envVars   map[string]string
		// Helm Chart repo authentication
		expectedSourceAuth *apiv1.Auth
		expectedTargetAuth *apiv1.Auth
		// Container registry authentication
		expectedSourceContainerAuth *apiv1.Containers_ContainerAuth
		expectedTargetContainerAuth *apiv1.Containers_ContainerAuth
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
			&apiv1.Auth{Username: "sUsername", Password: "sPassword"},
			&apiv1.Auth{Username: "tUsername", Password: "tPassword"},
			&apiv1.Containers_ContainerAuth{Username: "sUsername", Password: "sPassword", Registry: "sRegistry"},
			&apiv1.Containers_ContainerAuth{Username: "tUsername", Password: "tPassword"},
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
			&apiv1.Auth{Username: "sUsername", Password: "sPassword"},
			&apiv1.Auth{Username: "tUsername", Password: "tPassword"},
			nil, nil,
		},
		"full-file": {
			"example-config.yaml",
			map[string]string{},
			&apiv1.Auth{Username: "user123", Password: "password123"},
			&apiv1.Auth{Username: "user456", Password: "password456"},
			&apiv1.Containers_ContainerAuth{Username: "user123", Password: "password123", Registry: "sRegistry"},
			&apiv1.Containers_ContainerAuth{Username: "user456", Password: "password456", Registry: "test.registry.io"},
		},
		"user-file-pass-env": {
			"example-config-user-file.yaml",
			map[string]string{
				"SOURCE_REPO_AUTH_PASSWORD":       "sourcePassEnv",
				"TARGET_REPO_AUTH_PASSWORD":       "targetPassEnv",
				"SOURCE_CONTAINERS_AUTH_PASSWORD": "sPasswordEnv",
				"TARGET_CONTAINERS_AUTH_PASSWORD": "tPasswordEnv",
			},
			&apiv1.Auth{Username: "sourceUserFile", Password: "sourcePassEnv"},
			&apiv1.Auth{Username: "targetUserFile", Password: "targetPassEnv"},
			&apiv1.Containers_ContainerAuth{Username: "user123", Password: "sPasswordEnv", Registry: "sRegistry"},
			&apiv1.Containers_ContainerAuth{Username: "user456", Password: "tPasswordEnv", Registry: "test.registry.io"},
		},
		"full-file-existing-empty-env-vars": {
			"example-config.yaml",
			map[string]string{
				"SOURCE_REPO_AUTH_USERNAME":       "",
				"SOURCE_REPO_AUTH_PASSWORD":       "",
				"TARGET_REPO_AUTH_USERNAME":       "",
				"TARGET_REPO_AUTH_PASSWORD":       "",
				"SOURCE_CONTAINERS_AUTH_USERNAME": "",
				"SOURCE_CONTAINERS_AUTH_PASSWORD": "",
				"TARGET_CONTAINERS_AUTH_USERNAME": "",
				"TARGET_CONTAINERS_AUTH_PASSWORD": "",
			},
			&apiv1.Auth{Username: "user123", Password: "password123"},
			&apiv1.Auth{Username: "user456", Password: "password456"},
			&apiv1.Containers_ContainerAuth{Username: "user123", Password: "password123", Registry: "sRegistry"},
			&apiv1.Containers_ContainerAuth{Username: "user456", Password: "password456", Registry: "test.registry.io"},
		},
		"overwrite-user-with-env-var": {
			"example-config.yaml",
			map[string]string{
				"SOURCE_REPO_AUTH_USERNAME":       "newSourceUserFromEnvVar",
				"TARGET_REPO_AUTH_USERNAME":       "newTargetUserFromEnvVar",
				"SOURCE_CONTAINERS_AUTH_USERNAME": "newSourceUserFromEnvVar",
				"TARGET_CONTAINERS_AUTH_USERNAME": "newSourceUserFromEnvVar",
			},
			&apiv1.Auth{Username: "newSourceUserFromEnvVar", Password: "password123"},
			&apiv1.Auth{Username: "newTargetUserFromEnvVar", Password: "password456"},
			&apiv1.Containers_ContainerAuth{Username: "newSourceUserFromEnvVar", Password: "password123", Registry: "sRegistry"},
			&apiv1.Containers_ContainerAuth{Username: "newSourceUserFromEnvVar", Password: "password456", Registry: "test.registry.io"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var syncConfig apiv1.Config
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

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		config   *apiv1.Config
		wantErrs []string
	}{
		{
			name: "Should validate a proper config",
			config: &apiv1.Config{
				Source: &apiv1.Source{
					Repo: &apiv1.Repo{
						Url:  "https://fake.source.com",
						Kind: apiv1.Kind_OCI,
						Auth: &apiv1.Auth{
							Username: "user",
							Password: "password",
						},
					},
				},
				Target: &apiv1.Target{
					Repo: &apiv1.Repo{
						Url:  "https://fake.source.com",
						Kind: apiv1.Kind_OCI,
						Auth: &apiv1.Auth{
							Username: "user",
							Password: "password",
						},
					},
				},
			},
			wantErrs: nil,
		},
		{
			name: "Should complain about wrong source url",
			config: &apiv1.Config{
				Source: &apiv1.Source{
					Repo: &apiv1.Repo{
						Url:  "ht//:fake.source.com",
						Kind: apiv1.Kind_CHARTMUSEUM,
						Auth: &apiv1.Auth{
							Username: "user",
							Password: "password",
						},
					},
				},
			},
			wantErrs: []string{"\"source.repo.url\" should be a valid URL"},
		},
		{
			name: "Should complain about wrong target kind",
			config: &apiv1.Config{
				Source: &apiv1.Source{
					Repo: &apiv1.Repo{
						Url:  "https://fake.source.com",
						Kind: apiv1.Kind_CHARTMUSEUM,
						Auth: &apiv1.Auth{
							Username: "user",
							Password: "password",
						},
					},
				},
				Target: &apiv1.Target{
					Repo: &apiv1.Repo{
						Url:  "https://fake.source.com",
						Kind: apiv1.Kind_CHARTMUSEUM,
						Auth: &apiv1.Auth{
							Username: "user",
							Password: "password",
						},
					},
				},
			},
			wantErrs: []string{"\"target.repo.kind\" should be \"OCI\" or \"LOCAL\""},
		},
		{
			name: "Should complain about wrong source url and wrong target kind",
			config: &apiv1.Config{
				Source: &apiv1.Source{
					Repo: &apiv1.Repo{
						Url:  "ht//:fake.source.com",
						Kind: apiv1.Kind_CHARTMUSEUM,
						Auth: &apiv1.Auth{
							Username: "user",
							Password: "password",
						},
					},
				},
				Target: &apiv1.Target{
					Repo: &apiv1.Repo{
						Url:  "https://fake.source.com",
						Kind: apiv1.Kind_CHARTMUSEUM,
						Auth: &apiv1.Auth{
							Username: "user",
							Password: "password",
						},
					},
				},
			},
			wantErrs: []string{
				"\"source.repo.url\" should be a valid URL",
				"\"target.repo.kind\" should be \"OCI\" or \"LOCAL\"",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateChartConfig(tt.config)
			if len(tt.wantErrs) > 0 && err != nil {
				for _, wantErr := range tt.wantErrs {
					assert.Contains(t, err.Error(), wantErr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
