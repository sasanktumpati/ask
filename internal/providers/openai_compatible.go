package providers

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

type openAICompatibleClient struct {
	name          string
	apiKey        string
	base          string
	http          *http.Client
	modelsPath    string
	chatPath      string
	authHeader    string
	authPrefix    string
	requireAPIKey bool
	headers       map[string]string
}

func newOpenAICompatibleClient(settings OpenAICompatibleSettings, opts ClientOptions) Client {
	modelsPath := settings.ModelsPath
	if strings.TrimSpace(modelsPath) == "" {
		modelsPath = "/models"
	}
	chatPath := settings.ChatPath
	if strings.TrimSpace(chatPath) == "" {
		chatPath = "/chat/completions"
	}
	authHeader := settings.AuthHeader
	if strings.TrimSpace(authHeader) == "" {
		authHeader = "Authorization"
	}
	authPrefix := settings.AuthPrefix
	if authPrefix == "" {
		authPrefix = "Bearer "
	}

	headers := map[string]string{}
	for k, v := range opts.Headers {
		headers[k] = v
	}

	return &openAICompatibleClient{
		name:          normalize(settings.Name),
		apiKey:        strings.TrimSpace(opts.APIKey),
		base:          strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/"),
		http:          defaultHTTPClient(opts.HTTPClient),
		modelsPath:    ensureLeadingSlash(modelsPath),
		chatPath:      ensureLeadingSlash(chatPath),
		authHeader:    authHeader,
		authPrefix:    authPrefix,
		requireAPIKey: settings.RequireAPIKey,
		headers:       headers,
	}
}

func (c *openAICompatibleClient) Name() string {
	return c.name
}

func (c *openAICompatibleClient) ListModels(ctx context.Context) ([]Model, error) {
	if c.requiresAPIKey() && c.apiKey == "" {
		return nil, fmt.Errorf("API key not configured for %s", c.name)
	}
	url := joinURL(c.base, c.modelsPath)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	c.setHeaders(req)

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := doJSON(ctx, c.http, req, nil, &resp); err != nil {
		return nil, err
	}

	models := make([]Model, 0, len(resp.Data))
	for _, m := range resp.Data {
		id := strings.TrimSpace(m.ID)
		if id == "" {
			continue
		}
		models = append(models, Model{ID: id, DisplayName: id})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	return models, nil
}

func (c *openAICompatibleClient) Ask(ctx context.Context, reqBody AskRequest) (AskResponse, error) {
	if err := validateAskRequest(reqBody); err != nil {
		return AskResponse{}, err
	}
	if c.requiresAPIKey() && c.apiKey == "" {
		return AskResponse{}, fmt.Errorf("API key not configured for %s", c.name)
	}
	url := joinURL(c.base, c.chatPath)
	resp, err := c.askWithPayload(ctx, url, reqBody, true)
	if err != nil && reqBody.ExpectJSON && responseFormatLikelyUnsupported(err) {
		resp, err = c.askWithPayload(ctx, url, reqBody, false)
	}
	if err != nil {
		return AskResponse{}, err
	}
	if len(resp.Choices) == 0 {
		return AskResponse{}, fmt.Errorf("no choices returned by %s", c.name)
	}

	text, err := extractMessageContent(resp.Choices[0].Message.Content)
	if err != nil {
		return AskResponse{}, fmt.Errorf("decode %s response content: %w", c.name, err)
	}
	return AskResponse{Text: text}, nil
}

func (c *openAICompatibleClient) askWithPayload(ctx context.Context, url string, reqBody AskRequest, includeResponseFormat bool) (struct {
	Choices []struct {
		Message struct {
			Content any `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}, error) {
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return struct {
			Choices []struct {
				Message struct {
					Content any `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}{}, fmt.Errorf("build request: %w", err)
	}
	c.setHeaders(req)

	payload := map[string]any{
		"model": reqBody.Model,
		"messages": []map[string]string{
			{"role": "system", "content": reqBody.Prompt},
			{"role": "user", "content": reqBody.Question},
		},
		"temperature": 0.2,
	}
	if reqBody.ExpectJSON && includeResponseFormat {
		payload["response_format"] = map[string]string{"type": "json_object"}
	}

	var resp struct {
		Choices []struct {
			Message struct {
				Content any `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := doJSON(ctx, c.http, req, payload, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

func (c *openAICompatibleClient) setHeaders(req *http.Request) {
	if c.requiresAPIKey() && c.apiKey != "" {
		req.Header.Set(c.authHeader, c.authPrefix+c.apiKey)
	}
	for k, v := range c.headers {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		req.Header.Set(k, v)
	}
}

func (c *openAICompatibleClient) requiresAPIKey() bool {
	return c.requireAPIKey
}

func extractMessageContent(content any) (string, error) {
	switch value := content.(type) {
	case string:
		return strings.TrimSpace(value), nil
	case []any:
		parts := make([]string, 0, len(value))
		for _, item := range value {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			text, _ := obj["text"].(string)
			if strings.TrimSpace(text) != "" {
				parts = append(parts, text)
			}
		}
		if len(parts) == 0 {
			return "", fmt.Errorf("array content had no text parts")
		}
		return strings.TrimSpace(strings.Join(parts, "\n")), nil
	default:
		return "", fmt.Errorf("unsupported content type %T", value)
	}
}
