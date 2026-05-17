package listmodels

import (
	"fmt"
	"io"
		"sort"
	"strings"

	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/modelresolver"
)

// FormatTokenCount formats a number as human-readable (e.g., 200000 -> "200K", 1000000 -> "1M").
func FormatTokenCount(count int) string {
	if count >= 1_000_000 {
		millions := float64(count) / 1_000_000
		if millions == float64(int(millions)) {
			return fmt.Sprintf("%dM", int(millions))
		}
		return fmt.Sprintf("%.1fM", millions)
	}
	if count >= 1_000 {
		thousands := float64(count) / 1_000
		if thousands == float64(int(thousands)) {
			return fmt.Sprintf("%dK", int(thousands))
		}
		return fmt.Sprintf("%.1fK", thousands)
	}
	return fmt.Sprintf("%d", count)
}

// ListModels lists available models, optionally filtered by search pattern.
func ListModels(models []ai.ModelInfo, searchPattern string, w io.Writer) {
	if len(models) == 0 {
		fmt.Fprintln(w, "No models available. Set API keys in environment variables.")
		return
	}

	// Apply fuzzy filter if search pattern provided
	filteredModels := models
	if searchPattern != "" {
		filteredModels = fuzzyFilter(models, searchPattern)
	}

	if len(filteredModels) == 0 {
		fmt.Fprintf(w, "No models matching %q\n", searchPattern)
		return
	}

	// Sort by provider, then by model id
	sort.Slice(filteredModels, func(i, j int) bool {
		providerCmp := strings.Compare(string(filteredModels[i].Provider), string(filteredModels[j].Provider))
		if providerCmp != 0 {
			return providerCmp < 0
		}
		return filteredModels[i].ID < filteredModels[j].ID
	})

	// Calculate column widths
	type row struct {
		provider string
		model    string
		context  string
		maxOut   string
		thinking string
		images   string
	}

	rows := make([]row, len(filteredModels))
	for i, m := range filteredModels {
		thinking := "no"
		if m.Reasoning {
			thinking = "yes"
		}
		images := "no"
		for _, inp := range m.Input {
			if inp == "image" {
				images = "yes"
				break
			}
		}

		rows[i] = row{
			provider: string(m.Provider),
			model:    m.ID,
			context:  FormatTokenCount(m.ContextWindow),
			maxOut:   FormatTokenCount(m.MaxTokens),
			thinking: thinking,
			images:   images,
		}
	}

	headers := row{
		provider: "provider",
		model:    "model",
		context:  "context",
		maxOut:   "max-out",
		thinking: "thinking",
		images:   "images",
	}

	widths := struct {
		provider int
		model    int
		context  int
		maxOut   int
		thinking int
		images   int
	}{
		provider: len(headers.provider),
		model:    len(headers.model),
		context:  len(headers.context),
		maxOut:   len(headers.maxOut),
		thinking: len(headers.thinking),
		images:   len(headers.images),
	}

	for _, r := range rows {
		if len(r.provider) > widths.provider {
			widths.provider = len(r.provider)
		}
		if len(r.model) > widths.model {
			widths.model = len(r.model)
		}
		if len(r.context) > widths.context {
			widths.context = len(r.context)
		}
		if len(r.maxOut) > widths.maxOut {
			widths.maxOut = len(r.maxOut)
		}
		if len(r.thinking) > widths.thinking {
			widths.thinking = len(r.thinking)
		}
		if len(r.images) > widths.images {
			widths.images = len(r.images)
		}
	}

	// Print header
	headerLine := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s",
		widths.provider, headers.provider,
		widths.model, headers.model,
		widths.context, headers.context,
		widths.maxOut, headers.maxOut,
		widths.thinking, headers.thinking,
		widths.images, headers.images,
	)
	fmt.Fprintln(w, headerLine)

	// Print rows
	for _, r := range rows {
		line := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s",
			widths.provider, r.provider,
			widths.model, r.model,
			widths.context, r.context,
			widths.maxOut, r.maxOut,
			widths.thinking, r.thinking,
			widths.images, r.images,
		)
		fmt.Fprintln(w, line)
	}
}

// fuzzyFilter filters models by a fuzzy search pattern.
func fuzzyFilter(models []ai.ModelInfo, pattern string) []ai.ModelInfo {
	p := strings.ToLower(pattern)
	var result []ai.ModelInfo
	for _, m := range models {
		searchable := strings.ToLower(string(m.Provider) + " " + m.ID)
		if strings.Contains(searchable, p) {
			result = append(result, m)
		}
	}
	return result
}

// Ensure modelresolver is referenced
var _ = modelresolver.Resolve
