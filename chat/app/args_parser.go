package app

import (
	"fmt"
	"slices"
	"strings"
)

const ConfigLongPrefix = "--config"
const ConfigShortPrefix = "-c"

const HelpLongPrefix = "--help"
const HelpShortPrefix = "-h"

const PseudoFileStdout = "stdout"
const PseudoFileStdin = "stdin"

func IsHelp(args []string) ([]string, bool) {
	var help bool
	var res = args

	longIdx := slices.Index(res, HelpLongPrefix)
	shortIdx := slices.Index(res, HelpShortPrefix)

	if longIdx != -1 {
		res = slices.Delete(res, longIdx, longIdx+1)
		help = true
	}

	if shortIdx != -1 {
		res = slices.Delete(res, shortIdx, shortIdx+1)
		help = true
	}

	return res, help
}

func IsConfig(args []string) (bool, string, []string, error) {
	longIdx := slices.IndexFunc(args, func(s string) bool {
		return strings.HasPrefix(s, ConfigLongPrefix)
	})
	shortIdx := slices.IndexFunc(args, func(s string) bool {
		return strings.HasPrefix(s, ConfigShortPrefix)
	})

	if longIdx != -1 || shortIdx != -1 {
		var argsToReadConfig []string
		var configPath string
		var err error

		if longIdx != -1 {
			argsToReadConfig, configPath, err = parseOutConfig(args, longIdx, ConfigLongPrefix)
		} else {
			argsToReadConfig, configPath, err = parseOutConfig(args, shortIdx, ConfigShortPrefix)
		}

		return true, configPath, argsToReadConfig, err
	} else {
		return false, "", args, nil
	}
}

func parseOutConfig(args []string, idx int, prefix string) ([]string, string, error) {
	if len(args) == 0 {
		return []string{}, "", fmt.Errorf("wrong invariant - parseOutConfig should be invoked when at least one argument exists")
	}

	if idx > len(args)-1 {
		return []string{}, "", fmt.Errorf("wrong invariant - index out of bound")
	}

	var argsToReadConfig []string

	stringWithConfig := args[idx]

	// -c=/path/to/file.yaml
	// -c /path/to/file.yaml
	var thePath = stringWithConfig

	// =/path/to/file.yaml
	// -c
	thePath, _ = strings.CutPrefix(thePath, prefix)

	if strings.HasPrefix(thePath, "=") { // =/path/to/file.yaml
		thePath, _ = strings.CutPrefix(thePath, "=")
		argsToReadConfig = append(args[:idx], args[idx+1:]...)
	} else { // -c
		if idx+2 > len(args) { // checks whether we can safely take argsToReadConfig
			return nil, "", fmt.Errorf("expected file argument")
		}
		thePath = args[idx+1]
		argsToReadConfig = append(args[:idx], args[idx+2:]...)
	}

	thePath = strings.TrimSpace(thePath)
	return argsToReadConfig, thePath, nil
}
