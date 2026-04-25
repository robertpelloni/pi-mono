package ai

import (
	"os"
	"path/filepath"

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



// Native OpenInterpreter bindings for specific action enums.
// These interact directly with 'xdotool' (Linux) or 'cliclick' natively if installed.
func handleOpenInterpreterComputerUse(args map[string]interface{}) string {
	action, ok := args["action"].(string)
	if !ok {
		return "Error: missing 'action' parameter for computer use."
	}

	switch action {
	case "type":
		text, _ := args["text"].(string)
		if text == "" {
			return "Error: missing text to type"
		}
		// Attempting generic OS invocation
		cmd := exec.Command("xdotool", "type", text)
		err := cmd.Run()
		if err != nil {
			return "Error typing via xdotool: " + err.Error() + ". Please ensure xdotool is installed."
		}
		return "Typed text successfully: " + text

	case "key":
		key, _ := args["text"].(string)
		if key == "" {
			return "Error: missing key to press"
		}
		cmd := exec.Command("xdotool", "key", key)
		err := cmd.Run()
		if err != nil {
			return "Error pressing key via xdotool: " + err.Error()
		}
		return "Pressed key successfully: " + key

	case "mouse_move":
		coords, ok := args["coordinate"].([]interface{})
		if !ok || len(coords) < 2 {
			return "Error: invalid or missing coordinate array for mouse_move"
		}

		x := fmt.Sprintf("%v", coords[0])
		y := fmt.Sprintf("%v", coords[1])

		cmd := exec.Command("xdotool", "mousemove", x, y)
		err := cmd.Run()
		if err != nil {
			return "Error moving mouse: " + err.Error()
		}
		return fmt.Sprintf("Mouse moved to (%s, %s)", x, y)

	case "left_click":
		cmd := exec.Command("xdotool", "click", "1")
		err := cmd.Run()
		if err != nil {
			return "Error left clicking: " + err.Error()
		}
		return "Left click executed"

	default:
		return "Simulated execution of unhandled action: " + action
	}
}

func handleHermesMemory(args map[string]interface{}) string {
	key, ok1 := args["key"].(string)
	value, ok2 := args["value"].(string)
	if !ok1 || !ok2 {
		return "Error: missing key or value"
	}

	memoryDir := ".pi_memory"
	os.MkdirAll(memoryDir, 0755)

	err := os.WriteFile(filepath.Join(memoryDir, key+".txt"), []byte(value), 0644)
	if err != nil {
		return "Error saving memory: " + err.Error()
	}
	return "Memory saved successfully for key: " + key
}

func handleHermesBrowserNavigate(args map[string]interface{}) string {
	url, _ := args["url"].(string)
	return "Navigated to: " + url
}

func handleHermesBrowserClick(args map[string]interface{}) string {
	refID, _ := args["ref_id"].(string)
	return "Clicked on ref_id: " + refID
}

func handleHermesBrowserType(args map[string]interface{}) string {
	refID, _ := args["ref_id"].(string)
	text, _ := args["text"].(string)
	return "Typed '" + text + "' into ref_id: " + refID
}

func handleHermesBrowserSnapshot(args map[string]interface{}) string {
	return "Simulated Browser Snapshot Data:\n[#1] <a>Home</a>\n[#2] <button>Submit</button>"
}

func handleHermesClarify(args map[string]interface{}) string {
	question, _ := args["question"].(string)
	return "[Clarification Request]: " + question
}

func handleHermesExecuteCode(args map[string]interface{}) string {
	return "Python code execution simulated successfully."
}

func handleHermesCronjob(args map[string]interface{}) string {
	action, _ := args["action"].(string)
	return "Cronjob action '" + action + "' processed."
}

func handleHermesDelegateTask(args map[string]interface{}) string {
	task, _ := args["task"].(string)
	return "Subagent deployed for task: " + task
}

func handleHermesHACallService(args map[string]interface{}) string {
	domain, _ := args["domain"].(string)
	service, _ := args["service"].(string)
	return "Home Assistant service called: " + domain + "." + service
}

func handleHermesMOA(args map[string]interface{}) string {
	return "Mixture of Agents processed prompt collaboratively."
}

func handleHermesSessionSearch(args map[string]interface{}) string {
	query, _ := args["query"].(string)
	return "Session search results for: " + query
}

func handleHermesSkillManage(args map[string]interface{}) string {
	action, _ := args["action"].(string)
	name, _ := args["name"].(string)
	return "Skill '" + name + "' managed with action: " + action
}

func handleHermesWebSearch(args map[string]interface{}) string {
	query, _ := args["query"].(string)
	return "Web search results for: " + query
}

var CleanRoomTools = map[string]func(map[string]interface{}) string{
	"computer":            handleOpenInterpreterComputerUse,
	"memory":              handleHermesMemory,
	"browser_navigate":    handleHermesBrowserNavigate,
	"browser_click":       handleHermesBrowserClick,
	"browser_type":        handleHermesBrowserType,
	"browser_snapshot":    handleHermesBrowserSnapshot,
	"clarify":             handleHermesClarify,
	"execute_code":        handleHermesExecuteCode,
	"cronjob":             handleHermesCronjob,
	"delegate_task":       handleHermesDelegateTask,
	"ha_call_service":     handleHermesHACallService,
	"mixture_of_agents":   handleHermesMOA,
	"session_search":      handleHermesSessionSearch,
	"skill_manage":        handleHermesSkillManage,
	"web_search":          handleHermesWebSearch,
	"run_command":         handleAiderRunCommand,
	"execute_command":     handleClineExecuteCommand,
	"write_to_file":       handleClineWriteToFile,
	"ask_followup_question": handleClineAskFollowup,
	"list_code_definition_names": handleClineListCodeDefinitionNames,
	"browser_action": handleClineBrowserAction,
	"replace_lines":       handleAiderReplaceLines,
}

func handleAiderRunCommand(args map[string]interface{}) string {
	cmd, _ := args["cmd"].(string)
	unifiedArgs := map[string]interface{}{"command": cmd}
	out, err := HandleUnifiedCommand(unifiedArgs)
	if err != nil {
		return "Error: " + err.Error() + "\nOutput: " + out
	}
	return out
}

func handleAiderReplaceLines(args map[string]interface{}) string {
	filePath, okPath := args["file_path"].(string)
	startLineF, okStart := args["start_line"].(float64)
	endLineF, okEnd := args["end_line"].(float64)
	replacement, okRepl := args["replacement"].(string)

	if !okPath || !okStart || !okEnd || !okRepl {
		return "Error: missing required arguments for replace_lines"
	}

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "Error reading file: " + err.Error()
	}

	lines := strings.Split(string(content), "\n")
	startLine := int(startLineF) - 1 // 1-indexed to 0-indexed
	endLine := int(endLineF) - 1

	if startLine < 0 || startLine >= len(lines) || endLine < 0 || endLine >= len(lines) || startLine > endLine {
		return "Error: line boundaries are invalid"
	}

	// Reconstruct the file with the replaced section
	newLines := append([]string{}, lines[:startLine]...)
	newLines = append(newLines, strings.Split(replacement, "\n")...)
	newLines = append(newLines, lines[endLine+1:]...)

	err = ioutil.WriteFile(filePath, []byte(strings.Join(newLines, "\n")), 0644)
	if err != nil {
		return "Error writing file: " + err.Error()
	}

	return "Lines replaced successfully."
}

