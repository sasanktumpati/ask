package providers

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

type geminiClient struct {
	apiKey  string
	base    string
	http    *http.Client
	headers map[string]string
}

func newGeminiClient(opts ClientOptions) Client {
	base := opts.BaseURL
	if strings.TrimSpace(base) == "" {
		base = "https://generativelanguage.googleapis.com/v1beta"
	}
	headers := map[string]string{}
	for k, v := range opts.Headers {
		headers[k] = v
	}
	return &geminiClient{
		apiKey:  strings.TrimSpace(opts.APIKey),
		base:    strings.TrimRight(strings.TrimSpace(base), "/"),
		http:    defaultHTTPClient(opts.HTTPClient),
		headers: headers,
	}
}

func (c *geminiClient) Name() string { return "gemini" }

func (c *geminiClient) ListModels(ctx context.Context) ([]Model, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not configured")
	}

	req, err := http.NewRequest(http.MethodGet, joinURL(c.base, "/models"), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	c.setHeaders(req)

	var resp struct {
		Models []struct {
			Name                       string   `json:"name"`
			DisplayName                string   `json:"displayName"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
	}
	if err := doJSON(ctx, c.http, req, nil, &resp); err != nil {
		return nil, err
	}

	models := make([]Model, 0, len(resp.Models))
	for _, m := range resp.Models {
		if !supportsGenerateContent(m.SupportedGenerationMethods) {
			continue
		}
		id := strings.TrimPrefix(strings.TrimSpace(m.Name), "models/")
		if id == "" {
			continue
		}
		display := strings.TrimSpace(m.DisplayName)
		if display == "" {
			display = id
		}
		models = append(models, Model{ID: id, DisplayName: display})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	return models, nil
}

func (c *geminiClient) Ask(ctx context.Context, reqBody AskRequest) (AskResponse, error) {
	if err := validateAskRequest(reqBody); err != nil {
		return AskResponse{}, err
	}
	if c.apiKey == "" {
		return AskResponse{}, fmt.Errorf("GEMINI_API_KEY not configured")
	}

	model := strings.TrimSpace(reqBody.Model)
	model = strings.TrimPrefix(model, "models/")
	if model == "" {
		return AskResponse{}, fmt.Errorf("model is required")
	}

	path := fmt.Sprintf("/models/%s:generateContent", model)
	req, err := http.NewRequest(http.MethodPost, joinURL(c.base, path), nil)
	if err != nil {
		return AskResponse{}, fmt.Errorf("build request: %w", err)
	}
	c.setHeaders(req)

	payload := map[string]any{
		"systemInstruction": map[string]any{
			"parts": []map[string]string{{"text": reqBody.Prompt}},
		},
		"contents": []map[string]any{
			{
				"role":  "user",
				"parts": []map[string]string{{"text": reqBody.Question}},
			},
		},
		"generationConfig": map[string]any{
			"temperature": 0.2,
		},
	}
	if reqBody.ExpectJSON {
		payload["generationConfig"].(map[string]any)["responseMimeType"] = "application/json"
	}

	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := doJSON(ctx, c.http, req, payload, &resp); err != nil {
		if reqBody.ExpectJSON && responseFormatLikelyUnsupported(err) {
			payloadNoFormat := map[string]any{
				"systemInstruction": payload["systemInstruction"],
				"contents":          payload["contents"],
				"generationConfig": map[string]any{
					"temperature": 0.2,
				},
			}
			retryReq, buildErr := http.NewRequest(http.MethodPost, joinURL(c.base, path), nil)
			if buildErr != nil {
				return AskResponse{}, fmt.Errorf("build retry request: %w", buildErr)
			}
			c.setHeaders(retryReq)
			if retryErr := doJSON(ctx, c.http, retryReq, payloadNoFormat, &resp); retryErr != nil {
				return AskResponse{}, retryErr
			}
		} else {
			return AskResponse{}, err
		}
	}
	if len(resp.Candidates) == 0 {
		return AskResponse{}, fmt.Errorf("no candidates returned by Gemini")
	}

	parts := make([]string, 0, len(resp.Candidates[0].Content.Parts))
	for _, part := range resp.Candidates[0].Content.Parts {
		if strings.TrimSpace(part.Text) != "" {
			parts = append(parts, part.Text)
		}
	}
	if len(parts) == 0 {
		return AskResponse{}, fmt.Errorf("Gemini response had no text parts")
	}
	return AskResponse{Text: strings.Join(parts, "\n")}, nil
}

func (c *geminiClient) setHeaders(req *http.Request) {
	req.Header.Set("x-goog-api-key", c.apiKey)
	for k, v := range c.headers {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		req.Header.Set(k, v)
	}
}

func supportsGenerateContent(methods []string) bool {
	for _, method := range methods {
		if strings.EqualFold(strings.TrimSpace(method), "generateContent") {
			return true
		}
	}
	return false
}
