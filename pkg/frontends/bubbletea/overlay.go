package bubbletea

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// OverlayHandle represents a handle to an active overlay.
type OverlayHandle int

// OverlayOptions configures overlay appearance.
type OverlayOptions struct {
	// Title displayed at the top of the overlay.
	Title string
	// Width of the overlay in characters. Defaults to 60.
	Width int
	// Height of the overlay in lines. Defaults to 10.
	Height int
	// Closeable indicates whether the overlay can be closed with Esc.
	Closeable bool
	// OnClose callback invoked when overlay is closed.
	OnClose func()
}

// OverlayComponent represents a modal overlay.
type OverlayComponent struct {
	content   string
	options   OverlayOptions
	handle    OverlayHandle
	closed    bool
	onCloseFn func()
}

// NewOverlay creates a new overlay component.
func NewOverlay(content string, opts OverlayOptions, handle OverlayHandle) *OverlayComponent {
	if opts.Width <= 0 {
		opts.Width = 60
	}
	if opts.Height <= 0 {
		opts.Height = 10
	}
	return &OverlayComponent{
		content:   content,
		options:   opts,
		handle:    handle,
		onCloseFn: opts.OnClose,
	}
}

// Render draws the overlay with a border and title.
func (o *OverlayComponent) Render(viewportWidth, viewportHeight int) []string {
	if o.closed {
		return []string{}
	}

	width := o.options.Width
	if width > viewportWidth-4 {
		width = viewportWidth - 4
	}
	if width < 20 {
		width = 20
	}

	height := o.options.Height
	if height > viewportHeight-4 {
		height = viewportHeight - 4
	}
	if height < 5 {
		height = 5
	}

	// Split content into lines.
	contentLines := strings.Split(o.content, "\n")

	// Truncate or pad content lines to fit height.
	var displayLines []string
	for i, line := range contentLines {
		if i >= height-2 {
			break
		}
		// Truncate line to fit width.
		if len(line) > width-2 {
			line = line[:width-3] + "…"
		}
		displayLines = append(displayLines, line)
	}

	// Pad with empty lines if needed.
	for len(displayLines) < height-2 {
		displayLines = append(displayLines, "")
	}

	// Build border.
	var lines []string

	// Top border with title.
	title := ""
	if o.options.Title != "" {
		title = " " + o.options.Title + " "
	}
	topBorder := "┌" + strings.Repeat("─", width-2) + "┐"
	if title != "" && len(title) < width-2 {
		// Insert title into top border.
		topBorder = "┌" + strings.Repeat("─", (width-len(title))/2-1) + title + strings.Repeat("─", width-len(title)-(width-len(title))/2-1) + "┐"
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorHeader).Render(topBorder))

	// Content lines with side borders.
	for _, line := range displayLines {
		// Pad line to width.
		padded := line
		for len(padded) < width-2 {
			padded += " "
		}
		lines = append(lines, "│"+padded+"│")
	}

	// Bottom border.
	bottomBorder := "└" + strings.Repeat("─", width-2) + "┘"
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorHeader).Render(bottomBorder))

	// Add close hint if closeable.
	if o.options.Closeable {
		hint := lipgloss.NewStyle().Foreground(ColorSystem).Render(" (Esc to close) ")
		lines = append(lines, hint)
	}

	return lines
}

// Close marks the overlay as closed and invokes the callback.
func (o *OverlayComponent) Close() {
	if !o.closed && o.onCloseFn != nil {
		o.closed = true
		o.onCloseFn()
	}
}

// IsClosed returns whether the overlay is closed.
func (o *OverlayComponent) IsClosed() bool {
	return o.closed
}

// OverlayStack manages multiple overlays.
type OverlayStack struct {
	overlays []*OverlayComponent
	nextID   OverlayHandle
}

// NewOverlayStack creates a new overlay stack.
func NewOverlayStack() *OverlayStack {
	return &OverlayStack{
		overlays: make([]*OverlayComponent, 0),
		nextID:   0,
	}
}

// ShowOverlay adds a new overlay to the stack and returns its handle.
func (s *OverlayStack) ShowOverlay(content string, opts OverlayOptions) OverlayHandle {
	handle := s.nextID
	s.nextID++
	overlay := NewOverlay(content, opts, handle)
	s.overlays = append(s.overlays, overlay)
	return handle
}

// CloseOverlay closes the overlay with the given handle.
func (s *OverlayStack) CloseOverlay(handle OverlayHandle) {
	for _, o := range s.overlays {
		if o.handle == handle {
			o.Close()
			break
		}
	}
	// Remove closed overlays from stack.
	s.removeClosed()
}

// CloseTop closes the most recently added overlay.
func (s *OverlayStack) CloseTop() {
	if len(s.overlays) == 0 {
		return
	}
	s.overlays[len(s.overlays)-1].Close()
	s.removeClosed()
}

// removeClosed removes all closed overlays from the stack.
func (s *OverlayStack) removeClosed() {
	active := make([]*OverlayComponent, 0, len(s.overlays))
	for _, o := range s.overlays {
		if !o.IsClosed() {
			active = append(active, o)
		}
	}
	s.overlays = active
}

// HasOverlays returns whether there are any active overlays.
func (s *OverlayStack) HasOverlays() bool {
	s.removeClosed()
	return len(s.overlays) > 0
}

// Render renders all active overlays, centered on the screen.
// Returns lines to be appended to the main view.
func (s *OverlayStack) Render(viewportWidth, viewportHeight int) []string {
	s.removeClosed()
	if len(s.overlays) == 0 {
		return []string{}
	}

	// Render only the top overlay.
	top := s.overlays[len(s.overlays)-1]
	return top.Render(viewportWidth, viewportHeight)
}

// OverlayMsg is a tea.Msg for overlay operations.
type OverlayMsg struct {
	Type    OverlayMsgType
	Handle  OverlayHandle
	Content string
	Options OverlayOptions
}

// OverlayMsgType indicates the type of overlay operation.
type OverlayMsgType int

const (
	OverlayShow OverlayMsgType = iota
	OverlayClose
	OverlayCloseTop
)

// ShowOverlayCmd returns a tea.Cmd to show an overlay.
func ShowOverlayCmd(content string, opts OverlayOptions) tea.Cmd {
	return func() tea.Msg {
		return OverlayMsg{
			Type:    OverlayShow,
			Content: content,
			Options: opts,
		}
	}
}

// CloseOverlayCmd returns a tea.Cmd to close an overlay.
func CloseOverlayCmd(handle OverlayHandle) tea.Cmd {
	return func() tea.Msg {
		return OverlayMsg{
			Type:   OverlayClose,
			Handle: handle,
		}
	}
}

// CloseTopOverlayCmd returns a tea.Cmd to close the top overlay.
func CloseTopOverlayCmd() tea.Cmd {
	return func() tea.Msg {
		return OverlayMsg{
			Type: OverlayCloseTop,
		}
	}
}