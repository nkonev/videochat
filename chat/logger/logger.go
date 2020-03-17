package logger

import (
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
)

var Logger = log.New()

func init() {
	Logger.SetReportCaller(true)
	Logger.SetFormatter(&log.TextFormatter{ForceColors: true, FullTimestamp: true})
	Logger.SetOutput(os.Stdout)
}

func GetLogEntry(request *http.Request) *log.Entry {
	traceId := request.Header.Get("X-B3-Traceid")
	return Logger.WithFields(
		log.Fields{
			"traceId": traceId,
		})
}
