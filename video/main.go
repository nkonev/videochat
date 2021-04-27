package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	log "github.com/pion/ion-sfu/pkg/logger"
	"github.com/pion/ion-sfu/pkg/middlewares/datachannel"
	"github.com/pion/ion-sfu/pkg/sfu"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	"net"
	"net/http"
	_ "net/http/pprof"
	"nkonev.name/video/config"
	"nkonev.name/video/handlers"
	"nkonev.name/video/listener"
	"nkonev.name/video/producer"
	"os"
	"time"
	myRabbitmq "nkonev.name/video/rabbitmq"
)

var (
	conf   = config.ExtendedConfig{}
	file   string
	logger = log.New()
)

const (
	portRangeLimit = 100
)

func showHelp() {
	fmt.Printf("Usage:%s {params}\n", os.Args[0])
	fmt.Println("      -c {config file}")
	fmt.Println("      -h (show help info)")
}

func load() bool {
	_, err := os.Stat(file)
	if err != nil {
		return false
	}

	viper.SetConfigFile(file)
	viper.SetConfigType("yml")

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("config file %s read failed. %v\n", file, err)
		return false
	}
	err = viper.GetViper().Unmarshal(&conf)
	if err != nil {
		fmt.Printf("sfu extended config file %s loaded failed. %v\n", file, err)
		return false
	}
	err = viper.GetViper().Unmarshal(&conf.Config)
	if err != nil {
		fmt.Printf("sfu core config file %s loaded failed. %v\n", file, err)
		return false
	}

	if len(conf.WebRTC.ICEPortRange) > 2 {
		fmt.Printf("config file %s loaded failed. range port must be [min,max]\n", file)
		return false
	}

	if len(conf.WebRTC.ICEPortRange) != 0 && conf.WebRTC.ICEPortRange[1]-conf.WebRTC.ICEPortRange[0] < portRangeLimit {
		fmt.Printf("config file %s loaded failed. range port must be [min, max] and max - min >= %d\n", file, portRangeLimit)
		return false
	}

	if len(conf.Turn.PortRange) > 2 {
		logger.Error(nil, "config file loaded failed. turn port must be [min,max]", "file", file)
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

	logger.V(0).Info("Config file loaded", "file", file)

	fmt.Printf("config %s load ok!\n", file)
	return true
}

func parse() bool {
	flag.StringVar(&file, "config", "video/config.yml", "config file")
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

	s := sfu.NewSFU(conf.Config)
	dc := s.NewDatachannel(sfu.APIChannelLabel)
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
	extendedService := handlers.NewExtendedService(s, &conf, publisherService, client)
	handler := handlers.NewHandler(&upgrader, &conf, &extendedService)
	listener.NewVideoListener(&extendedService, rabbitmqConnection)

	r := mux.NewRouter()
	// SFU websocket endpoint
	r.Handle("/video/{chatId}/ws", http.HandlerFunc(handler.SfuHandler)).Methods("GET")
	r.Handle("/video/{chatId}/users", http.HandlerFunc(handler.Users)).Methods("GET")
	r.Handle("/video/{chatId}/notify", http.HandlerFunc(handler.StoreInfoAndNotifyChatParticipants)).Methods("PUT")
	r.Handle("/video/{chatId}/config", http.HandlerFunc(handler.Config)).Methods("GET")

	r.Handle("/internal/{chatId}/kick", http.HandlerFunc(handler.Kick)).Methods("PUT")
	r.Handle("/internal/{chatId}/user", http.HandlerFunc(handler.UserByStreamId)).Methods("GET")

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
	*schedule <- struct { }{}
	if err != nil {
		panic(err)
	}
}
