package utils

import (
	"context"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"regexp"
	"time"
)

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

func GetMongoDbName(mongoUrl string) string {
	uri, err := connstring.Parse(mongoUrl)
	if err != nil {
		log.Panicf("Error during parsing url: %v", err)
	}

	return uri.Database
}

func GetMongoUrl() string {
	return viper.GetString("mongo.databaseUrl")
}

func GetMongoConnectTimeout() time.Duration {
	viper.SetDefault("mongo.connect.timeout", "10s")
	return viper.GetDuration("mongo.connect.timeout")
}

func GetMongoDatabase(client *mongo.Client) *mongo.Database {
	return client.Database(GetMongoDbName(GetMongoUrl()))
}

func InitFlag(defaultLocation string) (string) {
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

const USER_ID = "userId"
const USER_ADMIN = "userAdmin"
const USER_LOGIN = "userLogin"
const DOWNLOAD_PREFIX = "/download/"
const PUBLIC_PREFIX = "/public"
const USER_PREFIX = "user"
const LIMITED = "limited"

func GetMongoClient() *mongo.Client {
	mongoUrl := GetMongoUrl()
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoUrl))
	if err != nil {
		log.Panicf("Error during create mongo client: %v", err)
	}
	ctx, _ := context.WithTimeout(context.Background(), GetMongoConnectTimeout())
	err = client.Connect(ctx)
	if err != nil {
		log.Panicf("Error during connect: %v", err)
	}
	return client
}
