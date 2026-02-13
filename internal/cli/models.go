package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"

	"ask/internal/providers"
)

func (a *App) runModels(args []string) error {
	if len(args) == 0 {
		return a.listModels("", "")
	}
	if a.showTopicHelpIfRequested("models", args, 0) {
		return nil
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	switch sub {
	case "list":
		if a.showTopicHelpIfAnyFlagRequested("models", args, 1) {
			return nil
		}
		provider, search, rest, err := parseProviderSearch(args[1:])
		if err != nil {
			return err
		}
		if len(rest) > 0 {
			search = strings.Join(rest, " ")
		}
		return a.listModels(provider, search)
	case "current":
		if a.showTopicHelpIfAnyFlagRequested("models", args, 1) {
			return nil
		}
		provider, _, rest, err := parseProviderSearch(args[1:])
		if err != nil {
			return err
		}
		if len(rest) > 0 {
			return fmt.Errorf("unexpected arguments: %s", strings.Join(rest, " "))
		}
		return a.currentModel(provider)
	case "set":
		if a.showTopicHelpIfAnyFlagRequested("models", args, 1) {
			return nil
		}
		provider, _, rest, err := parseProviderSearch(args[1:])
		if err != nil {
			return err
		}
		if len(rest) == 0 {
			return usageError("ask models set <model> [--provider <name>]")
		}
		return a.setModel(provider, strings.Join(rest, " "))
	case "select":
		if a.showTopicHelpIfAnyFlagRequested("models", args, 1) {
			return nil
		}
		provider, search, rest, err := parseProviderSearch(args[1:])
		if err != nil {
			return err
		}
		if len(rest) > 0 {
			search = strings.Join(rest, " ")
		}
		return a.selectModel(provider, search)
	default:
		provider, search, rest, err := parseProviderSearch(args)
		if err != nil {
			return err
		}
		if len(rest) > 0 {
			search = strings.Join(rest, " ")
		}
		return a.listModels(provider, search)
	}
}

func (a *App) listModels(providerInput string, search string) error {
	provider, err := a.resolveProvider(providerInput)
	if err != nil {
		return err
	}
	client, err := a.newClient(provider)
	if err != nil {
		return err
	}

	models, err := client.ListModels(context.Background())
	if err != nil {
		return err
	}
	models = filterModels(models, search)
	if len(models) == 0 {
		if strings.TrimSpace(search) == "" {
			fmt.Fprintf(a.stdout, "no models found for provider %s\n", provider)
		} else {
			fmt.Fprintf(a.stdout, "no models found for provider %s matching %q\n", provider, search)
		}
		return nil
	}

	current := a.cfg.GetModel(provider)
	tw := tabwriter.NewWriter(a.stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintf(tw, "Provider:\t%s\n", provider)
	fmt.Fprintf(tw, "Models:\t%d\n", len(models))
	if strings.TrimSpace(search) != "" {
		fmt.Fprintf(tw, "Search:\t%q\n", search)
	}
	fmt.Fprintln(tw)
	fmt.Fprintln(tw, "CURRENT\tMODEL\tDISPLAY")
	for _, model := range models {
		marker := ""
		if model.ID == current {
			marker = "*"
		}
		display := model.DisplayName
		if display == model.ID {
			display = ""
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", marker, model.ID, display)
	}
	return tw.Flush()
}

func (a *App) currentModel(providerInput string) error {
	provider, err := a.resolveProvider(providerInput)
	if err != nil {
		return err
	}
	model := strings.TrimSpace(a.cfg.GetModel(provider))
	if model == "" {
		fmt.Fprintf(a.stdout, "provider=%s model=<not set>\n", provider)
		return nil
	}
	fmt.Fprintf(a.stdout, "provider=%s model=%s\n", provider, model)
	return nil
}

func (a *App) setModel(providerInput string, model string) error {
	provider, err := a.resolveProvider(providerInput)
	if err != nil {
		return err
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return fmt.Errorf("model cannot be empty")
	}
	a.cfg.SetModel(provider, model)
	if err := a.saveConfig(); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "set model for %s to %s\n", provider, model)
	return nil
}

func (a *App) selectModel(providerInput string, search string) error {
	provider, err := a.resolveProvider(providerInput)
	if err != nil {
		return err
	}
	client, err := a.newClient(provider)
	if err != nil {
		return err
	}
	models, err := client.ListModels(context.Background())
	if err != nil {
		return err
	}
	if len(models) == 0 {
		return fmt.Errorf("no models available for %s", provider)
	}

	activeSearch := strings.TrimSpace(search)
	for {
		filtered := filterModels(models, activeSearch)
		if len(filtered) == 0 {
			fmt.Fprintf(a.stdout, "no models match %q\n", activeSearch)
		} else {
			fmt.Fprintf(a.stdout, "provider=%s models=%d\n", provider, len(filtered))
			limit := len(filtered)
			if limit > 40 {
				limit = 40
			}
			for i := 0; i < limit; i++ {
				fmt.Fprintf(a.stdout, "%2d. %s\n", i+1, filtered[i].ID)
			}
			if len(filtered) > limit {
				fmt.Fprintf(a.stdout, "... %d more models hidden\n", len(filtered)-limit)
			}
		}

		fmt.Fprintln(a.stdout, "Type number to select, /text to search, empty to refresh, q to cancel")
		line, err := readLine(a.stdin, a.stdout, "select> ")
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "q" || line == "quit" {
			fmt.Fprintln(a.stdout, "selection cancelled")
			return nil
		}
		if strings.HasPrefix(line, "/") {
			activeSearch = strings.TrimSpace(strings.TrimPrefix(line, "/"))
			continue
		}
		if line == "" {
			continue
		}
		n, err := strconv.Atoi(line)
		if err != nil || n <= 0 || n > len(filtered) {
			fmt.Fprintln(a.stdout, "invalid selection")
			continue
		}

		chosen := filtered[n-1].ID
		a.cfg.SetModel(provider, chosen)
		if err := a.saveConfig(); err != nil {
			return err
		}
		fmt.Fprintf(a.stdout, "set model for %s to %s\n", provider, chosen)
		return nil
	}
}

func parseProviderSearch(args []string) (provider string, search string, rest []string, err error) {
	rest, err = scanOptions(args, []optionSpec{
		{
			Names:      []string{"provider", "p"},
			TakesValue: true,
			Set: func(v string) error {
				provider = strings.TrimSpace(v)
				return nil
			},
		},
		{
			Names:      []string{"search", "s"},
			TakesValue: true,
			Set: func(v string) error {
				search = strings.TrimSpace(v)
				return nil
			},
		},
	})
	return provider, search, rest, err
}

func filterModels(models []providers.Model, query string) []providers.Model {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return models
	}
	filtered := make([]providers.Model, 0, len(models))
	for _, m := range models {
		id := strings.ToLower(m.ID)
		name := strings.ToLower(m.DisplayName)
		if strings.Contains(id, query) || strings.Contains(name, query) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func (a *App) resolveProvider(providerInput string) (string, error) {
	provider := strings.ToLower(strings.TrimSpace(providerInput))
	if provider == "" {
		provider = strings.ToLower(strings.TrimSpace(a.cfg.CurrentProvider))
	}
	if provider == "" {
		return "", fmt.Errorf("no default provider set; run `ask provider set <name>` or pass --provider")
	}
	if !a.cfg.ProviderExists(provider) {
		return "", fmt.Errorf("provider %q is not configured", provider)
	}
	return provider, nil
}
