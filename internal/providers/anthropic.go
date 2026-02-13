package providers

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

type anthropicClient struct {
	apiKey  string
	base    string
	http    *http.Client
	headers map[string]string
}

func newAnthropicClient(opts ClientOptions) Client {
	base := opts.BaseURL
	if strings.TrimSpace(base) == "" {
		base = "https://api.anthropic.com"
	}
	headers := map[string]string{}
	for k, v := range opts.Headers {
		headers[k] = v
	}
	return &anthropicClient{
		apiKey:  strings.TrimSpace(opts.APIKey),
		base:    strings.TrimRight(strings.TrimSpace(base), "/"),
		http:    defaultHTTPClient(opts.HTTPClient),
		headers: headers,
	}
}

func (c *anthropicClient) Name() string { return "anthropic" }

func (c *anthropicClient) ListModels(ctx context.Context) ([]Model, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not configured")
	}
	req, err := http.NewRequest(http.MethodGet, joinURL(c.base, "/v1/models"), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	c.setHeaders(req)

	var resp struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
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
		name := strings.TrimSpace(m.DisplayName)
		if name == "" {
			name = id
		}
		models = append(models, Model{ID: id, DisplayName: name})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	return models, nil
}

func (c *anthropicClient) Ask(ctx context.Context, reqBody AskRequest) (AskResponse, error) {
	if err := validateAskRequest(reqBody); err != nil {
		return AskResponse{}, err
	}
	if c.apiKey == "" {
		return AskResponse{}, fmt.Errorf("ANTHROPIC_API_KEY not configured")
	}
	req, err := http.NewRequest(http.MethodPost, joinURL(c.base, "/v1/messages"), nil)
	if err != nil {
		return AskResponse{}, fmt.Errorf("build request: %w", err)
	}
	c.setHeaders(req)

	payload := map[string]any{
		"model":      reqBody.Model,
		"max_tokens": 2048,
		"system":     reqBody.Prompt,
		"messages": []map[string]any{
			{
				"role":    "user",
				"content": []map[string]string{{"type": "text", "text": reqBody.Question}},
			},
		},
	}

	var resp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := doJSON(ctx, c.http, req, payload, &resp); err != nil {
		return AskResponse{}, err
	}

	parts := make([]string, 0, len(resp.Content))
	for _, block := range resp.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			parts = append(parts, block.Text)
		}
	}
	if len(parts) == 0 {
		return AskResponse{}, fmt.Errorf("no text content returned by Anthropic")
	}
	return AskResponse{Text: strings.Join(parts, "\n")}, nil
}

func (c *anthropicClient) setHeaders(req *http.Request) {
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	for k, v := range c.headers {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		req.Header.Set(k, v)
	}
}
