package cli

import (
	"fmt"
	"strings"
)

type optionSpec struct {
	Names      []string
	TakesValue bool
	Set        func(string) error
}

func scanOptions(args []string, specs []optionSpec) ([]string, error) {
	index := map[string]optionSpec{}
	for _, spec := range specs {
		for _, name := range spec.Names {
			key := strings.TrimSpace(name)
			if key == "" {
				continue
			}
			index[key] = spec
		}
	}

	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			continue
		}
		if arg == "--" {
			rest = append(rest, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			rest = append(rest, args[i])
			continue
		}

		name, value, hasValue := parseOptionToken(arg)
		spec, ok := index[name]
		if !ok {
			return nil, fmt.Errorf("unknown option %q (use --help)", arg)
		}

		if spec.TakesValue {
			if !hasValue {
				if i+1 >= len(args) {
					return nil, fmt.Errorf("%s requires a value", formatFlagName(name))
				}
				i++
				value = args[i]
			}
			if strings.TrimSpace(value) == "" {
				return nil, fmt.Errorf("%s requires a non-empty value", formatFlagName(name))
			}
			if spec.Set != nil {
				if err := spec.Set(value); err != nil {
					return nil, err
				}
			}
			continue
		}

		if hasValue {
			return nil, fmt.Errorf("%s does not accept a value", formatFlagName(name))
		}
		if spec.Set != nil {
			if err := spec.Set(""); err != nil {
				return nil, err
			}
		}
	}

	return rest, nil
}

func parseOptionToken(arg string) (name, value string, hasValue bool) {
	if strings.HasPrefix(arg, "--") {
		trimmed := strings.TrimPrefix(arg, "--")
		if idx := strings.IndexByte(trimmed, '='); idx >= 0 {
			return trimmed[:idx], trimmed[idx+1:], true
		}
		return trimmed, "", false
	}
	trimmed := strings.TrimPrefix(arg, "-")
	if idx := strings.IndexByte(trimmed, '='); idx >= 0 {
		return trimmed[:idx], trimmed[idx+1:], true
	}
	return trimmed, "", false
}

func formatFlagName(name string) string {
	if len(name) == 1 {
		return "-" + name
	}
	return "--" + name
}

func isHelpToken(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "help" || value == "-h" || value == "--help"
}

func containsHelpFlag(args []string) bool {
	for _, raw := range args {
		arg := strings.TrimSpace(raw)
		if isHelpToken(arg) {
			return true
		}
	}
	return false
}
