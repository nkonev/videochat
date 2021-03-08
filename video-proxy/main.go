package main

import (
	"flag"
	"fmt"
	"github.com/serialx/hashring"
	"github.com/spf13/viper"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

var Logger = log.New()

type VideoReplicasProvider interface {
	GetReplicas() []string
}

type FileVideoReplicasProvider struct { }

func (r FileVideoReplicasProvider) GetReplicas() []string {
	return viper.GetStringSlice("urls")
}

func main() {
	Logger.SetReportCaller(true)
	Logger.SetFormatter(&log.TextFormatter{ForceColors: true, FullTimestamp: true})
	Logger.SetOutput(os.Stdout)

	configFile := flag.String("config", "./video-proxy/config.yml", "Path to config file")
	flag.Parse()
	viper.SetConfigFile(*configFile)
	// call multiple times to add many search paths
	viper.SetEnvPrefix("VIDEO_PROXY")
	viper.AutomaticEnv()
	// Find and read the config file
	if err := viper.ReadInConfig(); err != nil { // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}


	address := viper.GetString("listenAddress")
	Logger.Printf("Starting on address %v", address)
	provider := &FileVideoReplicasProvider{}
	http.HandleFunc("/", handleRequestAndRedirect(provider))
	if err := http.ListenAndServe(address, nil); err != nil {
		panic(err)
	}
}

type Handler struct {

}

// Given a request send it to the appropriate url
func handleRequestAndRedirect(r VideoReplicasProvider) func (res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		url0 := getProxiedReplica(r, req)
		serveReverseProxy(url0, res, req)
	}
}

// Get the url for a given proxy condition
func getProxiedReplica(r VideoReplicasProvider, req *http.Request) string {
	serversInRing := r.GetReplicas()
	ring := hashring.New(serversInRing)
	path := req.URL.Path
	// "/video/{chatId}/ws"
	split := strings.Split(path, "/")
	if len(split) < 3 {
		Logger.Printf("Unable to get chatId from url, returnig 0")
		return serversInRing[0]
	}
	chat := split[2]
	node, ok := ring.GetNode(chat)
	if !ok {
		Logger.Printf("Unable to get node from hashring, returnig 0")
		return serversInRing[0]
	}
	Logger.Printf("Balancing url %v - selecting %v for %v", req.URL, node, chat)
	return node
}

// Serve a reverse proxy for a given url
func serveReverseProxy(target string, res http.ResponseWriter, req *http.Request) {
	// parse the url
	url0, _ := url.Parse(target)

	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url0)

	// Update the headers to allow for SSL redirection
	req.URL.Host = url0.Host
	req.URL.Scheme = url0.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url0.Host

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}
