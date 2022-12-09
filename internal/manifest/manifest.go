package manifest

import (
	"bytes"
	"fmt"

	"github.com/bitnami-labs/pbjson"
	"github.com/golang/protobuf/proto"
	"github.com/juju/errors"
	"github.com/spf13/viper"
	"io/ioutil"
	"sigs.k8s.io/yaml"

	"github.com/bitnami-labs/charts-syncer/api"
)

// Load unmarshall config file into Config struct.
func Load(config *api.Manifest) error {
	// Load the config file
	if err := yamlToProto(viper.ConfigFileUsed(), config); err != nil {
		return errors.Trace(fmt.Errorf("error unmarshalling config file: %w", err))
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
