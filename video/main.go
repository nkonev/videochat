package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/nkonev/ion-sfu/cmd/signal/json-rpc/server"
	"github.com/nkonev/ion-sfu/pkg/middlewares/datachannel"
	"github.com/nkonev/ion-sfu/pkg/sfu"
	log "github.com/pion/ion-log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rakyll/statik/fs"
	"github.com/sourcegraph/jsonrpc2"
	websocketjsonrpc2 "github.com/sourcegraph/jsonrpc2/websocket"
	"github.com/spf13/viper"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	_ "nkonev.name/video/static_assets"
	"os"
	"strings"
	"sync"
	"time"
)

type RestClientConfig struct {
	MaxIdleConns int `mapstructure:"maxIdleConns"`
	IdleConnTimeout time.Duration `mapstructure:"idleConnTimeout"`
	DisableCompression bool `mapstructure:"disableCompression"`
}

type FrontendConfig struct {
	ICEServers []sfu.ICEServerConfig `mapstructure:"iceserver"`
}

type ChatConfig struct {
	ChatUrlConfig ChatUrlConfig `mapstructure:"url"`
}

type ChatUrlConfig struct {
	Base string `mapstructure:"base"`
	Access string `mapstructure:"access"`
	Notify string `mapstructure:"notify"`
}


type ExtendedConfig struct {
	sfu.Config
	FrontendConfig FrontendConfig `mapstructure:"frontend"`
	RestClientConfig RestClientConfig `mapstructure:"http"`
	ChatConfig ChatConfig `mapstructure:"chat"`
}

var (
	conf        = ExtendedConfig{}
	file        string
	cert        string
	key         string
	addr        string
	metricsAddr string
)

const (
	portRangeLimit = 100
)

func showHelp() {
	fmt.Printf("Usage:%s {params}\n", os.Args[0])
	fmt.Println("      -c {config file}")
	fmt.Println("      -cert {cert file}")
	fmt.Println("      -key {key file}")
	fmt.Println("      -a {listen addr}")
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

	fmt.Printf("config %s load ok!\n", file)
	return true
}

