package ai

import (
    "fmt"
    "io/ioutil"
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
        "path":   path,
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
    out, err := handleHermesWriteFile(map[string]interface{}{"file_path": path, "content": content})
    if err != nil {
        return "Error: " + err.Error()
    }
    return out
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
    data, err := ioutil.ReadFile(filePath)
    if err != nil {
        return "Error reading file: " + err.Error()
    }
    if !strings.Contains(string(data), oldStr) {
        return "Error: could not find the specified text to replace"
    }
    updated := strings.Replace(string(data), oldStr, newStr, 1)
    if err := ioutil.WriteFile(filePath, []byte(updated), 0644); err != nil {
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
    entries, err := ioutil.ReadDir(dirPath)
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
    CleanRoomTools["amp_diff"] = handleAmpDiff
    CleanRoomTools["amp_review"] = handleAmpReview
    // search_files already points to handleHermesSearchFiles, which now supports Claude's "pattern".
}
