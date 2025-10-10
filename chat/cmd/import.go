package cmd

import (
	"fmt"
	"log/slog"
	"nkonev.name/chat/app"
	"nkonev.name/chat/config"
	"nkonev.name/chat/cqrs"
	"nkonev.name/chat/db"
	"nkonev.name/chat/kafka"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/otel"
	"nkonev.name/chat/sanitizer"
	"os"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

const CommandImportName = "import"

func RunImport(args []string) {
	processedArgs, hasHelp := app.IsHelp(args)
	if hasHelp {
		fmt.Printf(`
Performs import to the Kafka events topic from the json line file produced by "export" command
and sets the 'need_to_fast_forward_sequences' task into "technical" table.
And also, this command doesn't build projections.
To fast-forward Kafka offsets or "rewind" them, and build the projections,
use '%s' command.

See cqrs.import.file setting. This settings along with /path/to/file.json also accepts a special '%s' pseudofile.

To import from the file:
./%s %s --cqrs.import.file=/tmp/export.jsonl

To import from stdin via pipe:
cat /tmp/export.jsonl | ./%s %s --cqrs.import.file=%s

`, CommandRewindName,
			app.PseudoFileStdin,
			ExecutableName, CommandImportName,
			ExecutableName, CommandImportName, app.PseudoFileStdin,
		)

		return
	}

	cfg, err := config.CreateTypedConfig(processedArgs)
	if err != nil {
		panic(err)
	}
	lgr := logger.NewLogger(os.Stdout, cfg)
	defer lgr.CloseLogger()

	lgr.Info("Start import command")

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
			db.RunMigrations,
			kafka.RunCreateTopicChat,
			kafka.RunCreateTopicUser,
			kafka.Import,
			cqrs.SetIsNeedToFastForwardSequences,
			app.Shutdown,
		),
	)
	appFx.Run()
	lgr.Info("Exit import command")
}
