package bubbletea

import "testing"

func TestSimpleDiff_Changes(t *testing.T) {
    prev := []string{"line1", "line2"}
    curr := []string{"line1", "line2-mod"}
    out := SimpleDiff(prev, curr, 80)
    if out == "" {
        t.Fatalf("expected diff output, got empty")
    }
    if out[:8] != "\x1b[?2026h" {
        t.Fatalf("expected CSI 2026 start, got %q", out[:8])
    }
    if out[len(out)-8:] != "\x1b[?2026l" {
        t.Fatalf("expected CSI 2026 end, got %q", out[len(out)-8:])
    }
}

func TestSimpleDiff_NoChanges(t *testing.T) {
    prev := []string{"a", "b"}
    curr := []string{"a", "b"}
    out := SimpleDiff(prev, curr, 80)
    if out != "" {
        t.Fatalf("expected empty diff for identical content, got %q", out)
    }
}
