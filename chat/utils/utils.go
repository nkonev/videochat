package utils

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"regexp"
)

const SESSION_COOKIE = "SESSION"
const AUTH_URL = "auth.url"
const USER_PRINCIPAL_DTO = "userPrincipalDto"

type H map[string]interface{}

func StringsToRegexpArray(strings []string) []regexp.Regexp {
	regexps := make([]regexp.Regexp, len(strings))
	for i, str := range strings {
		r, err := regexp.Compile(str)
		if err != nil {
			panic(err)
		} else {
			regexps[i] = *r
		}
	}
	return regexps
}

func InitFlag(defaultLocation string) string {
	configFile := flag.String("config", defaultLocation, "Path to config file")

	flag.Parse()
	return *configFile
}

func InitViper(configFile string) {
	viper.SetConfigFile(configFile)
	// call multiple times to add many search paths
	viper.SetEnvPrefix("BLOG_STORE")
	viper.AutomaticEnv()
	// Find and read the config file
	if err := viper.ReadInConfig(); err != nil { // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
}

func CheckUrlInWhitelist(whitelist []regexp.Regexp, uri string) bool {
	for _, regexp0 := range whitelist {
		if regexp0.MatchString(uri) {
			log.Infof("Skipping authentication for %v because it matches %v", uri, regexp0.String())
			return true
		}
	}
	return false
}
