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
	})

	logWriteToFile := viper.GetBool("logger.writeToFile")
	if logWriteToFile {
		logDir := viper.GetString("logger.dir")

		err = os.MkdirAll(logDir, os.ModePerm)
		if err != nil {
			panic(err)
		}

		logFilename := viper.GetString("logger.filename")
		logFileVar, err = os.Create(logDir + string(os.PathSeparator) + logFilename)
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
		logFileVar.Close()
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
