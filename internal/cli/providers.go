package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"ask/internal/config"
)

func (a *App) runProviders(args []string) error {
	if len(args) == 0 {
		return a.providerList()
	}
	if a.showTopicHelpIfRequested("provider", args, 0) {
		return nil
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	switch sub {
	case "list":
		return a.providerList()
	case "current":
		fmt.Fprintln(a.stdout, a.cfg.CurrentProvider)
		return nil
	case "set":
		if a.showTopicHelpIfRequested("provider", args, 1) {
			return nil
		}
		if len(args) < 2 {
			return usageError("ask provider set <name>")
		}
		name := strings.ToLower(strings.TrimSpace(args[1]))
		if !a.cfg.ProviderExists(name) {
			return fmt.Errorf("provider %q is not configured", name)
		}
		a.cfg.SetCurrentProvider(name)
		if err := a.saveConfig(); err != nil {
			return err
		}
		fmt.Fprintf(a.stdout, "current provider set to %s\n", name)
		return nil
	case "add":
		if a.showTopicHelpIfRequested("provider", args, 1) {
			return nil
		}
		return a.providerAdd(args[1:])
	case "remove", "rm", "delete":
		if a.showTopicHelpIfRequested("provider", args, 1) {
			return nil
		}
		if len(args) < 2 {
			return usageError("ask provider remove <name>")
		}
		name := strings.ToLower(strings.TrimSpace(args[1]))
		if err := a.cfg.RemoveCustomProvider(name); err != nil {
			return err
		}
		if err := a.saveConfig(); err != nil {
			return err
		}
		fmt.Fprintf(a.stdout, "removed provider %s\n", name)
		return nil
	case "show", "inspect":
		if a.showTopicHelpIfRequested("provider", args, 1) {
			return nil
		}
		name := strings.TrimSpace(a.cfg.CurrentProvider)
		if len(args) > 1 {
			name = strings.TrimSpace(args[1])
		}
		return a.providerShow(name)
	default:
		return unknownSubcommand("provider", sub)
	}
}

func (a *App) providerList() error {
	names := a.cfg.ProviderNames()
	tw := tabwriter.NewWriter(a.stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "CURRENT\tNAME\tTYPE\tMODEL\tBASE_URL")
	for _, name := range names {
		marker := ""
		if name == a.cfg.CurrentProvider {
			marker = "*"
		}
		base := a.cfg.ResolveBaseURL(name)
		model := a.cfg.GetModel(name)
		ptype := "builtin"
		if _, ok := a.cfg.CustomProviders[name]; ok {
			ptype = "custom-openai-compatible"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", marker, name, ptype, model, base)
	}
	return tw.Flush()
}

func (a *App) providerShow(name string) error {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return fmt.Errorf("provider name is required")
	}
	if !a.cfg.ProviderExists(name) {
		return fmt.Errorf("provider %q is not configured", name)
	}

	type providerView struct {
		Name      string `json:"name"`
		Current   bool   `json:"current"`
		Model     string `json:"model,omitempty"`
		BaseURL   string `json:"base_url,omitempty"`
		APIKeyEnv string `json:"api_key_env,omitempty"`
		HasAPIKey bool   `json:"has_api_key"`
		Custom    bool   `json:"custom"`
	}

	view := providerView{
		Name:    name,
		Current: a.cfg.CurrentProvider == name,
		Model:   a.cfg.GetModel(name),
		BaseURL: a.cfg.ResolveBaseURL(name),
		Custom:  false,
	}

	if custom, ok := a.cfg.CustomProviders[name]; ok {
		view.Custom = true
		view.APIKeyEnv = custom.APIKeyEnv
		view.HasAPIKey = strings.TrimSpace(custom.APIKey) != ""
	} else {
		pc := a.cfg.Providers[name]
		view.APIKeyEnv = strings.TrimSpace(pc.APIKeyEnv)
		view.HasAPIKey = strings.TrimSpace(pc.APIKey) != ""
		if view.APIKeyEnv == "" {
			if defaults, ok := config.BuiltinProviderDefaults(name); ok {
				view.APIKeyEnv = defaults.APIKeyEnv
			}
		}
	}

	enc := json.NewEncoder(a.stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(view)
}

func (a *App) providerAdd(args []string) error {
	if len(args) == 0 {
		return usageError("ask provider add <name> --base-url <url> [options]")
	}
	name := strings.ToLower(strings.TrimSpace(args[0]))
	if name == "" {
		return fmt.Errorf("provider name is required")
	}

	input := config.OpenAICompatibleProvider{Headers: map[string]string{}}

	rest, err := scanOptions(args[1:], []optionSpec{
		{Names: []string{"base-url"}, TakesValue: true, Set: func(v string) error { input.BaseURL = strings.TrimSpace(v); return nil }},
		{Names: []string{"model"}, TakesValue: true, Set: func(v string) error { input.Model = strings.TrimSpace(v); return nil }},
		{Names: []string{"api-key"}, TakesValue: true, Set: func(v string) error { input.APIKey = strings.TrimSpace(v); return nil }},
		{Names: []string{"api-key-env"}, TakesValue: true, Set: func(v string) error { input.APIKeyEnv = strings.TrimSpace(v); return nil }},
		{Names: []string{"models-path"}, TakesValue: true, Set: func(v string) error { input.ModelsPath = strings.TrimSpace(v); return nil }},
		{Names: []string{"chat-path"}, TakesValue: true, Set: func(v string) error { input.ChatPath = strings.TrimSpace(v); return nil }},
		{Names: []string{"auth-header"}, TakesValue: true, Set: func(v string) error { input.AuthHeader = strings.TrimSpace(v); return nil }},
		{Names: []string{"auth-prefix"}, TakesValue: true, Set: func(v string) error { input.AuthPrefix = v; return nil }},
		{Names: []string{"header"}, TakesValue: true, Set: func(v string) error {
			k, val, err := parseKV(v)
			if err != nil {
				return fmt.Errorf("--header: %w", err)
			}
			input.Headers[k] = val
			return nil
		}},
	})
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(rest, " "))
	}

	if err := a.cfg.AddCustomProvider(name, input); err != nil {
		return err
	}
	if err := a.saveConfig(); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "added provider %s\n", name)
	return nil
}
