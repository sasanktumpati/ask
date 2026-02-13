package cli

import (
	"fmt"
	"strings"

	"github.com/sasanktumpati/ask/internal/config"

	"golang.org/x/term"
)

func (a *App) runKeys(args []string) error {
	if len(args) == 0 || a.showTopicHelpIfRequested("key", args, 0) {
		return nil
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	switch sub {
	case "set":
		if a.showTopicHelpIfRequested("key", args, 1) {
			return nil
		}
		return a.keySet(args[1:])
	case "clear":
		if a.showTopicHelpIfRequested("key", args, 1) {
			return nil
		}
		return a.keyClear(args[1:])
	case "show":
		if a.showTopicHelpIfRequested("key", args, 1) {
			return nil
		}
		return a.keyShow(args[1:])
	default:
		return unknownSubcommand("key", sub)
	}
}

func (a *App) keySet(args []string) error {
	if len(args) == 0 {
		return usageError("ask key set <provider> [--value <key>] [--env <ENV_VAR>]")
	}

	provider := strings.ToLower(strings.TrimSpace(args[0]))
	if !a.cfg.ProviderExists(provider) {
		return fmt.Errorf("provider %q is not configured", provider)
	}

	var value string
	var envVar string
	rest, err := scanOptions(args[1:], []optionSpec{
		{Names: []string{"value"}, TakesValue: true, Set: func(v string) error { value = strings.TrimSpace(v); return nil }},
		{Names: []string{"env"}, TakesValue: true, Set: func(v string) error { envVar = strings.TrimSpace(v); return nil }},
	})
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(rest, " "))
	}

	if value == "" && envVar == "" {
		prompted, err := a.readSecret("API key: ")
		if err != nil {
			return err
		}
		value = prompted
	}

	if envVar != "" {
		a.cfg.SetAPIKeyEnv(provider, envVar)
	}
	if value != "" {
		a.cfg.SetAPIKey(provider, value)
	}
	if err := a.saveConfig(); err != nil {
		return err
	}

	msg := fmt.Sprintf("updated credentials for %s", provider)
	if envVar != "" {
		msg += fmt.Sprintf(" (env=%s)", envVar)
	}
	fmt.Fprintln(a.stdout, msg)
	return nil
}

func (a *App) keyClear(args []string) error {
	if len(args) == 0 {
		return usageError("ask key clear <provider>")
	}
	provider := strings.ToLower(strings.TrimSpace(args[0]))
	if !a.cfg.ProviderExists(provider) {
		return fmt.Errorf("provider %q is not configured", provider)
	}
	if _, ok := a.cfg.CustomProviders[provider]; ok {
		custom := a.cfg.CustomProviders[provider]
		custom.APIKey = ""
		custom.APIKeyEnv = ""
		a.cfg.CustomProviders[provider] = custom
	} else {
		a.cfg.SetAPIKey(provider, "")
		a.cfg.SetAPIKeyEnv(provider, "")
	}
	if err := a.saveConfig(); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "cleared credentials for %s\n", provider)
	return nil
}

func (a *App) keyShow(args []string) error {
	if len(args) == 0 {
		return usageError("ask key show <provider>")
	}
	provider := strings.ToLower(strings.TrimSpace(args[0]))
	if !a.cfg.ProviderExists(provider) {
		return fmt.Errorf("provider %q is not configured", provider)
	}

	resolved := a.cfg.ResolveAPIKey(provider)
	masked := "<empty>"
	if strings.TrimSpace(resolved) != "" {
		masked = maskForShow(resolved)
	}
	storage := "none"
	if custom, ok := a.cfg.CustomProviders[provider]; ok {
		if strings.TrimSpace(custom.APIKey) != "" {
			storage = "plain"
		}
	} else {
		pc := a.cfg.Providers[provider]
		if strings.TrimSpace(pc.APIKey) != "" {
			storage = "plain"
		}
	}
	envVar := ""
	if custom, ok := a.cfg.CustomProviders[provider]; ok {
		envVar = strings.TrimSpace(custom.APIKeyEnv)
	} else {
		envVar = strings.TrimSpace(a.cfg.Providers[provider].APIKeyEnv)
		if envVar == "" {
			if defaults, ok := builtinEnv(provider); ok {
				envVar = defaults
			}
		}
	}

	fmt.Fprintf(a.stdout, "provider=%s\n", provider)
	fmt.Fprintf(a.stdout, "api_key=%s\n", masked)
	fmt.Fprintf(a.stdout, "storage=%s\n", storage)
	if envVar != "" {
		fmt.Fprintf(a.stdout, "api_key_env=%s\n", envVar)
	}
	return nil
}

func (a *App) readSecret(prompt string) (string, error) {
	file, ok := a.stdin.(interface{ Fd() uintptr })
	if !ok {
		line, err := readLine(a.stdin, a.stdout, prompt)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(line), nil
	}
	fd := int(file.Fd())
	if !term.IsTerminal(fd) {
		line, err := readLine(a.stdin, a.stdout, prompt)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(line), nil
	}

	fmt.Fprint(a.stdout, prompt)
	bytes, err := term.ReadPassword(fd)
	fmt.Fprintln(a.stdout)
	if err != nil {
		return "", fmt.Errorf("read api key: %w", err)
	}
	return strings.TrimSpace(string(bytes)), nil
}

func maskForShow(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "<empty>"
	}
	if len(v) <= 6 {
		return "******"
	}
	return strings.Repeat("*", len(v)-4) + v[len(v)-4:]
}

func builtinEnv(provider string) (string, bool) {
	defaults, ok := config.BuiltinProviderDefaults(provider)
	if !ok || strings.TrimSpace(defaults.APIKeyEnv) == "" {
		return "", false
	}
	return defaults.APIKeyEnv, true
}
