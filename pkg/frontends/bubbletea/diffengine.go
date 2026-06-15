package bubbletea

import "strings"

// RenderAtomic wraps the given content in the CSI 2026 synchronized output block.
// This mimics the TypeScript TUI's atomic screen updates, ensuring flicker‑free
// rendering on terminals that support the feature.
func RenderAtomic(content string) string {
    const start = "\x1b[?2026h"
    const end = "\x1b[?2026l"
    return start + content + end
}

// SimpleDiff renders the full screen if any line changed. For the initial
// implementation we keep it simple – a full clear and rewrite – but the API
// allows future optimisations.
func SimpleDiff(prev []string, curr []string, width int) string {
    // If previous and current are identical, return empty string.
    if len(prev) == len(curr) {
        identical := true
        for i := range prev {
            if prev[i] != curr[i] {
                identical = false
                break
            }
        }
        if identical {
            return ""
        }
    }
    // Fallback: render the whole screen atomically.
    var b strings.Builder
    for _, line := range curr {
        // Ensure line does not exceed width.
        if len(line) > width {
            b.WriteString(line[:width])
        } else {
            b.WriteString(line)
        }
        b.WriteByte('\n')
    }
    return RenderAtomic(b.String())
}
