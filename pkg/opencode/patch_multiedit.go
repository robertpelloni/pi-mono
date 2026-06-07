package opencode

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ApplyPatchParams struct {
	PatchText string `json:"patchText"`
}

type PatchHunk struct {
	Path     string
	Type     string // "add", "update", "delete"
	Contents string
	Chunks   []PatchChunk
	MovePath string
}

type PatchChunk struct {
	Type    string // "context", "add", "remove"
	Content string
}

func ParsePatch(patchText string) ([]PatchHunk, error) {
	patchText = strings.ReplaceAll(patchText, "\r\n", "\n")
	lines := strings.Split(patchText, "\n")

	var hunks []PatchHunk
	var current *PatchHunk

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if strings.HasPrefix(line, "*** ") {
			if current != nil {
				hunks = append(hunks, *current)
				current = nil
			}

			if line == "*** Begin Patch" || line == "*** End Patch" {
				continue
			}

			if strings.HasPrefix(line, "*** Add File: ") {
				current = &PatchHunk{
					Path: strings.TrimPrefix(line, "*** Add File: "),
					Type: "add",
				}
				continue
			}

			if strings.HasPrefix(line, "*** Update File: ") {
				rest := strings.TrimPrefix(line, "*** Update File: ")
				parts := strings.SplitN(rest, " -> ", 2)
				hunk := &PatchHunk{
					Path: parts[0],
					Type: "update",
				}
				if len(parts) == 2 {
					hunk.MovePath = parts[1]
				}
				current = hunk
				continue
			}

			if strings.HasPrefix(line, "*** Delete File: ") {
				current = &PatchHunk{
					Path: strings.TrimPrefix(line, "*** Delete File: "),
					Type: "delete",
				}
				continue
			}
		}

		if current == nil {
			continue
		}

		switch current.Type {
		case "add":
			if current.Contents != "" {
				current.Contents += "\n"
			}
			current.Contents += line
		case "update":
			if strings.HasPrefix(line, "+") {
				current.Chunks = append(current.Chunks, PatchChunk{Type: "add", Content: line[1:]})
			} else if strings.HasPrefix(line, "-") {
				current.Chunks = append(current.Chunks, PatchChunk{Type: "remove", Content: line[1:]})
			} else {
				current.Chunks = append(current.Chunks, PatchChunk{Type: "context", Content: line})
			}
		}
	}

	if current != nil {
		hunks = append(hunks, *current)
	}

	return hunks, nil
}

type ApplyPatchResult struct {
	Files []PatchFileResult
}

type PatchFileResult struct {
	Path      string
	Type      string
	Additions int
	Deletions int
	Content   string
	Err       error
}

func ApplyPatch(hunks []PatchHunk, cwd string) ApplyPatchResult {
	var result ApplyPatchResult

	for _, hunk := range hunks {
		absPath := resolveFilePath(hunk.Path, cwd)
		switch hunk.Type {
		case "add":
			result.Files = append(result.Files, applyPatchAdd(absPath, hunk))
		case "update":
			result.Files = append(result.Files, applyPatchUpdate(absPath, hunk))
		case "delete":
			result.Files = append(result.Files, applyPatchDelete(absPath))
		}
	}

	return result
}

func applyPatchAdd(absPath string, hunk PatchHunk) PatchFileResult {
	content := hunk.Contents
	if !strings.HasSuffix(content, "\n") && content != "" {
		content += "\n"
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return PatchFileResult{Path: absPath, Type: "add", Err: err}
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		return PatchFileResult{Path: absPath, Type: "add", Err: err}
	}

	return PatchFileResult{
		Path:      absPath,
		Type:      "add",
		Additions: strings.Count(content, "\n"),
		Content:   content,
	}
}

func applyPatchUpdate(absPath string, hunk PatchHunk) PatchFileResult {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return PatchFileResult{Path: absPath, Type: "update", Err: fmt.Errorf("failed to read file: %w", err)}
	}

	original := string(data)
	newContent, additions, deletions := applyChunks(original, hunk.Chunks)

	targetPath := absPath
	if hunk.MovePath != "" {
		targetPath = hunk.MovePath
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return PatchFileResult{Path: absPath, Type: "update", Err: err}
		}
	}

	if err := os.WriteFile(targetPath, []byte(newContent), 0o644); err != nil {
		return PatchFileResult{Path: targetPath, Type: "update", Err: err}
	}

	return PatchFileResult{
		Path:      targetPath,
		Type:      "update",
		Additions: additions,
		Deletions: deletions,
		Content:   newContent,
	}
}

func applyPatchDelete(absPath string) PatchFileResult {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return PatchFileResult{Path: absPath, Type: "delete", Err: err}
	}
	lines := strings.Count(string(data), "\n")

	if err := os.Remove(absPath); err != nil {
		return PatchFileResult{Path: absPath, Type: "delete", Err: err}
	}

	return PatchFileResult{
		Path:      absPath,
		Type:      "delete",
		Deletions: lines,
	}
}

func applyChunks(original string, chunks []PatchChunk) (string, int, int) {
	originalLines := strings.Split(original, "\n")
	var result []string
	additions, deletions := 0, 0

	origIdx := 0
	for _, chunk := range chunks {
		switch chunk.Type {
		case "context":
			if origIdx < len(originalLines) {
				result = append(result, originalLines[origIdx])
				origIdx++
			}
		case "remove":
			origIdx++
			deletions++
		case "add":
			result = append(result, chunk.Content)
			additions++
		}
	}

	for ; origIdx < len(originalLines); origIdx++ {
		result = append(result, originalLines[origIdx])
	}

	return strings.Join(result, "\n"), additions, deletions
}

type MultiEditParams struct {
	FilePath string          `json:"filePath"`
	Edits    []MultiEditItem `json:"edits"`
}

type MultiEditItem struct {
	OldString  string `json:"oldString"`
	NewString  string `json:"newString"`
	ReplaceAll bool   `json:"replaceAll,omitempty"`
}

func ApplyMultiEdit(params MultiEditParams) (*EditResult, error) {
	data, err := os.ReadFile(params.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)
	totalAdd, totalDel := 0, 0

	for i, edit := range params.Edits {
		if edit.OldString == edit.NewString {
			continue
		}

		if edit.ReplaceAll {
			count := strings.Count(content, edit.OldString)
			content = strings.ReplaceAll(content, edit.OldString, edit.NewString)
			totalAdd += count
			totalDel += count
		} else {
			if !strings.Contains(content, edit.OldString) {
				return nil, fmt.Errorf("edit %d: oldString not found in file", i)
			}
			content = strings.Replace(content, edit.OldString, edit.NewString, 1)
			totalAdd++
			totalDel++
		}
	}

	if err := os.WriteFile(params.FilePath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return &EditResult{
		Path:      params.FilePath,
		Additions: totalAdd,
		Deletions: totalDel,
		Content:   content,
	}, nil
}

type EditResult struct {
	Path      string
	Additions int
	Deletions int
	Content   string
}

func resolveFilePath(path, cwd string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(cwd, path)
}
