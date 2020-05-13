package config

import (
	"github.com/bitnami-labs/chart-repository-syncer/api"
	"github.com/spf13/viper"
	"k8s.io/klog"
)

// LoadConfig unmarshall config file into Config struct
func LoadConfig(Config *api.Config) {
	viper.BindEnv("source.auth.username", "SOURCE_AUTH_USERNAME")
	viper.BindEnv("source.auth.password", "SOURCE_AUTH_PASSWORD")
	viper.BindEnv("target.auth.username", "TARGET_AUTH_USERNAME")
	viper.BindEnv("target.auth.password", "TARGET_AUTH_PASSWORD")

	err := viper.Unmarshal(&Config)
	if err != nil {
		klog.Fatalf("Unable to unmarshall config file into struct, %v", err)
	}
}
