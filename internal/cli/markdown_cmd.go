package cli

import (
	"fmt"
	"strings"
)

func (a *App) runMarkdown(args []string) error {
	if len(args) == 0 {
		return a.markdownStatus()
	}
	if a.showTopicHelpIfRequested("markdown", args, 0) {
		return nil
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	switch sub {
	case "on", "enable":
		if a.showTopicHelpIfRequested("markdown", args, 1) {
			return nil
		}
		a.cfg.RenderMarkdown = true
		if err := a.saveConfig(); err != nil {
			return err
		}
		fmt.Fprintln(a.stdout, "markdown rendering enabled")
		return nil
	case "off", "disable":
		if a.showTopicHelpIfRequested("markdown", args, 1) {
			return nil
		}
		a.cfg.RenderMarkdown = false
		if err := a.saveConfig(); err != nil {
			return err
		}
		fmt.Fprintln(a.stdout, "markdown rendering disabled")
		return nil
	case "status":
		if a.showTopicHelpIfRequested("markdown", args, 1) {
			return nil
		}
		return a.markdownStatus()
	default:
		return unknownSubcommand("markdown", sub)
	}
}

func (a *App) markdownStatus() error {
	status := "off"
	if a.cfg.RenderMarkdown {
		status = "on"
	}
	fmt.Fprintf(a.stdout, "markdown=%s\n", status)
	return nil
}
