package logger

import (
	"context"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"nkonev.name/notification/app"
	"os"
	"time"
)

type Logger struct {
	*zap.SugaredLogger
	ZapLogger *zap.Logger
	file      *os.File
}

func Iso3339CleanTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000000000Z")
}

func Iso3339CleanTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(Iso3339CleanTime(t))
}

// should be after viper
func NewLogger() *Logger {
	ec := zapcore.EncoderConfig{
		TimeKey:        "@timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     Iso3339CleanTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	enc := zapcore.NewJSONEncoder(ec)

	sl := viper.GetString("logger.level")
	lvl, err := zapcore.ParseLevel(sl)
	if err != nil {
		panic(err)
	}

	var ws zapcore.WriteSyncer
	var fileVar *os.File

	logWriteToFile := viper.GetBool("logger.writeToFile")
	if logWriteToFile {
		logDir := viper.GetString("logger.dir")

		err = os.MkdirAll(logDir, os.ModePerm)
		if err != nil {
			panic(err)
		}

		logFilename := viper.GetString("logger.filename")
		fileVar, err = os.Create(logDir + string(os.PathSeparator) + logFilename)
		if err != nil {
			panic(err)
		}
		ws = zap.CombineWriteSyncers(fileVar, os.Stdout)
	} else {
		ws = os.Stdout
	}

	co := zapcore.NewCore(enc, ws, lvl)

	zl := zap.New(co,
		zap.WithCaller(true),
		zap.Fields(zap.String("service", app.APP_NAME)),
	)

	le := zl.Sugar()

	return &Logger{
		SugaredLogger: le,
		file:          fileVar,
		ZapLogger:     zl,
	}
}

func (l *Logger) CloseLogger() {
	l.Sync()
	if l.file != nil {
		l.file.Close()
	}
}

func (l *Logger) WithTracing(context context.Context) *zap.SugaredLogger {
	if p := trace.SpanFromContext(context); p != nil {
		return l.With(
			zap.String("trace_id", p.SpanContext().TraceID().String()),
			zap.String("span_id", p.SpanContext().SpanID().String()),
		)
	} else {
		return l.SugaredLogger
	}
}

func (l *Logger) Write(p []byte) (int, error) {
	l.Infof(string(p))
	return len(p), nil
}
