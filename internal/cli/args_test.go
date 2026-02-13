package cli

import (
	"errors"
	"testing"
	"time"
)

func TestParseAskArgs_OptionsAnywhere(t *testing.T) {
	opts, q, err := parseAskArgs([]string{"-p", "openai", "how", "to", "reset", "commit", "--timeout", "30"})
	if err != nil {
		t.Fatalf("parseAskArgs error = %v", err)
	}
	if q != "how to reset commit" {
		t.Fatalf("question = %q", q)
	}
	if opts.Provider != "openai" {
		t.Fatalf("provider = %q", opts.Provider)
	}
	if opts.Timeout != 30*time.Second {
		t.Fatalf("timeout = %s", opts.Timeout)
	}
}

func TestParseAskArgs_ShortsWithEquals(t *testing.T) {
	opts, q, err := parseAskArgs([]string{"-p=ollama", "-m=llama3.2", "list", "all", "branches"})
	if err != nil {
		t.Fatalf("parseAskArgs error = %v", err)
	}
	if opts.Provider != "ollama" || opts.Model != "llama3.2" {
		t.Fatalf("opts = %+v", opts)
	}
	if q != "list all branches" {
		t.Fatalf("question = %q", q)
	}
}

func TestParseAskArgs_Help(t *testing.T) {
	_, _, err := parseAskArgs([]string{"--help"})
	if !errors.Is(err, errShowHelp) {
		t.Fatalf("expected errShowHelp, got %v", err)
	}
}

func TestParseAskArgs_MissingQuestion(t *testing.T) {
	_, _, err := parseAskArgs([]string{"--provider", "openai"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseGlobalArgs_ConfigAndRest(t *testing.T) {
	global, rest, err := parseGlobalArgs([]string{"--config", "/tmp/ask.json", "models", "list"})
	if err != nil {
		t.Fatalf("parseGlobalArgs error = %v", err)
	}
	if global.ConfigPath != "/tmp/ask.json" {
		t.Fatalf("config path = %q", global.ConfigPath)
	}
	if len(rest) != 2 || rest[0] != "models" || rest[1] != "list" {
		t.Fatalf("rest = %#v", rest)
	}
}

func TestParseGlobalArgs_HelpBeforeCommand(t *testing.T) {
	global, rest, err := parseGlobalArgs([]string{"-h", "models"})
	if err != nil {
		t.Fatalf("parseGlobalArgs error = %v", err)
	}
	if !global.ShowHelp {
		t.Fatal("expected ShowHelp=true")
	}
	if len(rest) != 1 || rest[0] != "models" {
		t.Fatalf("rest = %#v", rest)
	}
}
