package main

import (
	"fmt"
	"os"

	"nkonev.name/chat/app"
	"nkonev.name/chat/cmd"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf(`No command provided.
Expected command is one of %s.
Also %s or %s is supported, e. g.:

%s %s %s

`, cmd.AllCommands.String(),
			app.HelpLongPrefix, app.HelpShortPrefix,
			cmd.ExecutableName, cmd.CommandServeName, app.HelpLongPrefix,
		)
		os.Exit(1)
	}

	theCmd := os.Args[1]
	remainingArgs := os.Args[2:]

	switch theCmd {
	case cmd.CommandImportName:
		cmd.RunImport(remainingArgs)
	case cmd.CommandExportName:
		cmd.RunExport(remainingArgs)
	case cmd.CommandResetName:
		cmd.RunReset(remainingArgs)
	case cmd.CommandHelpName:
		cmd.RunHelp(remainingArgs)
	case cmd.CommandRewindName:
		cmd.RunRewind(remainingArgs)
	case cmd.CommandServeName:
		cmd.RunServe(remainingArgs)
	case cmd.CommandMigrateName:
		cmd.RunMigrate(remainingArgs)
	default:
		fmt.Printf("Unknown command '%v'. Expected command is one of %s.\n", theCmd, cmd.AllCommands.String())
		os.Exit(1)
	}
}
