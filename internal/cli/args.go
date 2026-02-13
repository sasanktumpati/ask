package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type globalOptions struct {
	ConfigPath  string
	ShowHelp    bool
	ShowVersion bool
}

type askOptions struct {
	Provider   string
	Model      string
	NoMarkdown bool
	NoRun      bool
	AsJSON     bool
	Timeout    time.Duration
}

func parseGlobalArgs(args []string) (globalOptions, []string, error) {
	opts := globalOptions{}
	i := 0

	for i < len(args) {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			i++
			continue
		}
		if arg == "--" {
			i++
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			break
		}

		name, value, hasValue := parseOptionToken(arg)
		switch name {
		case "config", "c":
			if !hasValue {
				if i+1 >= len(args) {
					return opts, nil, fmt.Errorf("%s requires a value", formatFlagName(name))
				}
				i++
				value = args[i]
			}
			value = strings.TrimSpace(value)
			if value == "" {
				return opts, nil, fmt.Errorf("%s requires a non-empty value", formatFlagName(name))
			}
			opts.ConfigPath = value
		case "help", "h":
			opts.ShowHelp = true
		case "version", "v":
			opts.ShowVersion = true
		default:
			return opts, args[i:], nil
		}
		i++
	}

	return opts, args[i:], nil
}

func parseAskArgs(args []string) (askOptions, string, error) {
	opts := askOptions{Timeout: 90 * time.Second}
	showHelp := false

	rest, err := scanOptions(args, []optionSpec{
		{Names: []string{"help", "h"}, TakesValue: false, Set: func(string) error { showHelp = true; return nil }},
		{Names: []string{"provider", "p"}, TakesValue: true, Set: func(v string) error { opts.Provider = strings.TrimSpace(v); return nil }},
		{Names: []string{"model", "m"}, TakesValue: true, Set: func(v string) error { opts.Model = strings.TrimSpace(v); return nil }},
		{Names: []string{"timeout"}, TakesValue: true, Set: func(v string) error {
			d, err := parseDuration(v)
			if err != nil {
				return fmt.Errorf("--timeout: %w", err)
			}
			opts.Timeout = d
			return nil
		}},
		{Names: []string{"no-markdown"}, TakesValue: false, Set: func(string) error { opts.NoMarkdown = true; return nil }},
		{Names: []string{"no-run"}, TakesValue: false, Set: func(string) error { opts.NoRun = true; return nil }},
		{Names: []string{"json"}, TakesValue: false, Set: func(string) error { opts.AsJSON = true; return nil }},
	})
	if err != nil {
		return opts, "", err
	}
	if showHelp {
		return opts, "", errShowHelp
	}

	question := strings.TrimSpace(strings.Join(rest, " "))
	if question == "" {
		return opts, "", fmt.Errorf("question is required")
	}
	return opts, question, nil
}

func parseDuration(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("timeout value is empty")
	}
	if strings.ContainsAny(raw, "hms") {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return 0, err
		}
		if d <= 0 {
			return 0, fmt.Errorf("timeout must be positive")
		}
		return d, nil
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return 0, fmt.Errorf("timeout must be a positive integer seconds or duration")
	}
	return time.Duration(seconds) * time.Second, nil
}
