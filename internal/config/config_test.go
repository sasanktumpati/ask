package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDefaultPathUsesAskDirectory(t *testing.T) {
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath() error = %v", err)
	}
	suffix := filepath.Join(".ask", "config.json")
	if !strings.HasSuffix(path, suffix) {
		t.Fatalf("path %q does not end with %q", path, suffix)
	}
}

func TestDefaultTemplatePathUsesAskDirectory(t *testing.T) {
	path, err := DefaultTemplatePath()
	if err != nil {
		t.Fatalf("DefaultTemplatePath() error = %v", err)
	}
	suffix := filepath.Join(".ask", "config.template.json")
	if !strings.HasSuffix(path, suffix) {
		t.Fatalf("path %q does not end with %q", path, suffix)
	}
}

func TestAddCustomProviderDefaults(t *testing.T) {
	cfg := DefaultConfig()
	err := cfg.AddCustomProvider("myproxy", OpenAICompatibleProvider{BaseURL: "https://llm.example.com/v1"})
	if err != nil {
		t.Fatalf("AddCustomProvider() error = %v", err)
	}
	p := cfg.CustomProviders["myproxy"]
	if p.ModelsPath != "/models" || p.ChatPath != "/chat/completions" {
		t.Fatalf("unexpected defaults: %+v", p)
	}
	if p.AuthHeader != "Authorization" || p.AuthPrefix != "Bearer " {
		t.Fatalf("unexpected auth defaults: %+v", p)
	}
}

func TestResolveAPIKeyPrecedence_CustomProvider(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.AddCustomProvider("proxy", OpenAICompatibleProvider{
		BaseURL:   "https://llm.example.com/v1",
		APIKey:    "from-config",
		APIKeyEnv: "PROXY_API_KEY",
	}); err != nil {
		t.Fatalf("AddCustomProvider() error = %v", err)
	}

	t.Setenv("PROXY_API_KEY", "from-env")
	if got := cfg.ResolveAPIKey("proxy"); got != "from-env" {
		t.Fatalf("ResolveAPIKey() = %q, want from-env", got)
	}
}

func TestSetAPIKeyAffectsCustomProvider(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.AddCustomProvider("proxy", OpenAICompatibleProvider{BaseURL: "https://llm.example.com/v1"}); err != nil {
		t.Fatalf("AddCustomProvider() error = %v", err)
	}
	cfg.SetAPIKey("proxy", "abc123")
	if got := cfg.CustomProviders["proxy"].APIKey; got != "abc123" {
		t.Fatalf("custom api key = %q, want abc123", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetCurrentProvider("ollama")
	cfg.SetModel("ollama", "llama3.2")
	cfg.SetAPIKey("openai", "sk-test")
	cfg.RenderMarkdown = false

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.CurrentProvider != "ollama" || loaded.GetModel("ollama") != "llama3.2" {
		t.Fatalf("unexpected loaded config: %+v", loaded)
	}
	if loaded.Providers["openai"].APIKey != "sk-test" {
		t.Fatalf("api key mismatch after load")
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat config: %v", err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("mode = %o, want 600", info.Mode().Perm())
		}
	}
}

func TestEnsureTemplateCreatesTemplateOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.template.json")

	if err := EnsureTemplate(path); err != nil {
		t.Fatalf("EnsureTemplate() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("template was not created: %v", err)
	}

	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read template: %v", err)
	}

	if err := EnsureTemplate(path); err != nil {
		t.Fatalf("EnsureTemplate() second call error = %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read template after second call: %v", err)
	}
	if string(original) != string(after) {
		t.Fatalf("template content changed on second EnsureTemplate call")
	}
}

func TestTemplateConfigIncludesBuiltinProviders(t *testing.T) {
	cfg := TemplateConfig()
	for _, name := range BuiltinProviderNames() {
		if _, ok := cfg.Providers[name]; !ok {
			t.Fatalf("expected builtin provider %q in template providers", name)
		}
	}
	if _, ok := cfg.CustomProviders["myproxy"]; !ok {
		t.Fatalf("expected myproxy example in custom providers template")
	}
	if cfg.Providers["openai"].Model == "" {
		t.Fatalf("expected openai model in template providers")
	}
}

func TestResolvePath_EnvOverride(t *testing.T) {
	t.Setenv("ASK_CONFIG", "/tmp/custom-ask.json")
	path, err := ResolvePath("")
	if err != nil {
		t.Fatalf("ResolvePath() error = %v", err)
	}
	if path != filepath.Clean("/tmp/custom-ask.json") {
		t.Fatalf("path = %q", path)
	}
}

func TestResolvePath_ExplicitOverrideWins(t *testing.T) {
	t.Setenv("ASK_CONFIG", "/tmp/from-env.json")
	path, err := ResolvePath("/tmp/from-arg.json")
	if err != nil {
		t.Fatalf("ResolvePath() error = %v", err)
	}
	if path != filepath.Clean("/tmp/from-arg.json") {
		t.Fatalf("path = %q", path)
	}
}

func TestTemplatePathForConfig(t *testing.T) {
	got := TemplatePathForConfig("/tmp/ask/config.json")
	want := filepath.Join("/tmp/ask", "config.template.json")
	if got != want {
		t.Fatalf("TemplatePathForConfig() = %q, want %q", got, want)
	}
}

func TestDefaultConfigUsesBuiltinProviderDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.CurrentProvider != "" {
		t.Fatalf("CurrentProvider = %q, want empty", cfg.CurrentProvider)
	}
	if got := cfg.GetModel("openai"); got == "" {
		t.Fatalf("expected default openai model in providers")
	}
	for _, name := range BuiltinProviderNames() {
		def, _ := BuiltinProviderDefaults(name)
		if got := cfg.ResolveBaseURL(name); got == "" {
			t.Fatalf("provider %q resolved empty base_url", name)
		}
		if def.APIKeyEnv != "" {
			cfg.SetAPIKeyEnv(name, "")
			t.Setenv(def.APIKeyEnv, "from-env")
			if got := cfg.ResolveAPIKey(name); got != "from-env" {
				t.Fatalf("provider %q ResolveAPIKey() = %q, want from-env", name, got)
			}
		}
	}
}

func TestResolveAPIKey_EnvOverridesPlainKeyWithoutMutatingConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Providers["openai"] = ProviderConfig{
		APIKey:    "sk-config",
		APIKeyEnv: "OPENAI_API_KEY",
	}
	t.Setenv("OPENAI_API_KEY", "sk-env")

	got := cfg.ResolveAPIKey("openai")
	if got != "sk-env" {
		t.Fatalf("ResolveAPIKey() = %q, want sk-env", got)
	}

	if cfg.Providers["openai"].APIKey != "sk-config" {
		t.Fatalf("api_key was mutated to %q", cfg.Providers["openai"].APIKey)
	}
}

func TestSaveKeepsProviderScaffoldInDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	content := string(buf)
	for _, name := range BuiltinProviderNames() {
		if !strings.Contains(content, `"`+name+`"`) {
			t.Fatalf("expected default config to include provider %q, got: %s", name, content)
		}
	}
	if strings.Contains(content, "\"current_models\"") {
		t.Fatalf("expected default config to omit legacy current_models, got: %s", content)
	}
	if !strings.Contains(content, "\"model\": \"gpt-5-nano\"") {
		t.Fatalf("expected default config to include default provider model, got: %s", content)
	}
}
