package edittool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/editdiff"
)

// EditToolDetails contains metadata about an edit operation.
type EditToolDetails struct {
	Diff             string `json:"diff"`
	FirstChangedLine int    `json:"firstChangedLine"`
}

// CreateEditTool creates the edit tool that applies exact text replacements.
func CreateEditTool(cwd string) agent.AgentTool {
	return agent.AgentTool{
		Name:        "edit",
		Label:       "edit",
		Description: "Edit a single file using exact text replacement. Every edits[].oldText must match a unique, non-overlapping region of the original file. If two changes affect the same block or nearby lines, merge them into one edit instead of emitting overlapping edits. Do not include large unchanged regions just to connect distant changes.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file to edit (relative or absolute)",
				},
				"edits": map[string]interface{}{
					"type":        "array",
					"description": "One or more targeted replacements. Each edit is matched against the original file, not incrementally. Do not include overlapping or nested edits.",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"oldText": map[string]interface{}{
								"type":        "string",
								"description": "Exact text for one targeted replacement. It must be unique in the original file and must not overlap with any other edits[].oldText in the same call.",
							},
							"newText": map[string]interface{}{
								"type":        "string",
								"description": "Replacement text for this targeted edit.",
							},
						},
						"required": []string{"oldText", "newText"},
					},
				},
			},
			"required": []string{"path", "edits"},
		},
		PromptSnippet: "Make precise file edits with exact text replacement, including multiple disjoint edits in one call",
		PromptGuidelines: []string{
			"Use edit for precise changes (edits[].oldText must match exactly)",
			"When changing multiple separate locations in one file, use one edit call with multiple entries in edits[] instead of multiple edit calls",
			"Each edits[].oldText is matched against the original file, not after earlier edits are applied. Do not emit overlapping or nested edits.",
			"Keep edits[].oldText as small as possible while still being unique in the file. Do not pad with large unchanged regions.",
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			path, _ := params["path"].(string)
			if path == "" {
				return agent.AgentToolResult{}, fmt.Errorf("missing path parameter")
			}

			// Resolve path
			absolutePath := path
			if !filepath.IsAbs(path) {
				absolutePath = filepath.Join(cwd, path)
			}

			// Parse edits
			editsRaw, _ := params["edits"].([]interface{})
			if len(editsRaw) == 0 {
				// Check for legacy oldText/newText format
				if oldText, ok := params["oldText"].(string); ok {
					if newText, ok := params["newText"].(string); ok {
						editsRaw = []interface{}{
							map[string]interface{}{
								"oldText": oldText,
								"newText": newText,
							},
						}
					}
				}
			}

			if len(editsRaw) == 0 {
				return agent.AgentToolResult{}, fmt.Errorf("edits must contain at least one replacement")
			}

			var edits []editdiff.Edit
			for i, e := range editsRaw {
				editMap, ok := e.(map[string]interface{})
				if !ok {
					continue
				}
				oldText, _ := editMap["oldText"].(string)
				newText, _ := editMap["newText"].(string)
				if oldText == "" {
					return agent.AgentToolResult{}, fmt.Errorf("edits[%d].oldText must not be empty", i)
				}
				edits = append(edits, editdiff.Edit{OldText: oldText, NewText: newText})
			}

			// Check if file exists
			info, err := os.Stat(absolutePath)
			if err != nil {
				return agent.AgentToolResult{}, fmt.Errorf("file not found: %s", path)
			}
			if info.IsDir() {
				return agent.AgentToolResult{}, fmt.Errorf("path is a directory: %s", path)
			}

			// Read the file
			data, err := os.ReadFile(absolutePath)
			if err != nil {
				return agent.AgentToolResult{}, fmt.Errorf("cannot read file: %w", err)
			}

			rawContent := string(data)

			// Strip BOM before matching
			_, content := editdiff.StripBom(rawContent)

			// Detect and normalize line endings
			originalEnding := editdiff.DetectLineEnding(content)
			normalizedContent := editdiff.NormalizeToLF(content)

			// Apply edits
			result, err := editdiff.ApplyEditsToNormalizedContent(normalizedContent, edits, path)
			if err != nil {
				return agent.AgentToolResult{}, err
			}

			// Restore original line endings
			finalContent := editdiff.RestoreLineEndings(result.NewContent, originalEnding)

			// Write back
			err = os.WriteFile(absolutePath, []byte(finalContent), info.Mode())
			if err != nil {
				return agent.AgentToolResult{}, fmt.Errorf("cannot write file: %w", err)
			}

			// Generate diff
			diffResult := editdiff.GenerateDiffString(result.BaseContent, result.NewContent)

			return agent.AgentToolResult{
				Content: []ai.Content{
					ai.TextContent{Text: fmt.Sprintf("Successfully replaced %d block(s) in %s.", len(edits), path)},
				},
				Details: map[string]any{
					"diff":             diffResult.Diff,
					"firstChangedLine": diffResult.FirstChangedLine,
				},
			}, nil
		},
	}
}

// ComputeEditsDiff computes the diff for edits without applying them.
// Used for preview rendering in the TUI.
func ComputeEditsDiff(path string, edits []editdiff.Edit, cwd string) (*editdiff.DiffResult, error) {
	absolutePath := path
	if !filepath.IsAbs(path) {
		absolutePath = filepath.Join(cwd, path)
	}

	data, err := os.ReadFile(absolutePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	_, content := editdiff.StripBom(string(data))
	normalizedContent := editdiff.NormalizeToLF(content)

	result, err := editdiff.ApplyEditsToNormalizedContent(normalizedContent, edits, path)
	if err != nil {
		return nil, err
	}

	diffResult := editdiff.GenerateDiffString(result.BaseContent, result.NewContent)
	return &diffResult, nil
}

// Ensure strings import is used
var _ = strings.HasPrefix
