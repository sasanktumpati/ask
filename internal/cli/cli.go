package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"

	"ask/internal/assistant"
	"ask/internal/config"
	"ask/internal/providers"
	"ask/internal/render"
	"ask/internal/runner"

	"golang.org/x/term"
)

var errShowHelp = errors.New("show help")

// App encapsulates CLI runtime dependencies and loaded configuration.
type App struct {
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
	cfgPath string
	cfg     *config.Config
}

// Run executes the ask CLI with the provided process arguments and streams.
func Run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	if stdin == nil {
		stdin = os.Stdin
	}
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	global, rest, err := parseGlobalArgs(args)
	if err != nil {
		return err
	}

	cfgPath, err := config.ResolvePath(global.ConfigPath)
	if err != nil {
		return err
	}
	templatePath := config.TemplatePathForConfig(cfgPath)
	if err := config.EnsureTemplate(templatePath); err != nil {
		return err
	}

	cfg, loadErr := config.Load(cfgPath)
	if loadErr != nil && !errors.Is(loadErr, config.ErrConfigNotFound) {
		return loadErr
	}
	if errors.Is(loadErr, config.ErrConfigNotFound) {
		if err := config.Save(cfgPath, cfg); err != nil {
			return err
		}
	}

	app := &App{stdin: stdin, stdout: stdout, stderr: stderr, cfgPath: cfgPath, cfg: cfg}
	if global.ShowVersion {
		fmt.Fprintln(app.stdout, version)
		return nil
	}
	if global.ShowHelp {
		helpArgs := append([]string{"help"}, rest...)
		return app.dispatch(helpArgs)
	}
	return app.dispatch(rest)
}

