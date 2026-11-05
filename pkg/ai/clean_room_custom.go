package ai

import (
    "fmt"
    "os"
    "strings"
)

// Additional Claude Code handlers not present in the main clean_room_handlers.go.
// These handlers are registered via an init function to extend the CleanRoomTools map.

// handleClaudeCodeRead implements the read_file tool for Claude Code.
func handleClaudeCodeRead(args map[string]interface{}) string {
    path, ok := args["file_path"].(string)
    if !ok {
        if p, ok2 := args["path"].(string); ok2 {
            path = p
        } else {
            return "Error: missing 'file_path' parameter"
        }
    }

	safePath, err := validatePath(path)
	if err != nil {
		return "Error: security violation: " + err.Error()
	}
    // offset and limit may be float64 (from JSON) or int.
    var offset, limit float64
    if o, ok := args["offset"].(float64); ok {
        offset = o
    } else if oi, ok := args["offset"].(int); ok {
        offset = float64(oi)
    }
    if l, ok := args["limit"].(float64); ok {
        limit = l
    } else if li, ok := args["limit"].(int); ok {
        limit = float64(li)
    }
    // Build unified args for HandleUnifiedRead (expects float values).
    unified := map[string]interface{}{
        "path":   safePath,
        "offset": offset,
        "limit":  limit,
    }
    out, err := HandleUnifiedRead(unified)
    if err != nil {
        return "Error: " + err.Error()
    }
    return out
}

// handleClaudeCodeBash implements the bash tool for Claude Code.
func handleClaudeCodeBash(args map[string]interface{}) string {
    cmd, ok := args["command"].(string)
    if !ok {
        if c, ok2 := args["cmd"].(string); ok2 {
            cmd = c
        } else {
            return "Error: missing 'command' parameter"
        }
    }
    unified := map[string]interface{}{"command": cmd}
    out, err := HandleUnifiedCommand(unified)
    if err != nil {
        return fmt.Sprintf("Error: %v\nOutput: %s", err, out)
    }
    return out
}

// handleClaudeCodeWriteFile implements the write_file tool for Claude Code.
func handleClaudeCodeWriteFile(args map[string]interface{}) string {
    path, ok := args["file_path"].(string)
    if !ok {
        if p, ok2 := args["path"].(string); ok2 {
            path = p
        } else {
            return "Error: missing 'file_path' parameter"
        }
    }
    content, ok := args["content"].(string)
    if !ok {
        return "Error: missing 'content' parameter"
    }

	safePath, err := validatePath(path)
	if err != nil {
		return "Error: security violation: " + err.Error()
	}

	// Call the internal secure wrapper explicitly
	err = doWrite(safePath, content)
    if err != nil {
        return "Error: " + err.Error()
    }
    return "File written successfully"
}

// handleClaudeCodeEdit implements the edit tool for Claude Code (find & replace).
func handleClaudeCodeEdit(args map[string]interface{}) string {
    filePath, ok := args["file_path"].(string)
    if !ok {
        if p, ok2 := args["path"].(string); ok2 {
            filePath = p
        } else {
            return "Error: missing 'file_path' parameter"
        }
    }
    oldStr, ok1 := args["old_string"].(string)
    newStr, ok2 := args["new_string"].(string)
    if !ok1 || !ok2 {
        return "Error: missing 'old_string' or 'new_string' parameter"
    }

	safePath, err := validatePath(filePath)
	if err != nil {
		return "Error: security violation: " + err.Error()
	}

	data, err := os.ReadFile(safePath)
    if err != nil {
        return "Error reading file: " + err.Error()
    }
    if !strings.Contains(string(data), oldStr) {
        return "Error: could not find the specified text to replace"
    }
    updated := strings.Replace(string(data), oldStr, newStr, 1)
	if err := os.WriteFile(safePath, []byte(updated), 0644); err != nil {
        return "Error writing file: " + err.Error()
    }
    return "Edit applied successfully"
}

// handleClaudeCodeListFiles implements the list_files tool for Claude Code.
func handleClaudeCodeListFiles(args map[string]interface{}) string {
    dirPath := "."
    if p, ok := args["path"].(string); ok && p != "" {
        dirPath = p
    }

	safePath, err := validatePath(dirPath)
	if err != nil {
		return "Error: security violation: " + err.Error()
	}

	entries, err := os.ReadDir(safePath)
    if err != nil {
        return "Error: " + err.Error()
    }
    var lines []string
    for _, e := range entries {
        name := e.Name()
        if e.IsDir() {
            name += "/"
        }
        lines = append(lines, name)
    }
    return strings.Join(lines, "\n")
}

func init() {
    // Register the new Claude Code handlers in the global CleanRoomTools map.
    CleanRoomTools["read_file"] = handleClaudeCodeRead
    CleanRoomTools["bash"] = handleClaudeCodeBash
    CleanRoomTools["write_file"] = handleClaudeCodeWriteFile
    CleanRoomTools["edit"] = handleClaudeCodeEdit
    CleanRoomTools["list_files"] = handleClaudeCodeListFiles
    CleanRoomTools["Amp_ReadFile"] = handleAmpReadFile
    CleanRoomTools["Amp_WriteFile"] = handleAmpWriteFile
    CleanRoomTools["Amp_RunCommand"] = handleAmpRunCommand
    // search_files already points to handleHermesSearchFiles, which now supports Claude's "pattern".
}

// handleAmpReadFile implements the Amp_ReadFile tool.
func handleAmpReadFile(args map[string]interface{}) string {
	path, ok := args["path"].(string)
	if !ok {
		return "Error: missing 'path' parameter"
	}

	safePath, err := validatePath(path)
	if err != nil {
		return "Error: security violation: " + err.Error()
	}

	out, err := HandleUnifiedRead(map[string]interface{}{"path": safePath, "offset": float64(0), "limit": float64(0)})
	if err != nil {
		return "Error: " + err.Error()
	}
	return out
}

// handleAmpWriteFile implements the Amp_WriteFile tool.
func handleAmpWriteFile(args map[string]interface{}) string {
	path, ok := args["path"].(string)
	if !ok {
		return "Error: missing 'path' parameter"
	}
	content, ok := args["content"].(string)
	if !ok {
		return "Error: missing 'content' parameter"
	}

	safePath, err := validatePath(path)
	if err != nil {
		return "Error: security violation: " + err.Error()
	}

	err = doWrite(safePath, content)
	if err != nil {
		return "Error: " + err.Error()
	}
	return "File written successfully"
}

// handleAmpRunCommand implements the Amp_RunCommand tool.
func handleAmpRunCommand(args map[string]interface{}) string {
	cmd, ok := args["cmd"].(string)
	if !ok {
		return "Error: missing 'cmd' parameter"
	}

	unified := map[string]interface{}{"command": cmd}
	out, err := HandleUnifiedCommand(unified)
	if err != nil {
		return fmt.Sprintf("Error: %v\nOutput: %s", err, out)
	}
	return out
}
