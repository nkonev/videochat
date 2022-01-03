package config

import (
	log "github.com/pion/ion-sfu/pkg/logger"
	"github.com/pion/ion-sfu/pkg/sfu"
	"time"
)

type RestClientConfig struct {
	MaxIdleConns       int           `mapstructure:"maxIdleConns"`
	IdleConnTimeout    time.Duration `mapstructure:"idleConnTimeout"`
	DisableCompression bool          `mapstructure:"disableCompression"`
}

type ExtendedICEServerConfig struct {
	ICEServerConfig            sfu.ICEServerConfig `mapstructure:"server"`
	LongTermCredentialDuration time.Duration       `mapstructure:"turnCredentialDuration"`
}

type FrontendConfig struct {
	ICEServers     []ExtendedICEServerConfig `mapstructure:"iceserver"`
	PreferredCodec string                    `mapstructure:"preferredCodec"`
	Simulcast      bool                      `mapstructure:"simulcast"`
	ForceKickAfter time.Duration             `mapstructure:"forceKickAfter"`
}

type RabbitMqConfig struct {
	Url string `mapstructure:"url"`
}

type ChatConfig struct {
	ChatUrlConfig ChatUrlConfig `mapstructure:"url"`
}

type ChatUrlConfig struct {
	Base        string `mapstructure:"base"`
	Access      string `mapstructure:"access"`
	IsChatAdmin string `mapstructure:"isChatAdmin"`
}

type HttpServerConfig struct {
	Addr        string `mapstructure:"addr"`
	MetricsAddr string `mapstructure:"metricsAddr"`
	Cert        string `mapstructure:"cert"`
	Key         string `mapstructure:"key"`
}

type ExtendedConfig struct {
	sfu.Config
	FrontendConfig         FrontendConfig   `mapstructure:"frontend"`
	RestClientConfig       RestClientConfig `mapstructure:"http"`
	ChatConfig             ChatConfig       `mapstructure:"chat"`
	HttpServerConfig       HttpServerConfig `mapstructure:"server"`
	LogC                   log.GlobalConfig `mapstructure:"log"`
	SyncNotificationPeriod time.Duration    `mapstructure:"syncNotificationPeriod"`
	RabbitMqConfig         RabbitMqConfig   `mapstructure:"rabbitmq"`
}
