package logger

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/trace"
	"io"
	"os"
	"time"
)

var logFileVar *os.File

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

	logFilename := viper.GetString("logger.filename")
	logWriteToFile := viper.GetBool("logger.writeToFile")
	if len(logFilename) > 0 && logWriteToFile {
		logFileVar, err = os.Create(logFilename)
		if err != nil {
			panic(err)
		}
		mw := io.MultiWriter(os.Stdout, logFileVar)
		logger.SetOutput(mw)
	} else {
		logger.SetOutput(os.Stdout)
	}

	return logger
}

func CloseLogger() {
	if logFileVar != nil {
		fmt.Println("Closing log file")
		if err := logFileVar.Close(); err != nil {
			panic(err)
		}
	}
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
