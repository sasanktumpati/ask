package providers

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// Model describes a model option exposed by a provider.
type Model struct {
	ID          string
	DisplayName string
}

// AskRequest is the normalized prompt payload sent to a provider.
type AskRequest struct {
	Model      string
	Prompt     string
	Question   string
	ExpectJSON bool
}

// AskResponse is the normalized text response returned by a provider.
type AskResponse struct {
	Text string
}

// Client is the provider client interface used by the CLI.
type Client interface {
	Name() string
	ListModels(ctx context.Context) ([]Model, error)
	Ask(ctx context.Context, req AskRequest) (AskResponse, error)
}

// ClientOptions configures shared client settings for all providers.
type ClientOptions struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	Headers    map[string]string
}

// OpenAICompatibleSettings customizes behavior for OpenAI-compatible APIs.
type OpenAICompatibleSettings struct {
	Name          string
	ModelsPath    string
	ChatPath      string
	AuthHeader    string
	AuthPrefix    string
	RequireAPIKey bool
}

// New returns a built-in provider client by name.
func New(name string, opts ClientOptions) (Client, error) {
	name = normalize(name)
	if name == "" {
		return nil, fmt.Errorf("provider name is required")
	}

	switch name {
	case "openai":
		return newOpenAIClient(opts), nil
	case "anthropic":
		return newAnthropicClient(opts), nil
	case "gemini":
		return newGeminiClient(opts), nil
	case "ollama":
		return newOllamaClient(opts), nil
	case "openrouter":
		return newOpenRouterClient(opts), nil
	default:
		return nil, fmt.Errorf("unsupported provider %q", name)
	}
}

// NewOpenAICompatible returns a client for a custom OpenAI-compatible provider.
func NewOpenAICompatible(settings OpenAICompatibleSettings, opts ClientOptions) (Client, error) {
	settings.Name = normalize(settings.Name)
	if settings.Name == "" {
		return nil, fmt.Errorf("provider name is required")
	}
	if strings.TrimSpace(opts.BaseURL) == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	return newOpenAICompatibleClient(settings, opts), nil
}

// SupportedProviders returns the built-in provider names.
func SupportedProviders() []string {
	providers := []string{"anthropic", "gemini", "ollama", "openai", "openrouter"}
	sort.Strings(providers)
	return providers
}

func normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
