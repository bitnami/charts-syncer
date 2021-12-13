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
const DefaultIndexName = "index"

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
	err := yamlToProto(viper.ConfigFileUsed(), config)
	if err != nil {
		return errors.Trace(fmt.Errorf("error unmarshalling config file: %w", err))
	}
	if config.GetTarget().GetRepoName() == "" {
		klog.V(4).Infof("'target.repoName' property is empty. Using %q default value", defaultRepoName)
		config.GetTarget().RepoName = defaultRepoName
	}
	if config.GetSource().GetRepo().GetUseChartsIndex() && config.GetSource().GetRepo().GetChartsIndex() == "" {
		if err := setDefaultChartsIndex(config); err != nil {
			return err
		}
	}
	if err := config.Source.Repo.SetBasicAuth(os.Getenv("SOURCE_AUTH_USERNAME"), os.Getenv("SOURCE_AUTH_PASSWORD")); err != nil {
		return err
	}
	if err := config.Target.Repo.SetBasicAuth(os.Getenv("TARGET_AUTH_USERNAME"), os.Getenv("TARGET_AUTH_PASSWORD")); err != nil {
		return err
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
