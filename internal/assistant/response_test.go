package assistant

import (
	"strings"
	"testing"
)

func TestParseStrictJSON(t *testing.T) {
	resp, err := Parse(`{"answer":"hello","command":"git status"}`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if resp.Answer != "hello" {
		t.Fatalf("Answer = %q, want hello", resp.Answer)
	}
	if resp.Command != "git status" {
		t.Fatalf("Command = %q, want git status", resp.Command)
	}
	if !resp.HasCommand() {
		t.Fatal("HasCommand() = false, want true")
	}
}

func TestParseEmbeddedJSON(t *testing.T) {
	in := "Result:\n```json\n{\"answer\":\"Use this\",\"command\":\"\"}\n```"
	resp, err := Parse(in)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if resp.Answer != "Use this" {
		t.Fatalf("Answer = %q, want Use this", resp.Answer)
	}
	if resp.HasCommand() {
		t.Fatal("HasCommand() = true, want false")
	}
}

func TestParseInvalid(t *testing.T) {
	if _, err := Parse("not-json"); err == nil {
		t.Fatal("expected error for invalid response")
	}
}

func TestBuildPromptMarkdownEnabled(t *testing.T) {
	prompt := BuildPrompt("zsh", "/tmp/project", "darwin", true)
	if !strings.Contains(prompt, "use clean Markdown by default") {
		t.Fatalf("prompt missing markdown-default instruction: %q", prompt)
	}
	if !strings.Contains(prompt, "Do not use markdown code fences") {
		t.Fatalf("prompt missing markdown fence instruction: %q", prompt)
	}
}

func TestBuildPromptMarkdownDisabled(t *testing.T) {
	prompt := BuildPrompt("zsh", "/tmp/project", "darwin", false)
	if !strings.Contains(prompt, "plain text only") {
		t.Fatalf("prompt missing plain-text instruction: %q", prompt)
	}
}
