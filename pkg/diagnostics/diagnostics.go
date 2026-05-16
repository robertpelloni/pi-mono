package diagnostics

// ResourceDiagnostic represents a non-fatal issue discovered during resource loading.
type ResourceDiagnostic struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Source  string `json:"source,omitempty"` // "skill", "prompt", "theme", "context"
}

// FormatDiagnostics formats a list of diagnostics as a human-readable string.
func FormatDiagnostics(diagnostics []ResourceDiagnostic) string {
	if len(diagnostics) == 0 {
		return ""
	}
	var lines []string
	for _, d := range diagnostics {
		line := d.Message
		if d.Path != "" {
			line = d.Path + ": " + line
		}
		if d.Source != "" {
			line = "[" + d.Source + "] " + line
		}
		lines = append(lines, line)
	}
	return "Warnings:\n  " + joinStrings(lines, "\n  ")
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
