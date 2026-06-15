package bubbletea

import (
	"strings"
	"testing"
)

func TestContainer_Render(t *testing.T) {
    c := NewContainer()
    c.AddChild(NewText("hello", nil))
    c.AddChild(NewSpacer(1))
    out := c.Render(80)
    if len(out) != 2 {
        t.Fatalf("expected 2 lines, got %d", len(out))
    }
    if out[0] != "hello" {
        t.Errorf("first line mismatch, got %q", out[0])
    }
    if out[1] != strings.Repeat(" ", 80) {
        t.Errorf("spacer line should be spaces, got %q", out[1])
    }
}

func TestText_Render_Truncates(t *testing.T) {
    txt := NewText("this is a very long line", nil)
    out := txt.Render(10)
    if len(out) != 1 {
        t.Fatalf("expected 1 line, got %d", len(out))
    }
    if len(out[0]) != 10 {
        t.Errorf("expected line length 10, got %d", len(out[0]))
    }
}

func TestText_Render_Style(t *testing.T) {
    styleFn := func(s string) string { return "X" + s }
    txt := NewText("abc", styleFn)
    out := txt.Render(20)
    if out[0] != "Xabc" {
        t.Errorf("style not applied, got %q", out[0])
    }
}

func TestContainer_RemoveChild(t *testing.T) {
    c := NewContainer()
    t1 := NewText("a", nil)
    t2 := NewText("b", nil)
    c.AddChild(t1)
    c.AddChild(t2)
    c.RemoveChild(t1)
    out := c.Render(10)
    if len(out) != 1 {
        t.Fatalf("expected 1 line after removal, got %d", len(out))
    }
    if out[0] != "b" {
        t.Errorf("expected 'b', got %q", out[0])
    }
}