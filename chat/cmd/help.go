package cmd

import (
	"fmt"

	parser "github.com/nkonev/args-parser"
	"nkonev.name/chat/app"
)

const CommandHelpName = "help"

const ExecutableName = app.TRACE_RESOURCE

func RunHelp(args []string) {
	fmt.Printf(`SYNOPSIS:
%s <command> [[%s[=| ]|[%s[=| ]]/path/to/config.yml] [--help|-h] --some.option=overridedValue

Where command is one of %v.

The config reading order is:
* config file
* environment variables
* program arguments

E. g. you can override any value in the config file with environment variable or program argument.

Examples:

./%s %s %s=./config/config/config-dev.yml --logger.json=true
./%s %s %s ./config/config/config-dev.yml --server.address=:8888

./%s %s %s ./config/config/config-dev.yml --logger.level=debug --postgresql.prettyLog=false --logger.json=true
./%s %s %s=./config/config/config-dev.yml --server.dump=false --http.dump=false --postgresql.dump=false --cqrs.dump=false


To get the particular command's help, use
%s <command> [%s|%s]

Examples:
./%s %s %s

`, ExecutableName, parser.ConfigLongPrefix, parser.ConfigShortPrefix,
		AllCommands.String(),
		ExecutableName, CommandServeName, parser.ConfigLongPrefix,
		ExecutableName, CommandServeName, parser.ConfigLongPrefix,
		ExecutableName, CommandServeName, parser.ConfigShortPrefix,
		ExecutableName, CommandServeName, parser.ConfigShortPrefix,
		ExecutableName, parser.HelpLongPrefix, parser.HelpShortPrefix,
		ExecutableName, CommandServeName, parser.HelpLongPrefix,
	)
}
