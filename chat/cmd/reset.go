package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"nkonev.name/chat/app"
	"nkonev.name/chat/config"
	"nkonev.name/chat/cqrs"
	"nkonev.name/chat/db"
	"nkonev.name/chat/kafka"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/otel"
	"nkonev.name/chat/sanitizer"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

const CommandResetName = "reset"

func RunReset(args []string) {
	processedArgs, hasHelp := app.IsHelp(args)
	if hasHelp {
		fmt.Println(`
Performs reset the CQRS projections in PostgreSQL 
(not event store in Kafka, so it's relatively safe) 
and sets the 'need_to_fast_forward_sequences' task into "technical" table.
		`)

		return
	}

	cfg, err := config.CreateTypedConfig(processedArgs)
	if err != nil {
		panic(err)
	}
	lgr := logger.NewLogger(os.Stdout, cfg)
	defer lgr.CloseLogger()

	lgr.Info("Start reset command")

	appFx := fx.New(
		fx.Supply(cfg),
		fx.Supply(lgr),
		fx.WithLogger(func(lgr *logger.LoggerWrapper) fxevent.Logger {
			fsl := &fxevent.SlogLogger{Logger: lgr.Logger}
			fsl.UseLogLevel(slog.LevelDebug)
			return fsl
		}),
		fx.Provide(
			otel.ConfigureTracePropagator,
			otel.ConfigureTraceProvider,
			otel.ConfigureTraceExporter,
			db.ConfigureDatabase,
			kafka.ConfigureKafkaAdmin,
			cqrs.ConfigureCommonProjection,
			sanitizer.CreateStripTags,
		),
		fx.Invoke(
			db.RunResetDatabaseSoft,
			kafka.RunResetPartitionsChat,
			kafka.RunDeleteTopicUser,
			db.RunMigrations,
			kafka.RunCreateTopicChat,
			kafka.RunCreateTopicUser,
			cqrs.SetIsNeedToFastForwardSequences,
			app.Shutdown,
		),
	)
	appFx.Run()
	lgr.Info("Exit reset command")
}
