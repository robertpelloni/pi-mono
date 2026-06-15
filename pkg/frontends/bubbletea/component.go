package bubbletea

import "strings"

// Component represents a UI element that can render itself, handle input and be invalidated.
// This mirrors the TypeScript `Component` interface used by @mariozechner/pi-tui.
// All UI pieces in the Go TUI will implement this contract.

type Component interface {
    // Render returns the visual representation of the component as a slice of lines.
    // The width argument is the maximum allowed line width; implementations should
    // truncate or wrap as necessary.
    Render(width int) []string
    // HandleInput receives raw terminal data (key presses, mouse events, etc.).
    // Components that are focusable should react accordingly.
    HandleInput(data string)
    // Invalidate clears any cached rendering state. It is called when the
    // component needs to recompute its output (e.g., theme changes).
    Invalidate()
}

// Container groups child components and renders them sequentially.
// It is analogous to the TS `Container` class.

type Container struct {
    children []Component
}

// NewContainer creates an empty container.
func NewContainer() *Container {
    return &Container{children: []Component{}}
}

// AddChild appends a component to the container.
func (c *Container) AddChild(comp Component) {
    c.children = append(c.children, comp)
}

// RemoveChild removes a component from the container if present.
func (c *Container) RemoveChild(comp Component) {
    for i, child := range c.children {
        if child == comp {
            c.children = append(c.children[:i], c.children[i+1:]...)
            break
        }
    }
}

// Render concatenates the rendered output of all children.
func (c *Container) Render(width int) []string {
    var out []string
    for _, child := range c.children {
        out = append(out, child.Render(width)...)
    }
    return out
}

// HandleInput forwards the input to the focused child, if any.
// For simplicity, we forward to every child; concrete components may ignore it.
func (c *Container) HandleInput(data string) {
    for _, child := range c.children {
        child.HandleInput(data)
    }
}

// Invalidate propagates the call to all children.
func (c *Container) Invalidate() {
    for _, child := range c.children {
        child.Invalidate()
    }
}

// Text is a simple component that renders static lines.
type Text struct {
    content string
    // Optional style function that can be applied to each line.
    style func(string) string
}

// NewText creates a Text component.
func NewText(content string, styleFn func(string) string) *Text {
    return &Text{content: content, style: styleFn}
}

// Render splits the content by newlines and optionally styles each line.
func (t *Text) Render(width int) []string {
    lines := strings.Split(t.content, "\n")
    if t.style != nil {
        for i, l := range lines {
            lines[i] = t.style(l)
        }
    }
    // Ensure we do not exceed the width; truncate if needed.
    for i, l := range lines {
        if len(l) > width {
            lines[i] = l[:width]
        }
    }
    return lines
}

func (t *Text) HandleInput(_ string) {}
func (t *Text) Invalidate()        {}

// Spacer creates empty vertical space.
type Spacer struct {
    lines int
}

func NewSpacer(lines int) *Spacer {
    if lines < 0 {
        lines = 0
    }
    return &Spacer{lines: lines}
}

func (s *Spacer) Render(width int) []string {
    out := make([]string, s.lines)
    for i := 0; i < s.lines; i++ {
        out[i] = strings.Repeat(" ", width)
    }
    return out
}
func (s *Spacer) HandleInput(_ string) {}
func (s *Spacer) Invalidate()        {}
