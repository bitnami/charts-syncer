package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
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
		// TODO: Move env variable handling to VIPER, see container authentication below
		if err := config.GetSource().GetRepo().SetBasicAuth(os.Getenv("SOURCE_AUTH_USERNAME"), os.Getenv("SOURCE_AUTH_PASSWORD")); err != nil {
			return err
		}
	}
	if config.GetTarget() != nil {
		if err := config.GetTarget().GetRepo().SetBasicAuth(os.Getenv("TARGET_AUTH_USERNAME"), os.Getenv("TARGET_AUTH_PASSWORD")); err != nil {
			return err
		}
	}
	if config.GetTarget().GetRepoName() == "" {
		klog.V(4).Infof("'target.repoName' property is empty. Using %q default value", defaultRepoName)
		config.GetTarget().RepoName = defaultRepoName
	}

	// Container registry authentication override
	if err := loadContainerAuthFromEnv(config); err != nil {
		return err
	}

	return nil
}

// Sets the authentication configuration for container images
// It reads the configuration from the viper config repository which values might have come from the config file, env vars or flags
func loadContainerAuthFromEnv(c *api.Config) error {
	// Set the source OCI repository authentication
	// NOTE: Getting entries one by one is required since they match the env variables defined and being overridden i.e SOURCE_CONTAINERAUTH_REGISTRY
	username, password, registry := viper.GetString("source.containerauth.username"), viper.GetString("source.containerauth.password"), viper.GetString("source.containerauth.registry")
	if username != "" && password != "" && registry != "" && c.GetSource() != nil {
		c.GetSource().ContainerAuth = &api.ContainerAuth{
			Username: username,
			Password: password,
			Registry: registry,
		}
	}

	// Target OCI repository
	// NOTE: the registry value is retrieved from target.ContainerRegistry instead of target.ContainerAuth.
	// This is because as part of the target definition the registry is set to indicate where the images
	// should be pushed to, so the authentication must match this registry
	username, password, registry = viper.GetString("target.containerauth.username"), viper.GetString("target.containerauth.password"), viper.GetString("target.containerregistry")
	if username != "" && password != "" && registry != "" && c.GetTarget() != nil {
		c.GetTarget().ContainerAuth = &api.ContainerAuth{
			Username: username,
			Password: password,
			Registry: registry,
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
