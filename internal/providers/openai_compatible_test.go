package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAICompatible_ListModelsAndAsk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
				t.Fatalf("Authorization header = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": "gpt-test"}},
			})
		case "/v1/chat/completions":
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

	client, err := NewOpenAICompatible(OpenAICompatibleSettings{Name: "proxy", RequireAPIKey: true}, ClientOptions{
		APIKey:  "test-key",
		BaseURL: server.URL + "/v1",
	})
	if err != nil {
		t.Fatalf("NewOpenAICompatible error = %v", err)
	}

	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels error = %v", err)
	}
	if len(models) != 1 || models[0].ID != "gpt-test" {
		t.Fatalf("unexpected models: %+v", models)
	}

	resp, err := client.Ask(context.Background(), AskRequest{
		Model:    "gpt-test",
		Prompt:   "system",
		Question: "hello",
	})
	if err != nil {
		t.Fatalf("Ask error = %v", err)
	}
	if resp.Text == "" {
		t.Fatal("expected non-empty text")
	}
}

func TestOpenAICompatible_ArrayMessageContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{"content": []map[string]any{{"text": "part-1"}, {"text": "part-2"}}},
			}},
		})
	}))
	defer server.Close()

	client, err := NewOpenAICompatible(OpenAICompatibleSettings{
		Name:          "proxy",
		ChatPath:      "/chat/completions",
		RequireAPIKey: true,
	}, ClientOptions{
		APIKey:  "k",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("NewOpenAICompatible error = %v", err)
	}

	resp, err := client.Ask(context.Background(), AskRequest{Model: "m", Prompt: "p", Question: "q"})
	if err != nil {
		t.Fatalf("Ask error = %v", err)
	}
	if want := "part-1\npart-2"; resp.Text != want {
		t.Fatalf("resp.Text = %q, want %q", resp.Text, want)
	}
}

func TestOpenAICompatible_CustomProviderWithoutAPIKey(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/chat/completions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{"content": "{\"answer\":\"ok\",\"command\":\"\"}"},
			}},
		})
	}))
	defer server.Close()

	client, err := NewOpenAICompatible(OpenAICompatibleSettings{
		Name:     "myproxy",
		ChatPath: "/chat/completions",
	}, ClientOptions{
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("NewOpenAICompatible error = %v", err)
	}

	_, err = client.Ask(context.Background(), AskRequest{
		Model:    "m",
		Prompt:   "p",
		Question: "q",
	})
	if err != nil {
		t.Fatalf("Ask error = %v", err)
	}
	if gotAuth != "" {
		t.Fatalf("Authorization header should be empty for custom provider without API key, got %q", gotAuth)
	}
}
