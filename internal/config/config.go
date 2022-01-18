package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/pbjson"
	"github.com/golang/protobuf/proto"
	"github.com/juju/errors"
	"github.com/spf13/viper"
	"k8s.io/klog"
	"sigs.k8s.io/yaml"
)

const (
	defaultRepoName = "myrepo"
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
	// TODO: Use Viper to load the config file instead
	err := yamlToProto(viper.ConfigFileUsed(), config)
	if err != nil {
		return errors.Trace(fmt.Errorf("error unmarshalling config file: %w", err))
	}
	if config.GetSource().GetRepo() != nil {
		if config.GetSource().GetRepo().GetUseChartsIndex() && config.GetSource().GetRepo().GetChartsIndex() == "" {
			if err := setDefaultChartsIndex(config); err != nil {
				return err
			}
		}
	}

	if config.GetTarget().GetRepoName() == "" {
		klog.V(4).Infof("'target.repoName' property is empty. Using %q default value", defaultRepoName)
		config.GetTarget().RepoName = defaultRepoName
	}

	// Container registry authentication override
	if err := setAuthentication(config.GetSource(), config.GetTarget()); err != nil {
		return err
	}

	return nil
}

// Sets the authentication configuration for container images
// It reads the configuration from the viper config repository which values might have come from the config file, env vars or flags
func setAuthentication(source *api.Source, target *api.Target) error {
	// Source authentication for Helm and container registries
	if source != nil {
		// Helm Chart authentication
		// NOTE: Getting entries one by one is required since they match the env variables defined and being overridden i.e SOURCE_CONTAINERAUTH_REGISTRY
		username, password := viper.GetString("source.repo.auth.username"), viper.GetString("source.repo.auth.password")
		if username != "" && password != "" {
			source.GetRepo().Auth = &api.Auth{Username: username, Password: password}
		}

		// Set the source OCI repository authentication
		username, password, registry := viper.GetString("source.containerauth.username"), viper.GetString("source.containerauth.password"), viper.GetString("source.containerauth.registry")
		if username != "" && password != "" && registry != "" {
			source.ContainerAuth = &api.ContainerAuth{Username: username, Password: password, Registry: registry}
		}
	}

	// target Chart and container images authentication
	if target != nil {
		username, password := viper.GetString("target.repo.auth.username"), viper.GetString("target.repo.auth.password")
		if username != "" && password != "" {
			target.GetRepo().Auth = &api.Auth{Username: username, Password: password}
		}

		// Target OCI repository
		// NOTE: the registry value is retrieved from target.ContainerRegistry instead of target.ContainerAuth.
		// This is because as part of the target definition the registry is set to indicate where the images
		// should be pushed to, so the authentication must match this registry
		username, password, registry := viper.GetString("target.containerauth.username"), viper.GetString("target.containerauth.password"), viper.GetString("target.containerregistry")
		if username != "" && password != "" && registry != "" {
			target.ContainerAuth = &api.ContainerAuth{Username: username, Password: password, Registry: registry}
		}
	}

	return nil
}

// yamlToProto unmarshals `path` into the provided proto message
func yamlToProto(path string, v proto.Message) error {
	yamlBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.Trace(err)
	}
	jsonBytes, err := yaml.YAMLToJSONStrict(yamlBytes)
	if err != nil {
		return errors.Trace(err)
	}
	r := bytes.NewReader(jsonBytes)
	err = pbjson.NewDecoder(r).Decode(v)
	return errors.Trace(err)
}

// InitEnvBindings defines the env variables bindings enabled to override viper settings
func InitEnvBindings() error {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// Keys allowed to be overridden by env variables
	// i.e source.containerzauth.registry => SOURCE_CONTAINERAUTH_REGISTRY
	boundKeys := []struct {
		key, envNameFallback string
	}{
		// Container Authentication
		{key: "source.containerauth.registry"}, {key: "source.containerauth.username"}, {key: "source.containerauth.password"},
		// NOTE: target registry will be retrieved from target.containerregistry instead since it indicates
		// where the images are going to be pushed to so duplication is not needed
		{key: "target.containerauth.username"}, {key: "target.containerauth.password"},

		// Helm Chart repository authentication. Maintaining previous name for compabitility reasons
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
