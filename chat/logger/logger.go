package logger

import (
	"context"
	"io"
	"log/slog"
	"nkonev.name/chat/app"
	"nkonev.name/chat/config"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
)

func GetTraceId(ctx context.Context) string {
	sc := trace.SpanFromContext(ctx).SpanContext()
	tr := sc.TraceID()
	return tr.String()
}

type TracingContextHandler struct {
	slog.Handler
}

func (h *TracingContextHandler) Handle(ctx context.Context, r slog.Record) error {
	traceId := GetTraceId(ctx)
	if traceId != "" {
		r.AddAttrs(slog.String(AttributeTraceId, traceId))
	}

	return h.Handler.Handle(ctx, r)
}

type LoggerWrapper struct {
	*slog.Logger
	file *os.File
}

func NewLogger(consoleWriter io.Writer, cfg *config.AppConfig) *LoggerWrapper {
	var baseLogger *slog.Logger

	replaceFunc := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == "msg" {
			return slog.Attr{
				Key:   "message",
				Value: a.Value,
			}
		} else if a.Key == "time" {
			utcTime := time.Now().UTC()
			utcFormattedTime := utcTime.Format("2006-01-02T15:04:05.000000000Z")
			return slog.Attr{
				Key:   "@timestamp",
				Value: slog.AnyValue(utcFormattedTime),
			}
		} else if a.Key == "level" {
			return slog.Attr{
				Key:   "level",
				Value: slog.StringValue(strings.ToLower(a.Value.String())),
			}
		} else if a.Key == "err" {
			return slog.Attr{
				Key:   AttributeError,
				Value: a.Value,
			}
		} else {
			return a
		}
	}

	bh := &slog.HandlerOptions{
		Level:       cfg.Logger.GetLevel(),
		ReplaceAttr: replaceFunc,
		AddSource:   true,
	}
	commonAttrs := []slog.Attr{slog.String("service", app.TRACE_RESOURCE)}

	w := consoleWriter
	var fileVar *os.File
	if cfg.Logger.WriteToFile {
		logDir := cfg.Logger.Dir

		err := os.MkdirAll(logDir, os.ModePerm)
		if err != nil {
			panic(err)
		}

		logFilename := cfg.Logger.Filename
		fileVar, err = os.Create(logDir + string(os.PathSeparator) + logFilename)
		if err != nil {
			panic(err)
		}

		w = io.MultiWriter(consoleWriter, fileVar)
	}

	if cfg.Logger.Json {
		h := &TracingContextHandler{slog.NewJSONHandler(w, bh).WithAttrs(commonAttrs)}
		baseLogger = slog.New(h)
	} else {
		h := &TracingContextHandler{slog.NewTextHandler(w, bh).WithAttrs(commonAttrs)}
		baseLogger = slog.New(h)
	}

	return &LoggerWrapper{
		Logger: baseLogger,
		file:   fileVar,
	}
}

func (lw *LoggerWrapper) CloseLogger() {
	if lw.file != nil {
		lw.file.Close()
	}
}

// Do not use
func (lw *LoggerWrapper) WithTrace0(ctx context.Context) *slog.Logger {
	return lw.Logger.With(AttributeTraceId, GetTraceId(ctx))
}
