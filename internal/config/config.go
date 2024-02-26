// Package config handles the configuration passed to the chart-syncer tool
package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/bitnami/charts-syncer/api"
	"github.com/juju/errors"
	"github.com/spf13/viper"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"k8s.io/klog"
	"sigs.k8s.io/yaml"
)

// DefaultIndexName is the name for the OCI artifact with the index
const DefaultIndexName = "charts-index"

// DefaultIndexTag is the tag for the OCI artifact with the index
const DefaultIndexTag = "latest"

func setDefaultChartsIndex(config *api.Config) error {
	u, err := url.Parse(config.GetSource().GetRepo().GetUrl())
	if err != nil {
		return err
	}

	uri := strings.Trim(strings.Join([]string{u.Host, u.Path}, "/"), "/")
	ref := fmt.Sprintf("%s/%s:%s", uri, DefaultIndexName, DefaultIndexTag)
	klog.V(4).Infof("'source.repo.chartsIndex' property is empty. Using %q default value", ref)
	config.GetSource().GetRepo().ChartsIndex = ref

	return nil
}

// Load unmarshall config file into Config struct.
func Load(config *api.Config) error {
	// Load the config file
	if err := yamlToProto(viper.ConfigFileUsed(), config); err != nil {
		return errors.Trace(fmt.Errorf("error unmarshalling config file: %w", err))
	}

	if err := setDefaultOverrides(config); err != nil {
		return errors.Trace(err)
	}

	if len(config.GetCharts()) > 0 && len(config.GetSkipCharts()) > 0 {
		return errors.New("\"charts\" and \"skipCharts\" properties can not be set at the same time")
	}

	return nil
}

func setDefaultOverrides(config *api.Config) error {
	if repo := config.GetSource().GetRepo(); repo != nil {
		if !repo.GetDisableChartsIndex() && repo.GetChartsIndex() == "" {
			if err := setDefaultChartsIndex(config); err != nil {
				return err
			}
		}
	}

	// Target OCI Chart repositories do not use the custom index
	if repo := config.GetTarget().GetRepo(); repo != nil {
		if repo.Kind == api.Kind_OCI {
			repo.DisableChartsIndex = true
		}
	}

	// Container registry authentication override
	if err := setAuthentication(config.GetSource(), config.GetTarget()); err != nil {
		return err
	}

	return nil
}

// Sets the authentication configuration for container images and Helm Chart repositories
// It reads the configuration from the viper config repository which values might come from the config file, env vars or flags
func setAuthentication(source *api.Source, target *api.Target) error {
	// Source Chart and container images authentication
	if source != nil {
		// Helm Chart authentication
		// NOTE: Getting entries one by one is required since they match the env variables defined and being overridden i.e SOURCE_containers.auth_REGISTRY
		username, password := viper.GetString("source.repo.auth.username"), viper.GetString("source.repo.auth.password")
		if username != "" && password != "" && source.GetRepo() != nil {
			source.GetRepo().Auth = &api.Auth{Username: username, Password: password}
		}

		// Container images OCI repository authentication
		username, password, registry := viper.GetString("source.containers.auth.username"), viper.GetString("source.containers.auth.password"), viper.GetString("source.containers.auth.registry")
		// Validation will happen in a later stage config.Validate()
		// For now we set the struct value if any of the properties is available
		if username != "" || password != "" {
			if registry == "" {
				registry = viper.GetString("source.containers.url")
			}
			if source.GetContainers() == nil {
				source.Containers = &api.Containers{}
			}
			source.GetContainers().Auth = &api.Containers_ContainerAuth{Username: username, Password: password, Registry: registry}
		}
	}

	// Target Chart and container images authentication
	if target != nil {
		username, password := viper.GetString("target.repo.auth.username"), viper.GetString("target.repo.auth.password")
		if username != "" && password != "" && target.GetRepo() != nil {
			target.GetRepo().Auth = &api.Auth{Username: username, Password: password}
		}

		// Target container images OCI repository
		username, password, registry := viper.GetString("target.containers.auth.username"), viper.GetString("target.containers.auth.password"), viper.GetString("target.containers.auth.registry")
		if username != "" || password != "" {
			if registry == "" {
				registry = viper.GetString("target.containers.url")
			}
			if target.GetContainers() == nil {
				target.Containers = &api.Containers{}
			}
			target.GetContainers().Auth = &api.Containers_ContainerAuth{Username: username, Password: password, Registry: registry}
		}
	}

	return nil
}

// yamlToProto unmarshals `path` into the provided proto message
func yamlToProto(path string, v proto.Message) error {
	yamlBytes, err := os.ReadFile(path)
	if err != nil {
		return errors.Trace(err)
	}
	jsonBytes, err := yaml.YAMLToJSONStrict(yamlBytes)
	if err != nil {
		return errors.Trace(err)
	}
	err = protojson.Unmarshal(jsonBytes, v)
	return errors.Trace(err)
}

// InitEnvBindings defines the env variables bindings associated with local viper keys
func InitEnvBindings() error {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	boundKeys := []struct {
		// viper key associated with the env variable
		key string
		// name for the env variable in addition to the default one
		// i.e source.containers.auth.registry => SOURCE_CONTAINERS_AUTH_REGISTRY
		envNameFallback string
	}{
		// Container Authentication
		{key: "source.containers.auth.registry"}, {key: "source.containers.auth.username"}, {key: "source.containers.auth.password"},
		{key: "target.containers.auth.registry"}, {key: "target.containers.auth.username"}, {key: "target.containers.auth.password"},

		// Helm Chart repository authentication. Maintaining previous name for compatibility reasons
		{key: "source.repo.auth.username", envNameFallback: "SOURCE_AUTH_USERNAME"}, {key: "source.repo.auth.password", envNameFallback: "SOURCE_AUTH_PASSWORD"},
		{key: "target.repo.auth.username", envNameFallback: "TARGET_AUTH_USERNAME"}, {key: "target.repo.auth.password", envNameFallback: "TARGET_AUTH_PASSWORD"},
	}

	for _, k := range boundKeys {
		if err := viper.BindEnv(k.key); err != nil {
			return errors.Trace(err)
		}

		// If there is an environment name fallback name we also set it. This is for compatibility reasons
		if k.envNameFallback == "" {
			continue
		}

		if err := viper.BindEnv(k.key, k.envNameFallback); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
