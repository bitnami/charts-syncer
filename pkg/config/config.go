package config

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/bitnami-labs/chart-repository-syncer/api"
	"github.com/bitnami-labs/pbjson"
	"github.com/golang/protobuf/proto"
	"github.com/juju/errors"
	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"
)

// Load unmarshall config file into Config struct.
func Load(config *api.Config) error {
	viper.BindEnv("source.auth.username", "SOURCE_AUTH_USERNAME")
	viper.BindEnv("source.auth.password", "SOURCE_AUTH_PASSWORD")
	viper.BindEnv("target.auth.username", "TARGET_AUTH_USERNAME")
	viper.BindEnv("target.auth.password", "TARGET_AUTH_PASSWORD")

	err := yamlToProto(viper.ConfigFileUsed(), config)
	if err != nil {
		return errors.Trace(fmt.Errorf("tritri Error unmarshalling config file: %w", err))
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
