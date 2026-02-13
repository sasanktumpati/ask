package cli

import "fmt"

// showTopicHelpIfRequested prints topic help when args[idx] is a help token.
// It returns true when help was printed.
func (a *App) showTopicHelpIfRequested(topic string, args []string, idx int) bool {
	if idx < 0 || idx >= len(args) {
		return false
	}
	if !isHelpToken(args[idx]) {
		return false
	}
	printHelp(a.stdout, topic, a.cfgPath)
	return true
}

// showTopicHelpIfAnyFlagRequested prints topic help when any token from
// args[start:] is a help flag. It returns true when help was printed.
func (a *App) showTopicHelpIfAnyFlagRequested(topic string, args []string, start int) bool {
	if start < 0 || start >= len(args) {
		return false
	}
	if !containsHelpFlag(args[start:]) {
		return false
	}
	printHelp(a.stdout, topic, a.cfgPath)
	return true
}

func usageError(format string, args ...any) error {
	return fmt.Errorf("usage: "+format, args...)
}

func unknownSubcommand(command string, sub string) error {
	return fmt.Errorf("unknown %s subcommand %q", command, sub)
}
