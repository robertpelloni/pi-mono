package bubbletea

import "testing"

func TestMarkdown_Render(t *testing.T) {
    src := "# Title\n\n**bold** and _italic_"
    md := NewMarkdown(src, "dark")
    lines := md.Render(80)
    if len(lines) == 0 {
        t.Fatalf("expected at least one rendered line")
    }
    // The rendered output should contain ANSI escape sequences for styling.
    found := false
    for _, l := range lines {
        if len(l) > 0 && l[0] == '\x1b' {
            found = true
            break
        }
    }
    if !found {
        t.Fatalf("expected ANSI escape codes in rendered markdown, got %q", lines)
    }
}
