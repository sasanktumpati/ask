package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSupportedProviders(t *testing.T) {
	got := SupportedProviders()
	want := []string{"anthropic", "gemini", "ollama", "openai", "openrouter"}
	if len(got) != len(want) {
		t.Fatalf("SupportedProviders len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SupportedProviders[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestOpenAIEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			if got := r.Header.Get("Authorization"); got != "Bearer sk-openai" {
				t.Fatalf("Authorization header = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": "gpt-4o-mini"}},
			})
		case "/v1/chat/completions":
			if got := r.Header.Get("Authorization"); got != "Bearer sk-openai" {
				t.Fatalf("Authorization header = %q", got)
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			if payload["model"] != "gpt-4o-mini" {
				t.Fatalf("payload.model = %v", payload["model"])
			}
			responseFormat, _ := payload["response_format"].(map[string]any)
			if responseFormat["type"] != "json_object" {
				t.Fatalf("response_format.type = %v", responseFormat["type"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{{
					"message": map[string]any{"content": "{\"answer\":\"ok\",\"command\":\"\"}"},
				}},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := New("openai", ClientOptions{
		APIKey:  "sk-openai",
		BaseURL: server.URL + "/v1",
	})
	if err != nil {
		t.Fatalf("New(openai) error = %v", err)
	}

	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels error = %v", err)
	}
	if len(models) != 1 || models[0].ID != "gpt-4o-mini" {
		t.Fatalf("unexpected models: %+v", models)
	}

	resp, err := client.Ask(context.Background(), AskRequest{
		Model:      "gpt-4o-mini",
		Prompt:     "sys",
		Question:   "q",
		ExpectJSON: true,
	})
	if err != nil {
		t.Fatalf("Ask error = %v", err)
	}
	if resp.Text == "" {
		t.Fatal("expected non-empty response text")
	}
}

func TestOpenRouterEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/models":
			if got := r.Header.Get("Authorization"); got != "Bearer sk-or" {
				t.Fatalf("Authorization header = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": "openrouter/model"}},
			})
		case "/api/v1/chat/completions":
			if got := r.Header.Get("Authorization"); got != "Bearer sk-or" {
				t.Fatalf("Authorization header = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{{
					"message": map[string]any{"content": "{\"answer\":\"ok\",\"command\":\"\"}"},
				}},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := New("openrouter", ClientOptions{
		APIKey:  "sk-or",
		BaseURL: server.URL + "/api/v1",
	})
	if err != nil {
		t.Fatalf("New(openrouter) error = %v", err)
	}

	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels error = %v", err)
	}
	if len(models) != 1 || models[0].ID != "openrouter/model" {
		t.Fatalf("unexpected models: %+v", models)
	}

	if _, err := client.Ask(context.Background(), AskRequest{
		Model:      "openrouter/model",
		Prompt:     "sys",
		Question:   "q",
		ExpectJSON: true,
	}); err != nil {
		t.Fatalf("Ask error = %v", err)
	}
}

func TestAnthropicEndpointsAndHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			if got := r.Header.Get("x-api-key"); got != "ak-test" {
				t.Fatalf("x-api-key = %q", got)
			}
			if got := r.Header.Get("anthropic-version"); got != "2023-06-01" {
				t.Fatalf("anthropic-version = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{
					"id":           "claude-3-5-sonnet-latest",
					"display_name": "Claude Sonnet",
				}},
			})
		case "/v1/messages":
			if got := r.Header.Get("x-api-key"); got != "ak-test" {
				t.Fatalf("x-api-key = %q", got)
			}
			if got := r.Header.Get("anthropic-version"); got != "2023-06-01" {
				t.Fatalf("anthropic-version = %q", got)
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			if payload["model"] != "claude-3-5-sonnet-latest" {
				t.Fatalf("payload.model = %v", payload["model"])
			}
			if payload["system"] != "system prompt" {
				t.Fatalf("payload.system = %v", payload["system"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content": []map[string]any{{
					"type": "text",
					"text": "{\"answer\":\"ok\",\"command\":\"\"}",
				}},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := New("anthropic", ClientOptions{
		APIKey:  "ak-test",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("New(anthropic) error = %v", err)
	}

	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels error = %v", err)
	}
	if len(models) != 1 || models[0].ID != "claude-3-5-sonnet-latest" {
		t.Fatalf("unexpected models: %+v", models)
	}

	if _, err := client.Ask(context.Background(), AskRequest{
		Model:      "claude-3-5-sonnet-latest",
		Prompt:     "system prompt",
		Question:   "question",
		ExpectJSON: true,
	}); err != nil {
		t.Fatalf("Ask error = %v", err)
	}
}

func TestGeminiEndpointsAndHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1beta/models":
			if got := r.Header.Get("x-goog-api-key"); got != "g-test" {
				t.Fatalf("x-goog-api-key = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"models": []map[string]any{
					{
						"name":                       "models/gemini-2.0-flash",
						"displayName":                "Gemini 2.0 Flash",
						"supportedGenerationMethods": []string{"generateContent"},
					},
					{
						"name":                       "models/text-embedding-004",
						"displayName":                "Text Embedding",
						"supportedGenerationMethods": []string{"embedContent"},
					},
				},
			})
		case "/v1beta/models/gemini-2.0-flash:generateContent":
			if got := r.Header.Get("x-goog-api-key"); got != "g-test" {
				t.Fatalf("x-goog-api-key = %q", got)
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			generationConfig, _ := payload["generationConfig"].(map[string]any)
			if generationConfig["responseMimeType"] != "application/json" {
				t.Fatalf("generationConfig.responseMimeType = %v", generationConfig["responseMimeType"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"candidates": []map[string]any{{
					"content": map[string]any{
						"parts": []map[string]any{
							{"text": "{\"answer\":\"ok\",\"command\":\"\"}"},
						},
					},
				}},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := New("gemini", ClientOptions{
		APIKey:  "g-test",
		BaseURL: server.URL + "/v1beta",
	})
	if err != nil {
		t.Fatalf("New(gemini) error = %v", err)
	}

	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels error = %v", err)
	}
	if len(models) != 1 || models[0].ID != "gemini-2.0-flash" {
		t.Fatalf("unexpected models: %+v", models)
	}

	if _, err := client.Ask(context.Background(), AskRequest{
		Model:      "gemini-2.0-flash",
		Prompt:     "system prompt",
		Question:   "question",
		ExpectJSON: true,
	}); err != nil {
		t.Fatalf("Ask error = %v", err)
	}
}

func TestOllamaEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"models": []map[string]any{{"name": "llama3.2"}},
			})
		case "/api/chat":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			if payload["model"] != "llama3.2" {
				t.Fatalf("payload.model = %v", payload["model"])
			}
			if payload["format"] != "json" {
				t.Fatalf("payload.format = %v", payload["format"])
			}
			if stream, ok := payload["stream"].(bool); !ok || stream {
				t.Fatalf("payload.stream = %v", payload["stream"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": map[string]any{
					"content": "{\"answer\":\"ok\",\"command\":\"\"}",
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := New("ollama", ClientOptions{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New(ollama) error = %v", err)
	}

	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels error = %v", err)
	}
	if len(models) != 1 || models[0].ID != "llama3.2" {
		t.Fatalf("unexpected models: %+v", models)
	}

	if _, err := client.Ask(context.Background(), AskRequest{
		Model:      "llama3.2",
		Prompt:     "system",
		Question:   "question",
		ExpectJSON: true,
	}); err != nil {
		t.Fatalf("Ask error = %v", err)
	}
}
