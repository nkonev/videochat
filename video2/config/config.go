package config

import (
	"bytes"
	"embed"
	"errors"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

//go:embed config-dev
var configDev embed.FS

func InitViper() {
	overrideConfigPath := *flag.String("o", "", "Path to override config file")
	applyBaseConfig := *flag.Bool("b", true, "Use base config")

	flag.Parse()

	viper.SetConfigType("yaml")

	if applyBaseConfig {
		log.Info("Applying base config")
		if embedBytes, err := configDev.ReadFile("config-dev/config.yml"); err != nil {
			panic(fmt.Errorf("Fatal error during reading embedded config file: %s \n", err))
		} else if err := viper.ReadConfig(bytes.NewBuffer(embedBytes)); err != nil {
			panic(fmt.Errorf("Fatal error during viper reading embedded config file: %s \n", err))
		}
	} else {
		log.Info("Not applying base config")
	}

	if err := viper.MergeInConfig(); err != nil {
		if errors.As(err, &viper.ConfigFileNotFoundError{}) {
			log.Infof("Override config file is not found, overrideConfigPath=%v", overrideConfigPath)
		} else {
			// Handle errors reading the config file
			panic(fmt.Errorf("Fatal error during reading user config file: %s \n", err))
		}
	} else {
		log.Infof("Override config file successfully merged, overrideConfigPath=%v", overrideConfigPath)
	}

	viper.SetEnvPrefix("VIDEO")
	viper.AutomaticEnv()
	// Find and read the config file
}

type ICEServerConfig struct {
	URLs       []string `mapstructure:"urls"`
	Username   string   `mapstructure:"username"`
	Credential string   `mapstructure:"credential"`
}

type ExtendedICEServerConfig struct {
	ICEServerConfig ICEServerConfig `mapstructure:"server"`
}
type FrontendConfig struct {
	ICEServers     []ExtendedICEServerConfig `mapstructure:"iceserver"`
	PreferredCodec string                    `mapstructure:"preferredCodec"`
	Resolution     string                    `mapstructure:"resolution"`
}

type ExtendedConfig struct {
	FrontendConfig FrontendConfig `mapstructure:"frontend"`
}
