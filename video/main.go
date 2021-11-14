package main

import (
	"bytes"
	"crypto/tls"
	"embed"
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	log "github.com/pion/ion-sfu/pkg/logger"
	"github.com/pion/ion-sfu/pkg/middlewares/datachannel"
	"github.com/pion/ion-sfu/pkg/sfu"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"net"
	"net/http"
	_ "net/http/pprof"
	"nkonev.name/video/config"
	"nkonev.name/video/handlers"
	"nkonev.name/video/producer"
	myRabbitmq "nkonev.name/video/rabbitmq"
	"nkonev.name/video/service"
	"os"
	"time"
)

var (
	conf   = config.ExtendedConfig{}
	applyBaseConfig bool
	overrideConfigPath string
	logger = log.New()
)

//go:embed config/config-dev
var configDev embed.FS

const (
	portRangeLimit = 100
)

func showHelp() {
	fmt.Printf("Usage:%s {params}\n", os.Args[0])
	fmt.Println("      -o {override config file path}")
	fmt.Println("      -h (show help info)")
}

func load() bool {
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")

	if applyBaseConfig {
		logger.Info( "Applying base config")
		if embedBytes, err := configDev.ReadFile("config/config-dev/config.yml"); err != nil {
			panic(fmt.Errorf("Fatal error during reading embedded config file: %s \n", err))
		} else if err := viper.ReadConfig(bytes.NewBuffer(embedBytes)); err != nil {
			panic(fmt.Errorf("Fatal error during viper reading embedded config file: %s \n", err))
		}
	} else {
		logger.Info( "Not applying base config")
	}

	viper.AddConfigPath(overrideConfigPath)

	if err := viper.MergeInConfig(); err != nil {
		if errors.As(err, &viper.ConfigFileNotFoundError{}) {
			logger.Info( "Override config file is not found", "overrideConfigPath", overrideConfigPath)
		} else {
			// Handle errors reading the config file
			panic(fmt.Errorf("Fatal error during reading user config file: %s \n", err))
		}
	} else {
		logger.Info( "Override config file successfully merged", "overrideConfigPath", overrideConfigPath)
	}

	viper.SetEnvPrefix("VIDEO")
	viper.AutomaticEnv()

	err := viper.GetViper().Unmarshal(&conf)
	if err != nil {
		fmt.Printf("sfu extended config file loaded failed. %v\n", err)
		return false
	}
	err = viper.GetViper().Unmarshal(&conf.Config)
	if err != nil {
		fmt.Printf("sfu core config file loaded failed. %v\n", err)
		return false
	}
	for _, tc := range conf.FrontendConfig.ICEServers {
		err = viper.GetViper().Unmarshal(&tc.ICEServerConfig)
		if err != nil {
			fmt.Printf("sfu extended turn config loaded failed. %v\n", err)
			return false
		}
	}

	if len(conf.WebRTC.ICEPortRange) > 2 {
		fmt.Printf("config file loaded failed. range port must be [min,max]\n")
		return false
	}

	if len(conf.WebRTC.ICEPortRange) != 0 && conf.WebRTC.ICEPortRange[1]-conf.WebRTC.ICEPortRange[0] < portRangeLimit {
		fmt.Printf("config file loaded failed. range port must be [min, max] and max - min >= %d\n", portRangeLimit)
		return false
	}

	if len(conf.Turn.PortRange) > 2 {
		logger.Error(nil, "config file loaded failed. turn port must be [min,max]")
		return false
	}

	if conf.LogC.V < 0 {
		logger.Error(nil, "Logger V-Level cannot be less than 0")
		return false
	}

	if conf.SyncNotificationPeriod == 0 {
		conf.SyncNotificationPeriod = 2 * time.Second
		logger.Info("Setting default sync notification period", "syncNotificationPeriod", conf.SyncNotificationPeriod)
	}

	logger.V(0).Info("Config loaded", "overrideConfigPath", overrideConfigPath)

	d, err := yaml.Marshal(viper.AllSettings())
	if err != nil {
		logger.V(2).Error(err, "Unable to show yaml")
	}
	logger.V(2).Info("Parsed config", "config", string(d))
	return true
}

