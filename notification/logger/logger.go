package logger

import (
	"context"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
	"os"
)

var Logger = log.New()

func init() {
	Logger.SetReportCaller(true)
	Logger.SetFormatter(&log.TextFormatter{ForceColors: true, FullTimestamp: true})
	Logger.SetOutput(os.Stdout)
}

func GetLogEntry(context context.Context) *log.Entry {
	if p := trace.SpanFromContext(context); p != nil {
		return Logger.WithFields(
			log.Fields{
				"traceId": p.SpanContext().TraceID(),
			})
	} else {
		return Logger.WithContext(context)
	}
}
