package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func defaultHTTPClient(input *http.Client) *http.Client {
	if input != nil {
		return input
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func doJSON(ctx context.Context, client *http.Client, req *http.Request, payload any, out any) error {
	if payload != nil {
		buf, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode request JSON: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(buf))
		req.ContentLength = int64(len(buf))
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("provider returned %s: %s", resp.Status, truncate(string(body), 700))
	}

	if out == nil {
		return nil
	}
	if strings.TrimSpace(string(body)) == "" {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response JSON: %w; body=%s", err, truncate(string(body), 700))
	}
	return nil
}

func validateAskRequest(req AskRequest) error {
	if strings.TrimSpace(req.Model) == "" {
		return fmt.Errorf("model is required")
	}
	if strings.TrimSpace(req.Question) == "" {
		return fmt.Errorf("question is required")
	}
	return nil
}

func responseFormatLikelyUnsupported(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "response_format") ||
		strings.Contains(msg, "responsemimetype") ||
		strings.Contains(msg, "response_mime_type")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func joinURL(base, path string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	path = strings.TrimSpace(path)
	if path == "" {
		return base
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

func ensureLeadingSlash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "/") {
		return s
	}
	return "/" + s
}
