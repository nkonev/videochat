package cmd

import (
	"strings"
)

type Commands []string

var AllCommands = Commands{
	CommandExportName,
	CommandImportName,
	CommandResetName,
	CommandHelpName,
	CommandServeName,
	CommandRewindName,
}

func (c *Commands) String() string {
	bldr := strings.Builder{}

	if c != nil {
		for i, v := range *c {
			if i != 0 {
				bldr.WriteString(", ")
			}
			bldr.WriteString(v)
		}
	} else {
		bldr.WriteString("nil")
	}

	return bldr.String()
}
