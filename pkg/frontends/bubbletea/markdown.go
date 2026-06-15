package bubbletea

import (
    "github.com/charmbracelet/glamour"
    "strings"
)

// Markdown component renders markdown text as ANSI using the charmbracelet/glamour library.
// It implements the Component interface.

type Markdown struct {
    // raw markdown source
    src string
    // optional glamour style (e.g., "dark", "light", custom JSON)
    style string
    // cached rendered lines to avoid re‑rendering on every frame
    cached []string
    // width used during the last render (for truncation/ wrapping)
    lastWidth int
}

// NewMarkdown creates a Markdown component. If style is empty the default "dark" style is used.
func NewMarkdown(src string, style string) *Markdown {
    if style == "" {
        style = "dark"
    }
    return &Markdown{src: src, style: style}
}

// SetSource updates the markdown source and invalidates the cache.
func (m *Markdown) SetSource(src string) {
    if src != m.src {
        m.src = src
        m.cached = nil
    }
}

// Render converts the markdown source to ANSI escape‑coded lines. The result is cached
// until SetSource or Invalidate is called.
func (m *Markdown) Render(width int) []string {
    // Re‑render only if cache is missing or width changed.
    if m.cached == nil || m.lastWidth != width {
        // Use glamour with the desired style.
        r, err := glamour.NewTermRenderer(
            glamour.WithWordWrap(width),
            glamour.WithStandardStyle(m.style),
        )
        if err != nil {
            // Fallback to raw text on error.
            m.cached = []string{m.src}
        } else {
            rendered, err := r.Render(m.src)
            if err != nil {
                m.cached = []string{m.src}
            } else {
                // glamour adds a trailing newline; split into lines.
                // Trim any empty trailing line for consistency.
                lines := strings.Split(strings.TrimRight(rendered, "\n"), "\n")
                m.cached = lines
            }
        }
        m.lastWidth = width
    }
    return m.cached
}

func (m *Markdown) HandleInput(_ string) {}
func (m *Markdown) Invalidate() {
    m.cached = nil
}