func parse() bool {
	flag.BoolVar(&applyBaseConfig, "b", true, "use base config")
	flag.StringVar(&overrideConfigPath, "o", "", "override config file path")
	help := flag.Bool("h", false, "help info")
	flag.Parse()
	if !load() {
		return false
	}

	if *help {
		return false
	}
	return true
}

func startMetrics(addr string) {
	// start metrics server
	m := http.NewServeMux()
	m.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{
		Handler: m,
	}

	metricsLis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error(err, "cannot bind to metrics endpoint", "addr", addr)
		os.Exit(1)
	}
	logger.Info("Metrics Listening", "addr", addr)

	err = srv.Serve(metricsLis)
	if err != nil {
		logger.Error(err, "Metrics server stopped")
	}
}

func NewRestClient() *http.Client {
	tr := &http.Transport{
		MaxIdleConns:       conf.RestClientConfig.MaxIdleConns,
		IdleConnTimeout:    conf.RestClientConfig.IdleConnTimeout,
		DisableCompression: conf.RestClientConfig.DisableCompression,
	}
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{Transport: tr}
	return client
}

func main() {
	if !parse() {
		showHelp()
		os.Exit(-1)
	}

	log.SetGlobalOptions(conf.LogC)
	logger.Info("--- Starting SFU Node ---")

	// Pass logr instance
	sfu.Logger = logger

	conf.TurnAuth = service.GetCompositeTurnAuth(conf)
	sfuInstance := sfu.NewSFU(conf.Config)
	dc := sfuInstance.NewDatachannel(sfu.APIChannelLabel)
	dc.Use(datachannel.SubscriberAPI)

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	client := NewRestClient()

	rabbitmqConnection := myRabbitmq.CreateRabbitMqConnection(conf.RabbitMqConfig)
	publisherService := producer.NewRabbitPublisher(rabbitmqConnection)
	extendedService := service.NewExtendedService(sfuInstance, &conf, publisherService, client)
	handler := handlers.NewHandler(&upgrader, sfuInstance, &conf, &extendedService)

	r := mux.NewRouter()
	// SFU websocket endpoint
	r.Handle("/video/{chatId}/ws", http.HandlerFunc(handler.SfuHandler)).Methods("GET")
	r.Handle("/video/{chatId}/users", http.HandlerFunc(handler.CountUsers)).Methods("GET")
	r.Handle("/video/{chatId}/config", http.HandlerFunc(handler.Config)).Methods("GET")
	r.Handle("/video/{chatId}/mute", http.HandlerFunc(handler.ForceMute)).Methods("PUT")
	r.Handle("/video/{chatId}/kick", http.HandlerFunc(handler.PublicKick)).Methods("PUT")

	r.Handle("/internal/{chatId}/kick", http.HandlerFunc(handler.Kick)).Methods("PUT")
	r.Handle("/internal/{chatId}/user", http.HandlerFunc(handler.UserByStreamId)).Methods("GET")
	r.Handle("/internal/{chatId}/users", http.HandlerFunc(handler.Users)).Methods("GET")

	r.PathPrefix("/").Methods("GET").HandlerFunc(handler.Static())

	go startMetrics(conf.HttpServerConfig.MetricsAddr)

	schedule := extendedService.Schedule()

	var err error
	if conf.HttpServerConfig.Key != "" && conf.HttpServerConfig.Cert != "" {
		logger.Info("Started listening", "addr", "https://"+conf.HttpServerConfig.Addr)
		err = http.ListenAndServeTLS(conf.HttpServerConfig.Addr, conf.HttpServerConfig.Cert, conf.HttpServerConfig.Key, r)
	} else {
		logger.Info("Started listening", "addr", "http://"+conf.HttpServerConfig.Addr)
		err = http.ListenAndServe(conf.HttpServerConfig.Addr, r)
	}
	*schedule <- struct{}{}
	if err != nil {
		panic(err)
	}
}

