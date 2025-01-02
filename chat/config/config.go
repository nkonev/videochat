package config

import (
	"bytes"
	"embed"
	"errors"
	"flag"
	"fmt"
	"github.com/spf13/viper"
	"log"
	"nkonev.name/chat/app"
	"strings"
)

//go:embed config-dev
var configDev embed.FS

func InitViper() {
	overrideConfigPath := *flag.String("o", "", "Path to config file")
	applyBaseConfig := *flag.Bool("b", true, "Use base config")

	flag.Parse()

	viper.SetConfigType("yaml")

	if applyBaseConfig {
		log.Printf("Applying base config")
		if embedBytes, err := configDev.ReadFile("config-dev/config.yml"); err != nil {
			panic(fmt.Errorf("Fatal error during reading embedded config file: %s \n", err))
		} else if err := viper.ReadConfig(bytes.NewBuffer(embedBytes)); err != nil {
			panic(fmt.Errorf("Fatal error during viper reading embedded config file: %s \n", err))
		}
	} else {
		log.Printf("Not applying base config")
	}

	if err := viper.MergeInConfig(); err != nil {
		if errors.As(err, &viper.ConfigFileNotFoundError{}) {
			log.Printf("Override config file is not found, overrideConfigPath=%v", overrideConfigPath)
		} else {
			// Handle errors reading the config file
			panic(fmt.Errorf("Fatal error during reading user config file: %s \n", err))
		}
	} else {
		log.Printf("Override config file successfully merged, overrideConfigPath=%v", overrideConfigPath)
	}

	viper.SetEnvPrefix(strings.ToUpper(app.APP_NAME))
	viper.AutomaticEnv()
	// Find and read the config file
}