func parse() bool {
	flag.StringVar(&file, "c", "config.yml", "config file")
	flag.StringVar(&cert, "cert", "", "cert file")
	flag.StringVar(&key, "key", "", "key file")
	flag.StringVar(&addr, "a", ":7000", "address to use")
	flag.StringVar(&metricsAddr, "m", ":8100", "merics to use")
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
		log.Panicf("cannot bind to metrics endpoint %s. err: %s", addr, err)
	}
	log.Infof("Metrics Listening at %s", addr)

	err = srv.Serve(metricsLis)
	if err != nil {
		log.Errorf("debug server stopped. got err: %s", err)
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

type UsersResponse struct {
	UsersCount int64 `json:"usersCount"`
}

func configureStaticMiddleware() http.HandlerFunc {
	statikFS, err := fs.NewWithNamespace("assets")
	if err != nil {
		log.Panicf("Unable to load static assets %v", err)
	}

	fileServer := http.FileServer(statikFS)
	return func(w http.ResponseWriter, r *http.Request) {
		reqUrl := r.RequestURI
		if reqUrl == "/" || reqUrl == "/index.html" || reqUrl == "/favicon.ico" || strings.HasPrefix(reqUrl, "/build") || strings.HasPrefix(reqUrl, "/assets") || reqUrl == "/git.json" {
			fileServer.ServeHTTP(w, r)
		}
	}
}

func main() {
	if !parse() {
		showHelp()
		os.Exit(-1)
	}

	fixByFile := []string{"asm_amd64.s", "proc.go", "icegatherer.go", "jsonrpc2"}
	fixByFunc := []string{"Handle"}
	log.Init(conf.Log.Level, fixByFile, fixByFunc)

	log.Infof("--- Starting SFU Node ---")

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

	sessionUserPeer := &sync.Map{}

	http.Handle("/video/ws", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userId := r.Header.Get("X-Auth-UserId")
		chatId := r.URL.Query().Get("chatId")
		if !checkAccess(client, userId, chatId) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			panic(err)
		}
		defer c.Close()

		p := server.NewJSONSignal(sfu.NewPeer(s))
		addPeerToMap(sessionUserPeer, chatId, userId, p)
		defer p.Close()
		defer removePeerFromMap(sessionUserPeer, chatId, userId)

		jc := jsonrpc2.NewConn(r.Context(), websocketjsonrpc2.NewObjectStream(c), p)
		<-jc.DisconnectNotify()
	}))

	// GET /api/video/users?chatId=${this.chatId} - responds users count
	http.Handle("/video/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userId := r.Header.Get("X-Auth-UserId")
		chatId := r.URL.Query().Get("chatId")
		if !checkAccess(client, userId, chatId) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		chatInterface, ok := sessionUserPeer.Load(chatId)
		response := UsersResponse{}
		if ok {
			chat := chatInterface.(*sync.Map)
			response.UsersCount = countMapLen(chat)
		}
		w.Header().Set("Content-Type", "application/json")
		marshal, err := json.Marshal(response)
		if err != nil {
			log.Errorf("Error during marshalling UsersResponse to json")
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			_, err := w.Write(marshal)
			if err != nil {
				log.Errorf("Error during sending json")
			}
		}
	}))

	// PUT /api/video/notify?chatId=${this.chatId}` -> "/internal/video/notify"
	http.Handle("/video/notify", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userId := r.Header.Get("X-Auth-UserId")
		chatId := r.URL.Query().Get("chatId")
		if !checkAccess(client, userId, chatId) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var usersCount int64 = 0
		chatInterface, ok := sessionUserPeer.Load(chatId)
		if ok {
			chat := chatInterface.(*sync.Map)
			usersCount = countMapLen(chat)
		}

		url0 := conf.ChatConfig.ChatUrlConfig.Base
		url1 := conf.ChatConfig.ChatUrlConfig.Notify

		fullUrl := fmt.Sprintf("%v%v?usersCount=%v&chatId=%v", url0, url1, usersCount, chatId)
		parsedUrl, err := url.Parse(fullUrl)
		if err != nil {
			log.Errorf("Failed during parse chat url: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		req := &http.Request{Method: http.MethodPut, URL: parsedUrl}

		response, err := client.Do(req)
		if err != nil {
			log.Errorf("Transport error during notifying %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			if response.StatusCode != http.StatusOK {
				log.Errorf("Http Error %v during notifying %v", response.StatusCode, err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	}))

	// GET `/api/video/config`
	http.Handle("/video/config", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		marshal, err := json.Marshal(conf.FrontendConfig)
		if err != nil {
			log.Errorf("Error during marshalling ConfigResponse to json")
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			_, err := w.Write(marshal)
			if err != nil {
				log.Errorf("Error during sending json")
			}
		}
	}))

	http.HandleFunc("/", configureStaticMiddleware())

	go startMetrics(metricsAddr)

	var err error
	if key != "" && cert != "" {
		log.Infof("Listening at https://[%s]", addr)
		err = http.ListenAndServeTLS(addr, cert, key, nil)
	} else {
		log.Infof("Listening at http://[%s]", addr)
		err = http.ListenAndServe(addr, nil)
	}
	if err != nil {
		panic(err)
	}
}

func addPeerToMap(sessionUserPeer *sync.Map, chatId string, userId string, peer *server.JSONSignal) {
	userPeerInterface, _ := sessionUserPeer.LoadOrStore(chatId, &sync.Map{})
	userPeer := userPeerInterface.(*sync.Map)
	userPeer.Store(userId, peer)
}

func removePeerFromMap(sessionUserPeer *sync.Map, chatId string, userId string) {
	userPeerInterface, ok := sessionUserPeer.Load(chatId)
	if !ok {
		log.Infof("Cannot remove chatId=%v from sessionUserPeer", chatId)
		return
	}
	userPeer := userPeerInterface.(*sync.Map)
	userPeer.Delete(userId)

	userPeerLength := countMapLen(userPeer)

	if userPeerLength == 0 {
		log.Infof("For chatId=%v there is no peers, removing user %v", chatId, userId)
		sessionUserPeer.Delete(chatId)
	}
}

func countMapLen(m *sync.Map) int64 {
	var length int64 = 0
	m.Range(func(_, _ interface{}) bool {
		length++
		return true
	})
	return length
}

func checkAccess(client *http.Client, userIdString string, chatIdString string) bool {
	url0 := conf.ChatConfig.ChatUrlConfig.Base
	url1 := conf.ChatConfig.ChatUrlConfig.Access

	response, err := client.Get(url0 + url1 + "?userId=" + userIdString + "&chatId=" + chatIdString)
	if err != nil {
		log.Errorf("Transport error during checking access %v", err)
		return false
	}
	if response.StatusCode == http.StatusOK {
		return true
	} else if response.StatusCode == http.StatusUnauthorized {
		return false
	} else {
		log.Errorf("Unexpected status on checkAccess %v", response.StatusCode)
		return false
	}
}
