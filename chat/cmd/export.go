package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"nkonev.name/chat/app"
	"nkonev.name/chat/config"
	"nkonev.name/chat/kafka"
	"nkonev.name/chat/logger"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

const CommandExportName = "export"

func RunExport(args []string) {
	processedArgs, hasHelp := app.IsHelp(args)
	if hasHelp {
		fmt.Printf(`
Performs export of CQRS Kafka events topic to the json line file.
See cqrs.export.file setting. This settings along with /path/to/file.json also accepts a special '%s' pseudofile.
So all the logs are written to stderr.

To export to the file:
./%s %s --cqrs.export.file=/tmp/export.jsonl

or via pipe:
./%s %s --cqrs.export.file=%s > /tmp/export.jsonl

To export to stdout:
./%s %s --cqrs.export.file=%s

`, app.PseudoFileStdout,
			ExecutableName, CommandExportName,
			ExecutableName, CommandExportName, app.PseudoFileStdout,
			ExecutableName, CommandExportName, app.PseudoFileStdout,
		)

		return
	}

	cfg, err := config.CreateTypedConfig(processedArgs)
	if err != nil {
		panic(err)
	}
	lgr := logger.NewLogger(os.Stderr, cfg)
	defer lgr.CloseLogger()

	lgr.Info("Start export command")

	appFx := fx.New(
		fx.Supply(cfg),
		fx.Supply(lgr),
		fx.WithLogger(func(lgr *logger.LoggerWrapper) fxevent.Logger {
			fsl := &fxevent.SlogLogger{Logger: lgr.Logger}
			fsl.UseLogLevel(slog.LevelDebug)
			return fsl
		}),
		fx.Provide(
			kafka.ConfigureKafkaAdmin,
		),
		fx.Invoke(
			kafka.Export,
			app.Shutdown,
		),
	)
	appFx.Run()
	lgr.Info("Exit export command")
}
