package main

import (
	"embed"
	"flag"
	"fmt"
	"github.com/serialx/hashring"
	"github.com/spf13/viper"
	log "github.com/sirupsen/logrus"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

var Logger = log.New()

type ReplicaAndProxy struct {
	Url *url.URL
	Proxy *httputil.ReverseProxy
}

type VideoReplicasProvider interface {
	GetReplicaByUrl(url0 *url.URL) *ReplicaAndProxy
}

type FileVideoReplicasProvider struct {
	Hashring *hashring.HashRing
	CachedUrls []ReplicaAndProxy
}

func (r *FileVideoReplicasProvider) Init() {
	urls := viper.GetStringSlice("urls")
	var ret []ReplicaAndProxy
	for _, u := range urls {
		url0, err := url.Parse(u)
		if err != nil {
			Logger.Warnf("unable to parse %v, skipping", u)
			continue
		} else {
			ret = append(ret, ReplicaAndProxy{
				Url:          url0,
				Proxy: httputil.NewSingleHostReverseProxy(url0),
			})
		}
	}
	r.CachedUrls = ret
	r.Hashring = hashring.New(urls)
}


func (r *FileVideoReplicasProvider) GetReplicaByUrl(url0 *url.URL) *ReplicaAndProxy {
	ring := r.Hashring
	path := url0.Path
	// "/video/{chatId}/ws"
	split := strings.Split(path, "/")
	if len(split) < 3 {
		Logger.Warnf("Unable to get chatId from url, returning default replica")
		return r.getDefaultReplica()
	}
	chatId := split[2]
	node, ok := ring.GetNode(chatId)
	if !ok {
		Logger.Warnf("Unable to get node from hashring, returning replica")
		return r.getDefaultReplica()
	}

	for _, existing := range r.CachedUrls {
		if existing.Url.String() == node {
			Logger.Printf("Balancing url %v - selecting %v for %v", url0, node, chatId)
			return &existing
		}
	}
	Logger.Warnf("Unable to find replica for %v", url0)
	return nil
}

func (r *FileVideoReplicasProvider) getDefaultReplica() *ReplicaAndProxy{
	if len(r.CachedUrls) == 0 {
		Logger.Warnf("Unable to get default replica")
		return nil
	}
	return &r.CachedUrls[0]
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
	provider.Init()

	http.Handle("/git.json", Static())
	http.HandleFunc("/", handleRequestAndRedirect(provider))
	if err := http.ListenAndServe(address, nil); err != nil {
		panic(err)
	}
}

// Given a request send it to the appropriate url
func handleRequestAndRedirect(r VideoReplicasProvider) func (res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		serveReverseProxy(r.GetReplicaByUrl(req.URL), res, req)
	}
}

// Serve a reverse proxy for a given url
func serveReverseProxy(target *ReplicaAndProxy, res http.ResponseWriter, req *http.Request) {
	if target == nil {
		res.WriteHeader(http.StatusBadGateway)
		return
	}

	// parse the url
	url0 := target.Url
	// create the reverse proxy
	proxy := target.Proxy

	// Update the headers to allow for SSL redirection
	req.URL.Host = url0.Host
	req.URL.Scheme = url0.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url0.Host

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}

//go:embed static
var embeddedFiles embed.FS

func Static() http.HandlerFunc {
	fsys, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		panic("Cannot open static embedded dir")
	}
	staticDir := http.FS(fsys)

	fileServer := http.FileServer(staticDir)
	return func(w http.ResponseWriter, r *http.Request) {
		reqUrl := r.RequestURI
		if reqUrl == "/" || reqUrl == "/index.html" || reqUrl == "/favicon.ico" || strings.HasPrefix(reqUrl, "/build") || strings.HasPrefix(reqUrl, "/assets") || reqUrl == "/git.json" {
			fileServer.ServeHTTP(w, r)
		}
	}
}

