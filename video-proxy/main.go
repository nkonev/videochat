package video_proxy

import (
	"bytes"
	"encoding/json"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

type HostPort struct {
	Host string
	Port int16
}

type VideoReplicasProvider interface {
	GetReplicas() []HostPort
}

func main() {
	// start server
	http.HandleFunc("/", handleRequestAndRedirect)
	if err := http.ListenAndServe(viper.GetString("listenAddress"), nil); err != nil {
		panic(err)
	}
}

// Serve a reverse proxy for a given url
func serveReverseProxy(target string, res http.ResponseWriter, req *http.Request) {
	// parse the url
	url, _ := url.Parse(target)

	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	// Update the headers to allow for SSL redirection
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url.Host

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}

// Given a request send it to the appropriate url
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {
	requestPayload := parseRequestBody(req)
	url := getProxyUrl(requestPayload.ProxyCondition)

	logRequestPayload(requestPayload, url)

	serveReverseProxy(url, res, req)
}

// Get the url for a given proxy condition
func getProxyUrl(proxyConditionRaw string) string {
	proxyCondition := strings.ToUpper(proxyConditionRaw)

	a_condtion_url := os.Getenv("A_CONDITION_URL")
	b_condtion_url := os.Getenv("B_CONDITION_URL")
	default_condtion_url := os.Getenv("DEFAULT_CONDITION_URL")

	if proxyCondition == "A" {
		return a_condtion_url
	}

	if proxyCondition == "B" {
		return b_condtion_url
	}

	return default_condtion_url
}

// Log the typeform payload and redirect url
func logRequestPayload(requestionPayload requestPayloadStruct, proxyUrl string) {
	log.Printf("proxy_condition: %s, proxy_url: %s\n", requestionPayload.ProxyCondition, proxyUrl)
}

// Parse the requests body
func parseRequestBody(request *http.Request) requestPayloadStruct {
	decoder := requestBodyDecoder(request)

	var requestPayload requestPayloadStruct
	err := decoder.Decode(&requestPayload)

	if err != nil {
		panic(err)
	}

	return requestPayload
}

type requestPayloadStruct struct {
	ProxyCondition string `json:"proxy_condition"`
}

// Get a json decoder for a given requests body
func requestBodyDecoder(request *http.Request) *json.Decoder {
	// Read body to buffer
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		panic(err)
	}

	// Because go lang is a pain in the ass if you read the body then any susequent calls
	// are unable to read the body again....
	request.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	return json.NewDecoder(ioutil.NopCloser(bytes.NewBuffer(body)))
}