func (a *App) dispatch(args []string) error {
	if len(args) == 0 {
		printHelp(a.stdout, "", a.cfgPath)
		return nil
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	switch sub {
	case "help":
		topic := ""
		if len(args) > 1 {
			topic = strings.ToLower(strings.TrimSpace(args[1]))
		}
		printHelp(a.stdout, topic, a.cfgPath)
		return nil
	case "-h", "--help":
		printHelp(a.stdout, "", a.cfgPath)
		return nil
	case "version", "--version", "-v":
		fmt.Fprintln(a.stdout, version)
		return nil
	case "models", "model":
		return a.runModels(args[1:])
	case "provider", "providers":
		return a.runProviders(args[1:])
	case "key", "keys":
		return a.runKeys(args[1:])
	case "config":
		return a.runConfig(args[1:])
	case "markdown":
		return a.runMarkdown(args[1:])
	default:
		return a.runAsk(args)
	}
}

func (a *App) runAsk(args []string) error {
	opts, question, err := parseAskArgs(args)
	if err != nil {
		if errors.Is(err, errShowHelp) {
			printHelp(a.stdout, "ask", a.cfgPath)
			return nil
		}
		return err
	}

	provider := strings.ToLower(strings.TrimSpace(opts.Provider))
	if provider == "" {
		provider = strings.ToLower(strings.TrimSpace(a.cfg.CurrentProvider))
	}
	if provider == "" {
		return fmt.Errorf("no default provider set; run `ask provider set <name>` or pass --provider")
	}
	if !a.cfg.ProviderExists(provider) {
		return fmt.Errorf("provider %q is not configured", provider)
	}

	model := strings.TrimSpace(opts.Model)
	if model == "" {
		model = strings.TrimSpace(a.cfg.GetModel(provider))
	}

	client, err := a.newClient(provider)
	if err != nil {
		return err
	}

	if model == "" {
		models, listErr := client.ListModels(context.Background())
		if listErr != nil {
			return fmt.Errorf("no model set for provider %q and unable to list models: %w", provider, listErr)
		}
		if len(models) == 0 {
			return fmt.Errorf("no models available for provider %q", provider)
		}
		model = selectDefaultModel(models)
		a.cfg.SetModel(provider, model)
		if err := a.saveConfig(); err != nil {
			return err
		}
	}

	shell := strings.TrimSpace(os.Getenv("SHELL"))
	if shell == "" {
		shell = "sh"
	}
	cwd, _ := os.Getwd()
	renderMarkdown := a.cfg.RenderMarkdown && !opts.NoMarkdown
	prompt := assistant.BuildPrompt(shell, cwd, runtime.GOOS, renderMarkdown)

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	resp, err := client.Ask(ctx, providers.AskRequest{
		Model:      model,
		Prompt:     prompt,
		Question:   question,
		ExpectJSON: true,
	})
	if err != nil {
		return err
	}

	parsed, parseErr := assistant.Parse(resp.Text)
	if parseErr != nil {
		parsed = fallbackAssistantResponse(resp.Text)
	}

	if opts.AsJSON {
		out := map[string]any{
			"provider": provider,
			"model":    model,
			"question": question,
			"answer":   parsed.Answer,
			"command":  parsed.Command,
		}
		enc := json.NewEncoder(a.stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}
	if parsed.Answer != "" {
		width := terminalWidth(a.stdout)
		fmt.Fprintln(a.stdout, render.Markdown(parsed.Answer, width, renderMarkdown))
	}

	if parsed.HasCommand() {
		if opts.NoRun {
			fmt.Fprintln(a.stdout)
			fmt.Fprintln(a.stdout, parsed.Command)
			return nil
		}
		if err := runner.PromptAndRun(runner.RunOptions{
			Command: parsed.Command,
			Stdin:   a.stdin,
			Stdout:  a.stdout,
			Stderr:  a.stderr,
		}); err != nil {
			return err
		}
	}

	if parseErr != nil {
		fmt.Fprintln(a.stderr, "warning: provider response was not strict JSON; used fallback parser")
	}
	return nil
}

func (a *App) newClient(provider string) (providers.Client, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	apiKey := a.cfg.ResolveAPIKey(provider)
	if custom, ok := a.cfg.CustomProviders[provider]; ok {
		settings := providers.OpenAICompatibleSettings{
			Name:       provider,
			ModelsPath: custom.ModelsPath,
			ChatPath:   custom.ChatPath,
			AuthHeader: custom.AuthHeader,
			AuthPrefix: custom.AuthPrefix,
		}
		return providers.NewOpenAICompatible(settings, providers.ClientOptions{
			APIKey:  apiKey,
			BaseURL: custom.BaseURL,
			Headers: custom.Headers,
		})
	}
	return providers.New(provider, providers.ClientOptions{
		APIKey:  apiKey,
		BaseURL: a.cfg.ResolveBaseURL(provider),
	})
}

func (a *App) saveConfig() error {
	return config.Save(a.cfgPath, a.cfg)
}

func terminalWidth(w io.Writer) int {
	const fallback = 100
	fdw, ok := w.(interface{ Fd() uintptr })
	if !ok {
		return fallback
	}
	fd := int(fdw.Fd())
	if !term.IsTerminal(fd) {
		return fallback
	}
	width, _, err := term.GetSize(fd)
	if err != nil || width <= 0 {
		return fallback
	}
	return width
}

func selectDefaultModel(models []providers.Model) string {
	if len(models) == 0 {
		return ""
	}
	preferred := []string{"mini", "flash", "haiku", "small", "8b"}
	for _, token := range preferred {
		for _, m := range models {
			if strings.Contains(strings.ToLower(m.ID), token) {
				return m.ID
			}
		}
	}
	copyModels := make([]providers.Model, len(models))
	copy(copyModels, models)
	sort.Slice(copyModels, func(i, j int) bool { return copyModels[i].ID < copyModels[j].ID })
	return copyModels[0].ID
}

func readLine(reader io.Reader, writer io.Writer, prompt string) (string, error) {
	fmt.Fprint(writer, prompt)
	buffer := bufio.NewReader(reader)
	line, err := buffer.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func parseKV(input string) (string, string, error) {
	idx := strings.Index(input, "=")
	if idx <= 0 || idx == len(input)-1 {
		return "", "", fmt.Errorf("expected key=value")
	}
	k := strings.TrimSpace(input[:idx])
	v := strings.TrimSpace(input[idx+1:])
	if k == "" || v == "" {
		return "", "", fmt.Errorf("expected key=value")
	}
	return k, v, nil
}

func parseAssistantFallbackFromCodeBlock(text string) string {
	candidate := strings.TrimSpace(text)
	if candidate == "" {
		return ""
	}
	start := strings.Index(candidate, "```")
	if start < 0 {
		for _, line := range strings.Split(candidate, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "$ ") {
				return strings.TrimSpace(strings.TrimPrefix(trimmed, "$ "))
			}
		}
		return ""
	}
	end := strings.Index(candidate[start+3:], "```")
	if end < 0 {
		return ""
	}
	block := candidate[start+3 : start+3+end]
	block = strings.TrimSpace(block)
	lines := strings.Split(block, "\n")
	if len(lines) == 0 {
		return ""
	}
	if strings.HasPrefix(lines[0], "bash") || strings.HasPrefix(lines[0], "sh") || strings.HasPrefix(lines[0], "zsh") {
		lines = lines[1:]
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return strings.TrimPrefix(trimmed, "$ ")
		}
	}
	return ""
}

func fallbackAssistantResponse(text string) assistant.Response {
	text = strings.TrimSpace(text)
	if text == "" {
		return assistant.Response{}
	}
	cmd := parseAssistantFallbackFromCodeBlock(text)
	return assistant.Response{Answer: text, Command: cmd}
}
