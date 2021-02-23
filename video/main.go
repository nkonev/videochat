package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	log "github.com/pion/ion-log"
	"github.com/pion/ion-sfu/cmd/signal/json-rpc/server"
	"github.com/pion/ion-sfu/pkg/middlewares/datachannel"
	"github.com/pion/ion-sfu/pkg/sfu"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sourcegraph/jsonrpc2"
	websocketjsonrpc2 "github.com/sourcegraph/jsonrpc2/websocket"
	"github.com/spf13/viper"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
)

var (
	conf        = sfu.Config{}
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
	viper.SetConfigType("toml")

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("config file %s read failed. %v\n", file, err)
		return false
	}
	err = viper.GetViper().Unmarshal(&conf)
	if err != nil {
		fmt.Printf("sfu config file %s loaded failed. %v\n", file, err)
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
	flag.StringVar(&file, "c", "config.toml", "config file")
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
		MaxIdleConns:       viper.GetInt("http.idle.conns.max"),
		IdleConnTimeout:    viper.GetDuration("http.idle.connTimeout"),
		DisableCompression: viper.GetBool("http.disableCompression"),
	}
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{Transport: tr}
	return client
}

// TODO Map<ChatId<Map<UserId, Peer>>>

func main() {
	if !parse() {
		showHelp()
		os.Exit(-1)
	}

	fixByFile := []string{"asm_amd64.s", "proc.go", "icegatherer.go", "jsonrpc2"}
	fixByFunc := []string{"Handle"}
	log.Init(conf.Log.Level, fixByFile, fixByFunc)

	log.Infof("--- Starting SFU Node ---")

	s := sfu.NewSFU(conf)
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

	http.Handle("/ws", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO r.Header.Get("X-Auth-UserId")
		userId := r.URL.Query().Get("userId")
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
		defer p.Close()

		jc := jsonrpc2.NewConn(r.Context(), websocketjsonrpc2.NewObjectStream(c), p)
		<-jc.DisconnectNotify()
	}))

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

func checkAccess(client *http.Client, userIdString string, chatIdString string) bool {
	url0 := viper.GetString("chat.url.base")
	url1 := viper.GetString("chat.url.access")

	response, err := client.Get(url0 + url1 + "?userId=" + userIdString + "&chatId=" + chatIdString)
	if err != nil {
		log.Errorf("Error during checking access %v", err)
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
