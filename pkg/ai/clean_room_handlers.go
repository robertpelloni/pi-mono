package ai

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
)

// CleanRoomToolHandlers defines the unified underlying implementations for our exact parity tools.

// HandleUnifiedRead takes unified parameters (where path is guaranteed to be set) and returns file contents.
func HandleUnifiedRead(unifiedArgs map[string]interface{}) (string, error) {
	path, ok := unifiedArgs["path"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'path' parameter")
	}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("error reading file %s: %v", path, err)
	}

	// Apply offset/limit if provided
	lines := strings.Split(string(content), "\n")

	offset := 0
	if o, ok := unifiedArgs["offset"].(float64); ok {
		offset = int(o)
	}
	if offset < 0 {
		offset = 0
	}

	limit := len(lines)
	if l, ok := unifiedArgs["limit"].(float64); ok {
		limit = int(l)
	}
	if limit < 0 {
		limit = 0
	}

	start := offset
	if start >= len(lines) {
		return "", fmt.Errorf("offset %d is beyond end of file (%d lines)", start, len(lines))
	}

	end := start + limit
	if end > len(lines) {
		end = len(lines)
	}
	if end < start {
		end = start
	}

	return strings.Join(lines[start:end], "\n"), nil
}

// HandleUnifiedCommand takes unified parameters (where command is guaranteed to be set) and executes a bash command.
func HandleUnifiedCommand(unifiedArgs map[string]interface{}) (string, error) {
	command, ok := unifiedArgs["command"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'command' parameter")
	}

	cmd := exec.Command("bash", "-c", command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("command execution failed: %v\nOutput: %s", err, string(out))
	}

	return string(out), nil
}

// HandleHermesPatch implements the find-and-replace logic for the Hermes patch tool.
func HandleHermesPatch(unifiedArgs map[string]interface{}) (string, error) {
	filePath, ok := unifiedArgs["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("missing file_path")
	}
	findStr, ok := unifiedArgs["find"].(string)
	if !ok {
		return "", fmt.Errorf("missing find string")
	}
	replaceStr, ok := unifiedArgs["replace"].(string)
	if !ok {
		return "", fmt.Errorf("missing replace string")
	}

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, findStr) {
		return "", fmt.Errorf("target string not found in file")
	}

	// Basic string replacement. A full implementation would use fuzzy matching as documented.
	newContent := strings.Replace(contentStr, findStr, replaceStr, 1)

	err = ioutil.WriteFile(filePath, []byte(newContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	return "Patch applied successfully", nil
}

// HandleHermesWriteFile implements the destructive file overwrite for the Hermes write_file tool.
func HandleHermesWriteFile(unifiedArgs map[string]interface{}) (string, error) {
	filePath, ok := unifiedArgs["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("missing file_path")
	}
	content, ok := unifiedArgs["content"].(string)
	if !ok {
		return "", fmt.Errorf("missing content")
	}

	err := ioutil.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	return "File written successfully", nil
}

// MapHermesCleanRoomParams routes Hermes-specific parameter structures to the unified handlers.
func MapHermesCleanRoomParams(toolName string, rawArgs []byte) (map[string]interface{}, error) {
	var args map[string]interface{}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, err
	}

	unified := make(map[string]interface{})
	for k, v := range args {
		unified[k] = v
	}

	// Normalizing paths for Hermes schemas
	if path, ok := args["file_path"].(string); ok {
		unified["path"] = path
	}

	return unified, nil
}
