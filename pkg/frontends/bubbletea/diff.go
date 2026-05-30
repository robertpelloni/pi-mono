package bubbletea

import (
	"strings"
)

// RenderDiff colorizes a unified diff string for terminal display.
func RenderDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			lines[i] = StyleDiffAdd.Render(line)
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			lines[i] = StyleDiffDel.Render(line)
		}
	}
	return strings.Join(lines, "\n")
}
