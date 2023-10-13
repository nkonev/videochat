package config

import (
	"bytes"
	"embed"
	"errors"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
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

type ChatConfig struct {
	ChatUrlConfig ChatUrlConfig `mapstructure:"url"`
}

type AaaConfig struct {
	AaaUrlConfig AaaUrlConfig `mapstructure:"url"`
}

type StorageConfig struct {
	StorageUrlConfig StorageUrlConfig `mapstructure:"url"`
}

type ChatUrlConfig struct {
	Base                        string `mapstructure:"base"`
	Access                      string `mapstructure:"access"`
	IsChatAdmin                 string `mapstructure:"isChatAdmin"`
	DoesParticipantBelongToChat string `mapstructure:"doesParticipantBelongToChat"`
	ChatParticipantIds          string `mapstructure:"chatParticipants"`
	ChatInviteName              string `mapstructure:"chatInviteName"`
	ChatBasicInfoPath           string `mapstructure:"chatBasicInfoPath"`
}

type AaaUrlConfig struct {
	Base     string `mapstructure:"base"`
	GetUsers string `mapstructure:"getUsers"`
}

type StorageUrlConfig struct {
	Base string `mapstructure:"base"`
	S3   string `mapstructure:"s3"`
}

type HttpServerConfig struct {
	ApiAddress          string        `mapstructure:"apiAddress"`
	ShutdownTimeout     time.Duration `mapstructure:"shutdownTimeout"`
	BodyLimit           string        `mapstructure:"bodyLimit"`
}

type RestClientConfig struct {
	MaxIdleConns       int           `mapstructure:"maxIdleConns"`
	IdleConnTimeout    time.Duration `mapstructure:"idleConnTimeout"`
	DisableCompression bool          `mapstructure:"disableCompression"`
}

type FrontendConfig struct {
	VideoResolution    string `mapstructure:"videoResolution"`
	ScreenResolution   string `mapstructure:"screenResolution"`
	VideoSimulcast     *bool  `mapstructure:"videoSimulcast"`
	ScreenSimulcast    *bool  `mapstructure:"screenSimulcast"`
	RoomDynacast       *bool  `mapstructure:"roomDynacast"`
	RoomAdaptiveStream *bool  `mapstructure:"roomAdaptiveStream"`
	Codec 			   *string `mapstructure:"codec"`
}

type AuthConfig struct {
	ExcludePaths []string `mapstructure:"exclude"`
}

type JaegerConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

type LivekitApiConfig struct {
	Key    string `mapstructure:"key"`
	Secret string `mapstructure:"secret"`
}

type LivekitConfig struct {
	Api LivekitApiConfig `mapstructure:"api"`
	Url string           `mapstructure:"url"`
}

type RabbitMqConfig struct {
	Url   string `mapstructure:"url"`
	Debug bool   `mapstructure:"debug"`
}

type RedisConfig struct {
	Address    string `mapstructure:"address"`
	Password   string `mapstructure:"password"`
	Db         int    `mapstructure:"db"`
	MaxRetries int    `mapstructure:"maxRetries"`
}

type ExtendedConfig struct {
	FrontendConfig                        FrontendConfig   `mapstructure:"frontend"`
	RestClientConfig                      RestClientConfig `mapstructure:"http"`
	ChatConfig                            ChatConfig       `mapstructure:"chat"`
	AaaConfig                             AaaConfig        `mapstructure:"aaa"`
	StorageConfig                         StorageConfig    `mapstructure:"storage"`
	AuthConfig                            AuthConfig       `mapstructure:"auth"`
	LivekitConfig                         LivekitConfig    `mapstructure:"livekit"`
	JaegerConfig                          JaegerConfig     `mapstructure:"jaeger"`
	HttpServerConfig                      HttpServerConfig `mapstructure:"server"`
	RabbitMqConfig                        RabbitMqConfig   `mapstructure:"rabbitmq"`
	OnlyRoleAdminRecording                bool             `mapstructure:"onlyRoleAdminRecording"`
	RecordPreset                          string           `mapstructure:"recordPreset"`
	RedisConfig                           RedisConfig      `mapstructure:"redis"`
}
