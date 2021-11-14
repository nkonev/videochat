package config

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"github.com/spf13/viper"
)

//go:embed config-dev
var configDev embed.FS

func InitFlags() string {
	configFile := flag.String("config", "", "Path to config file")

	flag.Parse()
	return *configFile
}

func InitViper(configFile, envPrefix string) {
	viper.SetConfigType("yaml")
	if configFile == "" {
		if embedBytes, err := configDev.ReadFile("config-dev/config.yml"); err != nil {
			panic(fmt.Errorf("Fatal error during reading embedded config file: %s \n", err))
		} else if err := viper.ReadConfig(bytes.NewBuffer(embedBytes)); err != nil {
			panic(fmt.Errorf("Fatal error during viper reading embedded config file: %s \n", err))
		}
	} else {
		viper.SetConfigFile(configFile)
		if err := viper.ReadInConfig(); err != nil { // Handle errors reading the config file
			panic(fmt.Errorf("Fatal error during reading user config file: %s \n", err))
		}
	}
	viper.SetEnvPrefix(envPrefix)
	viper.AutomaticEnv()
	// Find and read the config file
}
