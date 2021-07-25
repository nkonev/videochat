package main

import (
	"embed"
	"flag"
	"fmt"
	"github.com/serialx/hashring"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io/fs"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

var Logger = log.New()

type ReplicaAndProxy struct {
	Url *url.URL
	Proxy *httputil.ReverseProxy
}

type VideoReplicasProvider interface {
	GetReplicaByUrl(url0 *url.URL) *ReplicaAndProxy
	Refresh()
}

type urlsAndHashRing struct {
	hashring *hashring.HashRing
	cachedUrls []ReplicaAndProxy
}

type AbstractReplicasProvider struct {
	cachedUrlsAndHashRing atomic.Value
}

type FileVideoReplicasProvider struct {
	AbstractReplicasProvider
}

func (f *AbstractReplicasProvider) getUrlsAndHashRing() *urlsAndHashRing {
	return f.cachedUrlsAndHashRing.Load().(*urlsAndHashRing)
}

func (f *AbstractReplicasProvider) setUrlsAndHashRing(slice *urlsAndHashRing) {
	f.cachedUrlsAndHashRing.Store(slice)
}

func (r *FileVideoReplicasProvider) Refresh() {
	if err := viper.ReadInConfig(); err != nil {
		Logger.Errorf("Fatal error config file: %s \n", err)
		return
	}

	urls := viper.GetStringSlice("urlsSource.file.urls")
	var ret = []ReplicaAndProxy{}
	for _, u := range urls {
		Logger.Debugf("Processing url from config: %v", u)
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
	r.setUrlsAndHashRing(&urlsAndHashRing{
		hashring:   hashring.New(urls),
		cachedUrls: ret,
	})
}

type DnsAVideoReplicasProvider struct {
	AbstractReplicasProvider
}

func (r *DnsAVideoReplicasProvider) Refresh() {
	nameToLookup := viper.GetString("urlsSource.dnsA.nameToLookup")
	ips, err := net.LookupIP(nameToLookup)
	if err != nil {
		Logger.Warnf("Could not get IPs for name %v: %v\n", nameToLookup, err)
		return
	}
	protocolVar := viper.GetString("urlsSource.dnsA.protocol")
	portVar := viper.GetString("urlsSource.dnsA.port")

	var ret = []ReplicaAndProxy{}
	var urls []string = []string{}
	for _, u := range ips {
		urlStr := protocolVar + "://" + u.String() + ":" + portVar
		Logger.Debugf("Built url with resolving ip from DNS: %v", urlStr)
		url0, err := url.Parse(urlStr)
		if err != nil {
			Logger.Warnf("unable to parse %v, skipping", u)
			continue
		} else {
			urls = append(urls, url0.String())
			ret = append(ret, ReplicaAndProxy{
				Url:          url0,
				Proxy: httputil.NewSingleHostReverseProxy(url0),
			})
		}
	}
	r.setUrlsAndHashRing(&urlsAndHashRing{
		hashring:   hashring.New(urls),
		cachedUrls: ret,
	})

}


func (r *AbstractReplicasProvider) GetReplicaByUrl(url0 *url.URL) *ReplicaAndProxy {
	urlsAndHashRing := r.getUrlsAndHashRing()
	ring := urlsAndHashRing.hashring
	path := url0.Path
	// "/video/{chatId}/ws"
	split := strings.Split(path, "/")
	if len(split) < 3 {
		Logger.Warnf("Unable to get chatId from url, returning default replica")
		return getDefaultReplica(urlsAndHashRing)
	}
	chatId := split[2]
	node, ok := ring.GetNode(chatId)
	if !ok {
		Logger.Warnf("Unable to get node from hashring, returning replica")
		return getDefaultReplica(urlsAndHashRing)
	}

	for _, existing := range urlsAndHashRing.cachedUrls {
		if existing.Url.String() == node {
			Logger.Printf("Balancing url %v - selecting %v for %v", url0, node, chatId)
			return &existing
		}
	}
	Logger.Warnf("Unable to find replica for %v", url0)
	return nil
}

func getDefaultReplica(urlsAndHashRing *urlsAndHashRing) *ReplicaAndProxy{
	if len(urlsAndHashRing.cachedUrls) == 0 {
		Logger.Warnf("Unable to get default replica")
		return nil
	}
	return &urlsAndHashRing.cachedUrls[0]
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

	logLevel := viper.GetString("log.level")
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		Logger.Errorf("Unable to parse log level from %v", logLevel)
	} else {
		Logger.SetLevel(level)
	}

	address := viper.GetString("listenAddress")
	Logger.Printf("Starting on address %v", address)

	var provider VideoReplicasProvider
	urlSourceType := viper.GetString("urlsSource.type")
	Logger.Infof("urlSourceType: %v", urlSourceType)
	switch urlSourceType {
	case "file":
		provider = &FileVideoReplicasProvider{}
	case "dnsA":
		provider = &DnsAVideoReplicasProvider{}
	default:
		panic("Unknown urlsSource.type " + urlSourceType)
	}

	provider.Refresh()

	refreshInterval := viper.GetDuration("refreshInterval")
	if refreshInterval != 0 {
		scheduleCachedUrlsRefreshing(refreshInterval, provider)
	}

	http.Handle("/git.json", Static())
	http.HandleFunc("/", handleRequestAndRedirect(provider))
	if err := http.ListenAndServe(address, nil); err != nil {
		panic(err)
	}
}

func scheduleCachedUrlsRefreshing(refreshInterval time.Duration, provider VideoReplicasProvider) *chan struct{} {
	ticker := time.NewTicker(refreshInterval)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <- ticker.C:
				Logger.Debugf("Refreshing config")
				provider.Refresh()
			case <- quit:
				ticker.Stop()
				return
			}
		}
	}()
	return &quit
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

