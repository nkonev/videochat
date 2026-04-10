package config

import (
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"nkonev.name/chat/app"

	"github.com/traefik/paerser/env"
	"github.com/traefik/paerser/file"
	"github.com/traefik/paerser/flag"
)

type KafkaTopicConfig struct {
	Topic             string
	NumPartitions     int32
	ReplicationFactor int16
	Retention         string
}

type KafkaConfig struct {
	BootstrapServers  []string
	TopicChat         KafkaTopicConfig
	TopicUser         KafkaTopicConfig
	ConsumerGroupChat string
	ConsumerGroupUser string
	Producer          KafkaProducerConfig
	Consumer          KafkaConsumerConfig
}

type KafkaProducerConfig struct {
	RetryMax      int
	ReturnSuccess bool
	RetryBackoff  time.Duration
	ClientId      string
}

type KafkaConsumerConfig struct {
	ReturnErrors bool
	ClientId     string
	BatchSize    int
	FetchMaxWait time.Duration
}

type OtlpConfig struct {
	Endpoint string
}

type HttpServerConfig struct {
	Address        string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxHeaderBytes int
	Dump           bool
	PrettyLog      bool
}

type MigrationConfig struct {
	MigrationTable    string
	StatementDuration time.Duration
}

type PostgreSQLConfig struct {
	Url                string
	MaxOpenConnections int
	MaxIdleConnections int
	MaxLifetime        time.Duration
	Migration          MigrationConfig
	Dump               bool
	LogLevel           string
}

type CommandsConfig struct {
	MaxParticipantsPerSingleCommand int32
}

type CqrsConfig struct {
	SleepBeforeEvent                time.Duration
	CheckAreEventsProcessedInterval time.Duration
	Dump                            bool
	PrettyLog                       bool
	Export                          ExportConfig
	Import                          ImportConfig
	Projections                     ProjectionsConfig
	Commands                        CommandsConfig
	TestHelperMethods               bool
	SleepBeforePolling              time.Duration
	PollingMaxTimes                 int
}

type RestClientConfig struct {
	MaxIdleConns       int
	IdleConnTimeout    time.Duration
	DisableCompression bool
	Dump               bool
	PrettyLog          bool
}

type ImportConfig struct {
	File string
}

type ExportConfig struct {
	File string
}

type ChatUserViewConfig struct {
	MaxViewableParticipants         int32
	LastMessageMaxTextDbPreviewSize int32
}

type BlogViewConfig struct {
	MaxTextPreviewSize int32
}

type ProjectionsConfig struct {
	ChatUserView ChatUserViewConfig
	BlogView     BlogViewConfig
}

type LoggerConfig struct {
	Level       string
	Json        bool
	WriteToFile bool
	Dir         string
	Filename    string
}

type AaaConfig struct {
	Url AaaUrlConfig
}

type AaaUrlConfig struct {
	Base           string
	GetUsers       string
	GetUserOnlines string
	GetUserExists  string
	SearchUsers    string
}

func (lc *LoggerConfig) GetLevel() slog.Leveler {
	var lvl slog.Level
	err := lvl.UnmarshalText([]byte(lc.Level))
	if err != nil {
		panic(err)
	}
	return lvl
}

type MessageConfig struct {
	AllowedMediaUrls            string // comma-separated
	AllowedIframeUrls           string // comma-separated
	MaxMedias                   int
	MaxDisplayableReactionUsers int
	PreviewMaxTextSize          int
}

type ChatConfig struct {
	TetATet TetATetConfig
}

type BlogConfig struct {
	RestrictCreateBlog bool
}

type TetATetConfig struct {
	CanResend bool
	CanReact  bool
}

type RabbitMQConfig struct {
	Url                             string
	Debug                           bool
	CheckAreEventsProcessedInterval time.Duration // for tests
	MaxWaitForEvents                time.Duration // for tests
	DumpTestAccumulator             bool          // for tests

	Dump      bool
	PrettyLog bool

	SkipPublishOutputEventsOnRewind       bool
	SkipPublishNotificationEventsOnRewind bool
}

type CleanAbandonedChatsTask struct {
	Enabled    bool
	Cron       string
	Expiration time.Duration
}

type CleanDeletedUsersDataTask struct {
	Enabled    bool
	Cron       string
	Expiration time.Duration
}

type TaskConfig struct {
	CleanAbandonedChatsTask   CleanAbandonedChatsTask
	CleanDeletedUsersDataTask CleanDeletedUsersDataTask
}

type RedisConfig struct {
	Address    string
	Password   string
	Db         int
	MaxRetries int
}

type AppConfig struct {
	Kafka         KafkaConfig
	Otlp          OtlpConfig
	PostgreSQL    PostgreSQLConfig
	PostgreSQLOld PostgreSQLConfig
	Server        HttpServerConfig
	Cqrs          CqrsConfig
	Http          RestClientConfig
	Logger        LoggerConfig
	Aaa           AaaConfig
	Message       MessageConfig
	Chat          ChatConfig
	Blog          BlogConfig
	FrontendUrl   string
	RabbitMQ      RabbitMQConfig
	Schedulers    TaskConfig
	Redis         RedisConfig
}

//go:embed config
var configFs embed.FS

func CreateTypedConfig(args []string) (*AppConfig, error) {
	return createTypedConfig("config-dev.yml", args[:]...)
}

func CreateTestTypedConfig() (*AppConfig, error) {
	return createTypedConfig("config-test.yml")
}

func createTypedConfig(filename string, args ...string) (*AppConfig, error) {
	conf := AppConfig{}
	var err error

	var argsToReadConfig []string

	hasConfigInArgs, configFilePath, argsToConfig, err := app.IsConfig(args)
	if err != nil {
		return nil, fmt.Errorf("An error occured during working with config: %w", err)
	}

	if hasConfigInArgs {
		argsToReadConfig = argsToConfig

		err = file.Decode(configFilePath, &conf)
		if err != nil {
			return nil, fmt.Errorf("config file loaded failed. %v\n", err)
		}
	} else {
		// load default embed config
		embedBytes, err := configFs.ReadFile("config/" + filename)
		if err != nil {
			return nil, fmt.Errorf("Fatal error during reading embedded config file: %s \n", err)
		}
		fileContentString := string(embedBytes)

		err = file.DecodeContent(fileContentString, ".yml", &conf)

		if err != nil {
			return nil, fmt.Errorf("config file loaded failed. %v\n", err)
		}

		argsToReadConfig = argsToConfig
	}

	err = env.Decode(os.Environ(), strings.ToUpper(app.TRACE_RESOURCE)+"_", &conf)
	if err != nil {
		return nil, err
	}

	err = flag.Decode(argsToReadConfig, &conf)
	if err != nil {
		return nil, err
	}

	err = validate(&conf)
	if err != nil {
		return nil, err
	}

	return &conf, nil
}

func validate(conf *AppConfig) error {
	if conf == nil {
		return errors.New("nil config")
	}

	if conf.Cqrs.Projections.ChatUserView.MaxViewableParticipants < 2 {
		return fmt.Errorf("max viewable participants = %d < 2", conf.Cqrs.Projections.ChatUserView.MaxViewableParticipants)
	}

	if conf.Cqrs.Commands.MaxParticipantsPerSingleCommand == 0 {
		return errors.New("max participants = per comamnd cannot be 0")
	}

	return nil
}
