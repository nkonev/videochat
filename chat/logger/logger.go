package logger

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/trace"
	"os"
	"time"
)

// should be after viper
func NewLogger() *log.Logger {
	var logger = log.New()

	sl := viper.GetString("logger.level")
	pl, err := log.ParseLevel(sl)
	if err == nil {
		logger.SetLevel(pl)
	} else {
		logger.Errorf("Unable to parse log level from %v", sl)
	}

	logger.SetReportCaller(true)
	logger.SetFormatter(&log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
		FieldMap: log.FieldMap{
			log.FieldKeyTime:  "@timestamp",
			log.FieldKeyLevel: "level",
			log.FieldKeyMsg:   "message",
			log.FieldKeyFunc:  "caller",
		},
		PrettyPrint: true,
	})
	logger.SetOutput(os.Stdout)

	return logger
}

func GetLogEntry(context context.Context, lgr *log.Logger) *log.Entry {
	if p := trace.SpanFromContext(context); p != nil {
		return lgr.WithFields(
			log.Fields{
				"traceId": p.SpanContext().TraceID(),
			})
	} else {
		return lgr.WithContext(context)
	}
}
