package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultDirName          = "ask"
	defaultFileName         = "config.json"
	defaultTemplateFileName = "config.template.json"
	defaultOpenAIModel      = "gpt-5-nano"
	currentVersion          = 1
	envConfigPath           = "ASK_CONFIG"
	envConfigDir            = "ASK_CONFIG_DIR"
)

var (
	// ErrConfigNotFound indicates the config file does not exist yet.
	ErrConfigNotFound = errors.New("config file not found")
)

// ProviderConfig stores per-provider defaults and credentials.
type ProviderConfig struct {
	APIKey    string `json:"api_key"`
	Model     string `json:"model"`
	BaseURL   string `json:"base_url,omitempty"`
	APIKeyEnv string `json:"api_key_env,omitempty"`
}

// OpenAICompatibleProvider defines a custom OpenAI-compatible provider.
type OpenAICompatibleProvider struct {
	BaseURL    string            `json:"base_url"`
	APIKey     string            `json:"api_key"`
	Model      string            `json:"model"`
	APIKeyEnv  string            `json:"api_key_env,omitempty"`
	ModelsPath string            `json:"models_path,omitempty"`
	ChatPath   string            `json:"chat_path,omitempty"`
	AuthHeader string            `json:"auth_header,omitempty"`
	AuthPrefix string            `json:"auth_prefix,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// Config is the persisted ask CLI configuration.
type Config struct {
	Version         int                                 `json:"version"`
	CurrentProvider string                              `json:"current_provider"`
	CurrentModels   map[string]string                   `json:"current_models,omitempty"` // legacy read-only compatibility
	Providers       map[string]ProviderConfig           `json:"providers,omitempty"`
	CustomProviders map[string]OpenAICompatibleProvider `json:"custom_providers,omitempty"`
	OllamaHost      string                              `json:"ollama_host,omitempty"`
	RenderMarkdown  bool                                `json:"render_markdown"`
}

// BuiltinDefaults defines immutable defaults for built-in providers.
type BuiltinDefaults struct {
	BaseURL   string
	APIKeyEnv string
}

var builtinProviders = map[string]BuiltinDefaults{
	"anthropic": {
		BaseURL:   "https://api.anthropic.com",
		APIKeyEnv: "ANTHROPIC_API_KEY",
	},
	"gemini": {
		BaseURL:   "https://generativelanguage.googleapis.com/v1beta",
		APIKeyEnv: "GEMINI_API_KEY",
	},
	"ollama": {
		BaseURL:   "http://127.0.0.1:11434",
		APIKeyEnv: "",
	},
	"openai": {
		BaseURL:   "https://api.openai.com/v1",
		APIKeyEnv: "OPENAI_API_KEY",
	},
	"openrouter": {
		BaseURL:   "https://openrouter.ai/api/v1",
		APIKeyEnv: "OPENROUTER_API_KEY",
	},
}

// ResolvePath resolves config file path from CLI override, environment, or default.
func ResolvePath(pathOverride string) (string, error) {
	if path := strings.TrimSpace(pathOverride); path != "" {
		return filepath.Clean(path), nil
	}
	if path := strings.TrimSpace(os.Getenv(envConfigPath)); path != "" {
		return filepath.Clean(path), nil
	}
	return DefaultPath()
}

// DefaultDir returns the default directory where ask stores its config.
func DefaultDir() (string, error) {
	if custom := strings.TrimSpace(os.Getenv(envConfigDir)); custom != "" {
		return filepath.Clean(custom), nil
	}

	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}
	return filepath.Join(home, "."+defaultDirName), nil
}

// DefaultPath returns the default full path to config.json.
func DefaultPath() (string, error) {
	dir, err := DefaultDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, defaultFileName), nil
}

// TemplatePathForConfig returns the template path for a given config path.
func TemplatePathForConfig(configPath string) string {
	path := strings.TrimSpace(configPath)
	if path == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(path), defaultTemplateFileName)
}

// DefaultTemplatePath returns the default full path to config.template.json.
func DefaultTemplatePath() (string, error) {
	path, err := DefaultPath()
	if err != nil {
		return "", err
	}
	return TemplatePathForConfig(path), nil
}

// BuiltinProviderNames returns built-in provider names sorted alphabetically.
func BuiltinProviderNames() []string {
	names := make([]string, 0, len(builtinProviders))
	for name := range builtinProviders {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// IsBuiltinProvider reports whether name is a built-in provider.
func IsBuiltinProvider(name string) bool {
	_, ok := builtinProviders[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

// BuiltinProviderDefaults returns defaults for a built-in provider.
func BuiltinProviderDefaults(name string) (BuiltinDefaults, bool) {
	defaults, ok := builtinProviders[strings.ToLower(strings.TrimSpace(name))]
	return defaults, ok
}

// Load reads config from path. When missing, it returns DefaultConfig and ErrConfigNotFound.
func Load(path string) (*Config, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultConfig(), ErrConfigNotFound
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(buf, cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	cfg.normalize()
	return cfg, nil
}

// Save persists config to path using normalized and compact representation.
func Save(path string, cfg *Config) error {
	cfg.normalize()
	return writeSecureJSON(path, cfg.compactForSave())
}

// DefaultConfig returns a new default configuration.
func DefaultConfig() *Config {
	cfg := &Config{
		Version:         currentVersion,
		CurrentProvider: "",
		CurrentModels:   nil,
		Providers:       builtinProviderScaffold(),
		CustomProviders: map[string]OpenAICompatibleProvider{},
		RenderMarkdown:  true,
	}
	cfg.normalize()
	return cfg
}

// TemplateConfig returns the starter template configuration.
func TemplateConfig() *Config {
	cfg := DefaultConfig()
	cfg.CustomProviders = map[string]OpenAICompatibleProvider{
		"myproxy": {
			BaseURL:   "https://llm.example.com/v1",
			APIKeyEnv: "MYPROXY_API_KEY",
			Headers: map[string]string{
				"X-Client-Name": "ask",
			},
		},
	}
	return cfg
}

// EnsureTemplate creates template config file if it does not already exist.
func EnsureTemplate(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("template path is empty")
	}
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat template: %w", err)
	}
	return writeSecureJSON(path, TemplateConfig())
}

func (c *Config) normalize() {
	if c.Version == 0 {
		c.Version = currentVersion
	}
	if c.Providers == nil {
		c.Providers = map[string]ProviderConfig{}
	}
	if c.CustomProviders == nil {
		c.CustomProviders = map[string]OpenAICompatibleProvider{}
	}
	for provider, model := range c.CurrentModels {
		provider = strings.ToLower(strings.TrimSpace(provider))
		model = strings.TrimSpace(model)
		if provider == "" || model == "" {
			continue
		}
		if custom, ok := c.CustomProviders[provider]; ok {
			if strings.TrimSpace(custom.Model) == "" {
				custom.Model = model
				c.CustomProviders[provider] = custom
			}
			continue
		}
		pc := c.Providers[provider]
		if strings.TrimSpace(pc.Model) == "" {
			pc.Model = model
			c.Providers[provider] = pc
		}
	}
	c.CurrentModels = nil
	c.OllamaHost = strings.TrimRight(strings.TrimSpace(c.OllamaHost), "/")
	c.CurrentProvider = strings.ToLower(strings.TrimSpace(c.CurrentProvider))
}

// GetModel returns the configured default model for provider.
func (c *Config) GetModel(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return ""
	}
	if custom, ok := c.CustomProviders[provider]; ok {
		return strings.TrimSpace(custom.Model)
	}
	return strings.TrimSpace(c.Providers[provider].Model)
}

// SetModel sets the default model for provider.
func (c *Config) SetModel(provider string, model string) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	c.normalize()
	model = strings.TrimSpace(model)
	if custom, ok := c.CustomProviders[provider]; ok {
		custom.Model = model
		c.CustomProviders[provider] = custom
		return
	}
	pc := c.Providers[provider]
	pc.Model = model
	c.Providers[provider] = pc
}

// ProviderExists reports whether provider is configured or built in.
func (c *Config) ProviderExists(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if IsBuiltinProvider(name) {
		return true
	}
	if c.CustomProviders == nil {
		return false
	}
	_, ok := c.CustomProviders[name]
	return ok
}

// ResolveBaseURL returns effective base URL for provider.
func (c *Config) ResolveBaseURL(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return ""
	}

	if custom, ok := c.CustomProviders[provider]; ok {
		if strings.TrimSpace(custom.BaseURL) != "" {
			return strings.TrimRight(custom.BaseURL, "/")
		}
		return ""
	}

	defaults, builtin := BuiltinProviderDefaults(provider)
	pc := c.Providers[provider]
	if provider == "ollama" {
		if strings.TrimSpace(c.OllamaHost) != "" {
			return strings.TrimRight(strings.TrimSpace(c.OllamaHost), "/")
		}
	}
	if strings.TrimSpace(pc.BaseURL) != "" {
		return strings.TrimRight(pc.BaseURL, "/")
	}
	if builtin {
		return strings.TrimRight(defaults.BaseURL, "/")
	}
	return ""
}

func (c *Config) compactForSave() *Config {
	compacted := *c

	compacted.CurrentModels = nil

	compacted.Providers = nil
	if len(c.Providers) > 0 {
		providers := map[string]ProviderConfig{}
		for provider, raw := range c.Providers {
			provider = strings.ToLower(strings.TrimSpace(provider))
			if provider == "" {
				continue
			}
			normalized := ProviderConfig{
				APIKey:    strings.TrimSpace(raw.APIKey),
				Model:     strings.TrimSpace(raw.Model),
				BaseURL:   strings.TrimRight(strings.TrimSpace(raw.BaseURL), "/"),
				APIKeyEnv: strings.TrimSpace(raw.APIKeyEnv),
			}
			if normalized.APIKey == "" && normalized.Model == "" && normalized.BaseURL == "" && normalized.APIKeyEnv == "" {
				continue
			}
			providers[provider] = normalized
		}
		if len(providers) > 0 {
			compacted.Providers = providers
		}
	}

	compacted.CustomProviders = nil
	if len(c.CustomProviders) > 0 {
		customProviders := map[string]OpenAICompatibleProvider{}
		for name, raw := range c.CustomProviders {
			name = strings.ToLower(strings.TrimSpace(name))
			if name == "" {
				continue
			}
			normalized := OpenAICompatibleProvider{
				BaseURL:    strings.TrimRight(strings.TrimSpace(raw.BaseURL), "/"),
				APIKey:     strings.TrimSpace(raw.APIKey),
				Model:      strings.TrimSpace(raw.Model),
				APIKeyEnv:  strings.TrimSpace(raw.APIKeyEnv),
				ModelsPath: strings.TrimSpace(raw.ModelsPath),
				ChatPath:   strings.TrimSpace(raw.ChatPath),
				AuthHeader: strings.TrimSpace(raw.AuthHeader),
				AuthPrefix: raw.AuthPrefix,
			}
			if normalized.BaseURL == "" {
				continue
			}
			if normalized.ModelsPath == "/models" {
				normalized.ModelsPath = ""
			}
			if normalized.ChatPath == "/chat/completions" {
				normalized.ChatPath = ""
			}
			if normalized.AuthHeader == "Authorization" {
				normalized.AuthHeader = ""
			}
			if normalized.AuthPrefix == "Bearer " {
				normalized.AuthPrefix = ""
			}

			headers := map[string]string{}
			for key, value := range raw.Headers {
				key = strings.TrimSpace(key)
				value = strings.TrimSpace(value)
				if key == "" || value == "" {
					continue
				}
				headers[key] = value
			}
			if len(headers) > 0 {
				normalized.Headers = headers
			}

			customProviders[name] = normalized
		}
		if len(customProviders) > 0 {
			compacted.CustomProviders = customProviders
		}
	}

	compacted.OllamaHost = strings.TrimRight(strings.TrimSpace(c.OllamaHost), "/")
	if compacted.OllamaHost == strings.TrimRight(builtinProviders["ollama"].BaseURL, "/") {
		compacted.OllamaHost = ""
	}

	return &compacted
}

func builtinProviderScaffold() map[string]ProviderConfig {
	providers := map[string]ProviderConfig{}
	for _, name := range BuiltinProviderNames() {
		defaults, _ := BuiltinProviderDefaults(name)
		cfg := ProviderConfig{}
		if name == "openai" {
			cfg.Model = defaultOpenAIModel
		}
		if strings.TrimSpace(defaults.APIKeyEnv) != "" {
			cfg.APIKeyEnv = strings.TrimSpace(defaults.APIKeyEnv)
		}
		if name == "ollama" {
			cfg.BaseURL = strings.TrimRight(strings.TrimSpace(defaults.BaseURL), "/")
		}
		providers[name] = cfg
	}
	return providers
}

// ResolveAPIKey returns effective API key, preferring configured env vars over stored key.
func (c *Config) ResolveAPIKey(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return ""
	}

	if custom, ok := c.CustomProviders[provider]; ok {
		if env := strings.TrimSpace(custom.APIKeyEnv); env != "" {
			if v := strings.TrimSpace(os.Getenv(env)); v != "" {
				return v
			}
		}
		if v := strings.TrimSpace(custom.APIKey); v != "" {
			return v
		}
		return ""
	}

	pc := c.Providers[provider]
	if env := strings.TrimSpace(pc.APIKeyEnv); env != "" {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			return v
		}
	}
	if defaults, ok := BuiltinProviderDefaults(provider); ok {
		if env := strings.TrimSpace(defaults.APIKeyEnv); env != "" {
			if v := strings.TrimSpace(os.Getenv(env)); v != "" {
				return v
			}
		}
	}
	if v := strings.TrimSpace(pc.APIKey); v != "" {
		return v
	}
	return ""
}

// SetAPIKey sets a provider API key in config.
func (c *Config) SetAPIKey(provider string, key string) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	c.normalize()
	if custom, ok := c.CustomProviders[provider]; ok {
		custom.APIKey = strings.TrimSpace(key)
		c.CustomProviders[provider] = custom
		return
	}
	pc := c.Providers[provider]
	pc.APIKey = strings.TrimSpace(key)
	c.Providers[provider] = pc
}

// SetAPIKeyEnv sets a provider API key environment variable name.
func (c *Config) SetAPIKeyEnv(provider, envVar string) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	envVar = strings.TrimSpace(envVar)
	c.normalize()
	if custom, ok := c.CustomProviders[provider]; ok {
		custom.APIKeyEnv = envVar
		c.CustomProviders[provider] = custom
		return
	}
	pc := c.Providers[provider]
	pc.APIKeyEnv = envVar
	c.Providers[provider] = pc
}

// SetBaseURL sets provider base URL.
func (c *Config) SetBaseURL(provider, baseURL string) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	c.normalize()
	if provider == "ollama" {
		c.OllamaHost = baseURL
	}
	pc := c.Providers[provider]
	pc.BaseURL = baseURL
	c.Providers[provider] = pc
}

// SetCurrentProvider sets the default provider for ask calls.
func (c *Config) SetCurrentProvider(provider string) {
	c.normalize()
	c.CurrentProvider = strings.ToLower(strings.TrimSpace(provider))
}

// ProviderNames returns all provider names (built-in and custom), sorted.
func (c *Config) ProviderNames() []string {
	names := BuiltinProviderNames()
	for name := range c.CustomProviders {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// AddCustomProvider adds or updates a custom OpenAI-compatible provider.
func (c *Config) AddCustomProvider(name string, input OpenAICompatibleProvider) error {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return fmt.Errorf("provider name is required")
	}
	if IsBuiltinProvider(name) {
		return fmt.Errorf("%q is a built-in provider", name)
	}
	if strings.TrimSpace(input.BaseURL) == "" {
		return fmt.Errorf("base_url is required")
	}

	input.BaseURL = strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	if strings.TrimSpace(input.ModelsPath) == "" {
		input.ModelsPath = "/models"
	}
	if strings.TrimSpace(input.ChatPath) == "" {
		input.ChatPath = "/chat/completions"
	}
	if strings.TrimSpace(input.AuthHeader) == "" {
		input.AuthHeader = "Authorization"
	}
	if input.AuthPrefix == "" {
		input.AuthPrefix = "Bearer "
	}
	if input.Headers == nil {
		input.Headers = map[string]string{}
	}

	c.normalize()
	c.CustomProviders[name] = input
	return nil
}

// RemoveCustomProvider removes a custom provider from config.
func (c *Config) RemoveCustomProvider(name string) error {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return fmt.Errorf("provider name is required")
	}
	if IsBuiltinProvider(name) {
		return fmt.Errorf("cannot remove built-in provider")
	}
	if _, ok := c.CustomProviders[name]; !ok {
		return fmt.Errorf("provider %q not found", name)
	}
	delete(c.CustomProviders, name)
	if c.CurrentProvider == name {
		c.CurrentProvider = ""
	}
	return nil
}

func writeSecureJSON(path string, payload any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("set config directory permissions: %w", err)
	}

	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, encoded, 0o600); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("set config file permissions: %w", err)
	}
	return nil
}
