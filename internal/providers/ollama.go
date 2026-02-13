package providers

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

type ollamaClient struct {
	base string
	http *http.Client
}

func newOllamaClient(opts ClientOptions) Client {
	base := opts.BaseURL
	if strings.TrimSpace(base) == "" {
		base = "http://127.0.0.1:11434"
	}
	return &ollamaClient{
		base: strings.TrimRight(strings.TrimSpace(base), "/"),
		http: defaultHTTPClient(opts.HTTPClient),
	}
}

func (c *ollamaClient) Name() string { return "ollama" }

func (c *ollamaClient) ListModels(ctx context.Context) ([]Model, error) {
	req, err := http.NewRequest(http.MethodGet, joinURL(c.base, "/api/tags"), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	var resp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := doJSON(ctx, c.http, req, nil, &resp); err != nil {
		return nil, err
	}

	models := make([]Model, 0, len(resp.Models))
	for _, m := range resp.Models {
		id := strings.TrimSpace(m.Name)
		if id == "" {
			continue
		}
		models = append(models, Model{ID: id, DisplayName: id})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	return models, nil
}

func (c *ollamaClient) Ask(ctx context.Context, reqBody AskRequest) (AskResponse, error) {
	if err := validateAskRequest(reqBody); err != nil {
		return AskResponse{}, err
	}
	req, err := http.NewRequest(http.MethodPost, joinURL(c.base, "/api/chat"), nil)
	if err != nil {
		return AskResponse{}, fmt.Errorf("build request: %w", err)
	}

	payload := map[string]any{
		"model": reqBody.Model,
		"messages": []map[string]string{
			{"role": "system", "content": reqBody.Prompt},
			{"role": "user", "content": reqBody.Question},
		},
		"stream": false,
	}
	if reqBody.ExpectJSON {
		payload["format"] = "json"
	}

	var resp struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := doJSON(ctx, c.http, req, payload, &resp); err != nil {
		return AskResponse{}, err
	}
	if strings.TrimSpace(resp.Message.Content) == "" {
		return AskResponse{}, fmt.Errorf("ollama response had empty content")
	}
	return AskResponse{Text: resp.Message.Content}, nil
}