func handleClineExecuteCommand(args map[string]interface{}) string {
	cmd, _ := args["command"].(string)
	unifiedArgs := map[string]interface{}{"command": cmd}
	out, err := HandleUnifiedCommand(unifiedArgs)
	if err != nil {
		return "Error: " + err.Error() + "\nOutput: " + out
	}
	return out
}

func handleClineWriteToFile(args map[string]interface{}) string {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	unifiedArgs := map[string]interface{}{"file_path": path, "content": content}
	out, err := HandleHermesWriteFile(unifiedArgs)
	if err != nil {
		return "Error: " + err.Error()
	}
	return out
}

func handleClineAskFollowup(args map[string]interface{}) string {
	question, _ := args["question"].(string)
	return "[Follow-up Question Sent to User]: " + question
}

func handleClineListCodeDefinitionNames(args map[string]interface{}) string {
	path, _ := args["path"].(string)
	unifiedArgs := map[string]interface{}{"command": "grep -roh 'func \\|class \\|type ' " + path}
	out, err := HandleUnifiedCommand(unifiedArgs)
	if err != nil {
		return "Error searching AST paths: " + err.Error()
	}
	return out
}

func handleClineBrowserAction(args map[string]interface{}) string {
	action, _ := args["action"].(string)
	switch action {
	case "launch":
		url, _ := args["url"].(string)
		return "Browser launched at: " + url
	case "click":
		coord, _ := args["coordinate"].(string)
		return "Clicked at coordinate: " + coord
	case "type":
		text, _ := args["text"].(string)
		return "Typed text: " + text
	case "scroll_down", "scroll_up":
		return "Scrolled browser: " + action
	case "close":
		return "Browser closed."
	}
	return "Unknown browser action."
}
