package bubbletea

import (
    "strings"
    "github.com/charmbracelet/bubbles/textarea"
    tea "github.com/charmbracelet/bubbletea"
)

// EditorComponent is a boxed multi‑line text editor that wraps a textarea.Model.
// It implements the Component interface. The underlying textarea is expected to
// be initialised (placeholder, prompt, etc.) before being passed to NewEditor.

type EditorComponent struct {
    ta *textarea.Model
}

// NewEditor creates an EditorComponent that delegates to the provided textarea.
func NewEditor(ta *textarea.Model) *EditorComponent {
    return &EditorComponent{ta: ta}
}

// Render draws the editor with a border.
func (e *EditorComponent) Render(width int) []string {
    view := e.ta.View()
    lines := strings.Split(view, "\n")
    // Ensure we have at least one line.
    if len(lines) == 0 {
        lines = []string{""}
    }
    // Pad/truncate each line to fit within the border.
    innerWidth := width - 2
    if innerWidth < 1 {
        innerWidth = 1
    }
    for i, l := range lines {
        if len(l) > innerWidth {
            lines[i] = l[:innerWidth]
        } else if len(l) < innerWidth {
            lines[i] = l + strings.Repeat(" ", innerWidth-len(l))
        }
    }
    top := "┌" + strings.Repeat("─", innerWidth) + "┐"
    bottom := "└" + strings.Repeat("─", innerWidth) + "┘"
    out := []string{top}
    for _, l := range lines {
        out = append(out, "│"+l+"│")
    }
    out = append(out, bottom)
    return out
}

func (e *EditorComponent) HandleInput(data string) {
    // Convert the raw input into a tea.Msg and forward to the textarea.
    var msg tea.Msg
    if len(data) == 1 {
        msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune(data[0])}}
    } else {
        msg = tea.Msg(data)
    }
    // Update the textarea model. We ignore returned command.
    *e.ta, _ = (*e.ta).Update(msg)
}

func (e *EditorComponent) Invalidate() {}

// Focus and Blur forward to the textarea model.
func (e *EditorComponent) Focus()   { e.ta.Focus() }
func (e *EditorComponent) Blur()    { e.ta.Blur() }

// Expose underlying textarea functionality for the rest of the UI.
func (e *EditorComponent) GetValue() string   { return e.ta.Value() }
func (e *EditorComponent) SetValue(v string) { e.ta.SetValue(v) }
func (e *EditorComponent) SetCursor(p int)   { e.ta.SetCursor(p) }
