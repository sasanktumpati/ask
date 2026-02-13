package render

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
)

var rendererCache sync.Map

// Markdown renders text as terminal markdown when enabled.
// It falls back to plain trimmed text if rendering fails.
func Markdown(text string, width int, enabled bool) string {
	clean := strings.TrimSpace(text)
	if clean == "" {
		return ""
	}
	if !enabled {
		return clean
	}
	if width <= 0 {
		width = 100
	}

	renderer, err := getRenderer(width)
	if err != nil {
		return clean
	}
	out, err := renderer.Render(clean)
	if err != nil {
		return clean
	}
	return out
}

func getRenderer(width int) (*glamour.TermRenderer, error) {
	if cached, ok := rendererCache.Load(width); ok {
		if r, ok := cached.(*glamour.TermRenderer); ok {
			return r, nil
		}
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, fmt.Errorf("create markdown renderer: %w", err)
	}
	rendererCache.Store(width, renderer)
	return renderer, nil
}
