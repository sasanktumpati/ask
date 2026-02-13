package cli

import (
	"fmt"
	"os"
	"strings"

	"ask/internal/config"
)

func (a *App) runConfig(args []string) error {
	if len(args) == 0 {
		return a.configShow()
	}
	if a.showTopicHelpIfRequested("config", args, 0) {
		return nil
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	switch sub {
	case "show":
		if a.showTopicHelpIfRequested("config", args, 1) {
			return nil
		}
		return a.configShow()
	case "path":
		if a.showTopicHelpIfRequested("config", args, 1) {
			return nil
		}
		fmt.Fprintln(a.stdout, a.cfgPath)
		return nil
	case "template":
		if a.showTopicHelpIfRequested("config", args, 1) {
			return nil
		}
		fmt.Fprintln(a.stdout, config.TemplatePathForConfig(a.cfgPath))
		return nil
	default:
		return unknownSubcommand("config", sub)
	}
}

func (a *App) configShow() error {
	buf, err := os.ReadFile(a.cfgPath)
	if err != nil {
		return err
	}
	if _, err := a.stdout.Write(buf); err != nil {
		return err
	}
	if len(buf) == 0 || buf[len(buf)-1] != '\n' {
		fmt.Fprintln(a.stdout)
	}
	return nil
}
