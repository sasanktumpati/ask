package assistant

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Response is the normalized assistant payload consumed by the CLI.
type Response struct {
	Answer  string `json:"answer"`
	Command string `json:"command"`
}

// Parse decodes the model output into the expected JSON response shape.
// It accepts either a raw JSON object or a larger string containing
// the first valid JSON object fragment.
func Parse(text string) (Response, error) {
	candidate := strings.TrimSpace(text)
	if candidate == "" {
		return Response{}, errors.New("empty model response")
	}

	var parsed Response
	if json.Unmarshal([]byte(candidate), &parsed) == nil {
		parsed.normalize()
		return parsed, nil
	}

	fragment, ok := firstJSONObject(candidate)
	if !ok {
		return Response{}, fmt.Errorf("model response is not valid JSON")
	}
	if err := json.Unmarshal([]byte(fragment), &parsed); err != nil {
		return Response{}, fmt.Errorf("decode model JSON response: %w", err)
	}
	parsed.normalize()
	return parsed, nil
}

func (r *Response) normalize() {
	r.Answer = strings.TrimSpace(r.Answer)
	r.Command = strings.TrimSpace(r.Command)
}

// HasCommand reports whether the response includes a runnable command.
func (r Response) HasCommand() bool {
	return strings.TrimSpace(r.Command) != ""
}

func firstJSONObject(s string) (string, bool) {
	start := strings.IndexRune(s, '{')
	if start == -1 {
		return "", false
	}

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		ch := s[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1], true
			}
		}
	}

	return "", false
}

// BuildPrompt returns the system prompt used for provider calls.
// It enforces a strict JSON output contract and includes terminal context.
func BuildPrompt(shell string, cwd string, osName string, allowMarkdown bool) string {
	formatInstruction := "In the answer field, use plain text only (no markdown formatting, headings, bullet markers, or code fences). "
	if allowMarkdown {
		formatInstruction = "In the answer field, use clean Markdown by default (short headings, concise bullet lists, and inline code where helpful). " +
			"Keep formatting readable and minimal. Do not use markdown code fences. "
	}

	instructions := "You are a terminal assistant. Return only strict JSON with exactly these keys: answer, command. " +
		"If the user asks for a terminal command, set command to one runnable command and include concise explanation in answer unless specified otherwise. " +
		"If no command is needed, set command to an empty string. " +
		formatInstruction +
		"Do not include any text outside JSON."

	return fmt.Sprintf("%s\nEnvironment: os=%s, shell=%s, cwd=%s", instructions, osName, shell, cwd)
}
