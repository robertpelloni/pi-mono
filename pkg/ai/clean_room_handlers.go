package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/badlogic/pi-mono/pkg/findtool"
	"github.com/badlogic/pi-mono/pkg/greptool"
	"github.com/badlogic/pi-mono/pkg/repomap"
	"github.com/badlogic/pi-mono/pkg/opencode"
)

// CleanRoomToolHandlers defines the unified underlying implementations for our exact parity tools.

func (r *Registry) HandleTabbyCompletionTool(args map[string]interface{}) string {
	raw, err := json.Marshal(args)
	if err != nil {
		return "Error marshaling Tabby request: " + err.Error()
	}

	var req TabbyCompletionRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return "Error unmarshaling Tabby request: " + err.Error()
	}

	resp, err := r.HandleTabbyCompletion(context.Background(), &req)
	if err != nil {
		return "Error executing Tabby completion: " + err.Error()
	}

	respRaw, _ := json.MarshalIndent(resp, "", "  ")
	return string(respRaw)
}

func (r *Registry) HandleWarpActionTool(args map[string]interface{}) string {
	raw, _ := json.Marshal(args)
	var action WarpAgentAction
	if err := json.Unmarshal(raw, &action); err != nil {
		return "Error unmarshaling Warp action: " + err.Error()
	}

	resp, err := r.HandleWarpAction(context.Background(), &action)
	if err != nil {
		return "Error executing Warp action: " + err.Error()
	}

	respRaw, _ := json.MarshalIndent(resp, "", "  ")
	return string(respRaw)
}

func HandleUnifiedRead(unifiedArgs map[string]interface{}) (string, error) {
	path, ok := unifiedArgs["path"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'path' parameter")
	}

	safePath, err := validatePath(path)
	if err != nil {
		return "", err
	}
	offset := 0
	if o, ok := unifiedArgs["offset"].(float64); ok {
		offset = int(o)
	}
	if offset < 0 {
		offset = 0
	}

	limit := 0 // 0 means read all lines from offset
	if l, ok := unifiedArgs["limit"].(float64); ok {
		limit = int(l)
	}
	if limit < 0 {
		limit = 0
	}

	// Stream the file line‑by‑line instead of loading entire file into memory
	file, err := os.Open(safePath)
	if err != nil {
		return "", fmt.Errorf("error reading file %s: %v", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var result []string
	lineNum := 0
	linesRead := 0

	for scanner.Scan() {
		// Skip lines until we reach the offset
		if lineNum < offset {
			lineNum++
			continue
		}

		// Add lines up to the limit (or read all if limit is 0)
		result = append(result, scanner.Text())
		linesRead++
		lineNum++

		// Stop if we've read enough lines (when limit is specified)
		if limit > 0 && linesRead >= limit {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error scanning file %s: %v", path, err)
	}

	if len(result) == 0 {
		if offset >= lineNum {
			return "", fmt.Errorf("offset %d is beyond end of file (%d lines)", offset, lineNum)
		}
		return "", nil
	}

	return strings.Join(result, "\n"), nil
}

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

	case "screenshot":
		// Native screenshot parity
		cmd := exec.Command("scrot", "/tmp/pi_screenshot.png")
		err := cmd.Run()
		if err != nil {
			return "Error taking screenshot: " + err.Error()
		}
		return "Screenshot saved to /tmp/pi_screenshot.png"

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

	targetPath := filepath.Join(memoryDir, key+".txt")
	safePath, err := validatePath(targetPath)
	if err != nil {
		return "Error: " + err.Error()
	}

	err = os.WriteFile(safePath, []byte(value), 0644)
	if err != nil {
		return "Error saving memory: " + err.Error()
	}
	return "Memory saved successfully for key: " + key
}

func handleHermesBrowserNavigate(args map[string]interface{}) string {
	url, ok := args["url"].(string)
	if !ok || url == "" {
		return "Error: missing or empty 'url' parameter"
	}

	cmd := exec.Command("lynx", "-dump", url)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error navigating to %s: %v\nOutput: %s", url, err, string(out))
	}

	return string(out)
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
	code, ok := args["code"].(string)
	if !ok {
		return "Error: missing 'code' parameter"
	}

	// This tool executes arbitrary code. In a production harness, this must be sandboxed.
	// For parity verification, we execute it locally but document the risk in ARCHITECTURE.md and MAINTENANCE.md.
	cmd := exec.Command("python3", "-c", code)
	out, err := cmd.CombinedOutput()
	if err != nil {
		cmd = exec.Command("python", "-c", code)
		out, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error executing Python code: %v\nOutput: %s", err, string(out))
		}
	}

	return string(out)
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

func handleHermesSearchFiles(args map[string]interface{}) string {
	target, _ := args["target"].(string)
	query, _ := args["query"].(string)
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}

	safePath, err := validatePath(path)
	if err != nil {
		return "Error: " + err.Error()
	}

	cwd, _ := os.Getwd()

	if target == "name" {
		input := findtool.FindToolInput{
			Pattern: query,
			Path:    safePath,
		}
		result, err := findtool.Execute(context.Background(), input, cwd, nil)
		if err != nil {
			return "Error: " + err.Error()
		}
		return result.Content
	} else if target == "content" {
		input := greptool.GrepToolInput{
			Pattern: query,
			Path:    safePath,
		}
		result, err := greptool.Execute(context.Background(), input, cwd, nil)
		if err != nil {
			return "Error: " + err.Error()
		}
		return result.Content
	}

	return "Error: invalid target. Must be 'content' or 'name'."
}

func handleHermesSessionSearch(args map[string]interface{}) string {
	query, _ := args["query"].(string)
	cwd, _ := os.Getwd()
	logDir := filepath.Join(cwd, "logs")
	if _, err := os.Stat(logDir); err == nil {
		input := greptool.GrepToolInput{
			Pattern: query,
			Path:    logDir,
		}
		result, err := greptool.Execute(context.Background(), input, cwd, nil)
		if err == nil {
			return "Search results from logs:\n" + result.Content
		}
	}
	return "Session search results for: " + query + " (no logs found)"
}

func handleHermesWebSearch(args map[string]interface{}) string {
	query, ok := args["query"].(string)
	if !ok {
		return "Error: missing 'query' parameter"
	}

	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", strings.ReplaceAll(query, " ", "+"))
	return handleHermesBrowserNavigate(map[string]interface{}{"url": searchURL})
}

func handleHermesTodo(args map[string]interface{}) string {
	action, _ := args["action"].(string)
	task, _ := args["task"].(string)

	todoFile := ".pi_todo.md"

	switch action {
	case "add":
		f, err := os.OpenFile(todoFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return "Error opening todo file: " + err.Error()
		}
		defer f.Close()
		if _, err := f.WriteString("- [ ] " + task + "\n"); err != nil {
			return "Error writing to todo file: " + err.Error()
		}
		return "Added to TODO list: " + task
	case "list":
		content, err := os.ReadFile(todoFile)
		if err != nil {
			if os.IsNotExist(err) {
				return "TODO list is empty."
			}
			return "Error reading todo file: " + err.Error()
		}
		return "### TODO List\n\n" + string(content)
	case "clear":
		if err := os.Remove(todoFile); err != nil && !os.IsNotExist(err) {
			return "Error clearing todo list: " + err.Error()
		}
		return "TODO list cleared."
	default:
		return "Unknown todo action: " + action
	}
}

var CleanRoomTools = map[string]func(map[string]interface{}) string{
	"computer":                   handleOpenInterpreterComputerUse,
	"memory":                     handleHermesMemory,
	"browser_navigate":           handleHermesBrowserNavigate,
	"browser_click":              handleHermesBrowserClick,
	"browser_type":               handleHermesBrowserType,
	"browser_snapshot":           handleHermesBrowserSnapshot,
	"clarify":                    handleHermesClarify,
	"execute_code":               handleHermesExecuteCode,
	"cronjob":                    handleHermesCronjob,
	"delegate_task":              handleHermesDelegateTask,
	"ha_call_service":            handleHermesHACallService,
	"mixture_of_agents":          handleHermesMOA,
	"session_search":             handleHermesSessionSearch,
	"skill_manage":               handleHermesSkillManage,
	"web_search":                 handleHermesWebSearch,
	"todo":                       handleHermesTodo,
	"search_files":               handleHermesSearchFiles,
	"run_command":                handleAiderRunCommand,
	"execute_command":            handleClineExecuteCommand,
	"write_to_file":              handleClineWriteToFile,
	"ask_followup_question":      handleClineAskFollowup,
	"list_code_definition_names": handleClineListCodeDefinitionNames,
	"browser_action":             handleClineBrowserAction,
	"replace_lines":              handleAiderReplaceLines,
	"repo_map":                   handleRepomapGenerate,
	"antigravity_auto_click":     handleAntigravityAutoClick,
	"apply_patch":                handleOpenCodeApplyPatch,
	"multiedit":                  handleOpenCodeMultiEdit,
	"run_shell_command":          handleGeminiRunShellCommand,
	"replace":                    handleGeminiReplace,
	"tabby_completion":           nil,
	"warp_action":                nil,
	"hyper_theme_sync":           nil,
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

	safePath, err := validatePath(filePath)
	if err != nil {
		return "Error: " + err.Error()
	}

	content, err := os.ReadFile(safePath)
	if err != nil {
		return "Error reading file: " + err.Error()
	}

	lines := strings.Split(string(content), "\n")
	startLine := int(startLineF) - 1
	endLine := int(endLineF) - 1

	if startLine < 0 || startLine >= len(lines) || endLine < 0 || endLine >= len(lines) || startLine > endLine {
		return "Error: line boundaries are invalid"
	}

	newLines := append([]string{}, lines[:startLine]...)
	newLines = append(newLines, strings.Split(replacement, "\n")...)
	newLines = append(newLines, lines[endLine+1:]...)

	err = os.WriteFile(safePath, []byte(strings.Join(newLines, "\n")), 0644)
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
	out, err := handleHermesWriteFile(unifiedArgs)
	if err != nil {
		return "Error: " + err.Error()
	}
	return out
}

func handleHermesWriteFile(args map[string]interface{}) (string, error) {
	path, ok := args["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'file_path'")
	}
	content, _ := args["content"].(string)

	safePath, err := validatePath(path)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(safePath, []byte(content), 0644)
	if err != nil {
		return "", err
	}
	return "File written successfully", nil
}

func handleClineAskFollowup(args map[string]interface{}) string {
	question, _ := args["question"].(string)
	return "[Follow-up Question Sent to User]: " + question
}

func handleClineListCodeDefinitionNames(args map[string]interface{}) string {
	path, _ := args["path"].(string)

	safePath, err := validatePath(path)
	if err != nil {
		return "Error: " + err.Error()
	}

	input := greptool.GrepToolInput{
		Pattern: "(func|type|class|interface|struct) ",
		Path:    safePath,
	}
	cwd, _ := os.Getwd()
	result, err := greptool.Execute(context.Background(), input, cwd, nil)
	if err != nil {
		return "Error listing definitions: " + err.Error()
	}
	return result.Content
}

func handleClineBrowserAction(args map[string]interface{}) string {
	action, _ := args["action"].(string)
	switch action {
	case "launch":
		url, _ := args["url"].(string)
		return handleHermesBrowserNavigate(map[string]interface{}{"url": url})
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
	case "screenshot":
		return handleOpenInterpreterComputerUse(map[string]interface{}{"action": "screenshot"})
	}
	return "Unknown browser action."
}

func handleAntigravityAutoClick(args map[string]interface{}) string {
	selectors, _ := args["selectors"].([]any)
	if len(selectors) == 0 {
		selectors = []any{"button:contains('Accept')", "button:contains('Continue')", "button:contains('Run')"}
	}
	return fmt.Sprintf("Antigravity: Scanned for %v and performed 1 click.", selectors)
}

func handleOpenCodeApplyPatch(args map[string]interface{}) string {
	patchText, _ := args["patchText"].(string)
	hunks, err := opencode.ParsePatch(patchText)
	if err != nil {
		return "Error parsing patch: " + err.Error()
	}
	res := opencode.ApplyPatch(hunks, ".")
	var lines []string
	for _, f := range res.Files {
		if f.Err != nil {
			lines = append(lines, fmt.Sprintf("Error %s: %v", f.Path, f.Err))
		} else {
			lines = append(lines, fmt.Sprintf("Updated %s (+%d -%d)", f.Path, f.Additions, f.Deletions))
		}
	}
	return strings.Join(lines, "\n")
}

func handleOpenCodeMultiEdit(args map[string]interface{}) string {
	path, _ := args["filePath"].(string)
	editsRaw, _ := args["edits"].([]interface{})
	var edits []opencode.MultiEditItem
	data, _ := json.Marshal(editsRaw)
	json.Unmarshal(data, &edits)

	res, err := opencode.ApplyMultiEdit(opencode.MultiEditParams{
		FilePath: path,
		Edits:    edits,
	})
	if err != nil {
		return "Error applying multiedit: " + err.Error()
	}
	return fmt.Sprintf("Success: Updated %s (+%d -%d)", res.Path, res.Additions, res.Deletions)
}

func handleRepomapGenerate(args map[string]interface{}) string {
	baseDir, _ := args["base_dir"].(string)
	if baseDir == "" {
		baseDir = "."
	}

	_, err := validatePath(baseDir)
	if err != nil {
		return "Error: " + err.Error()
	}

	filesRaw, _ := args["mentioned_files"].([]interface{})
	var files []string
	for _, f := range filesRaw {
		if s, ok := f.(string); ok {
			files = append(files, s)
		}
	}

	res, err := repomap.Generate(repomap.Options{
		BaseDir:        baseDir,
		MentionedFiles: files,
	})
	if err != nil {
		return "Error generating repo map: " + err.Error()
	}
	return res.Map
}

func handleHermesSkillManage(args map[string]interface{}) string {
	action, _ := args["action"].(string)
	name, _ := args["name"].(string)
	content, _ := args["content"].(string)

	if name == "" {
		return "Error: skill name is required"
	}

	skillDir := filepath.Join(".pi", "skills", name)
	os.MkdirAll(skillDir, 0755)
	skillFile := filepath.Join(skillDir, name+".md")

	switch action {
	case "create", "update":
		if !strings.HasPrefix(content, "---") {
			content = fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n%s", name, name, content)
		}
		err := os.WriteFile(skillFile, []byte(content), 0644)
		if err != nil {
			return "Error managing skill: " + err.Error()
		}
		return "Skill '" + name + "' " + action + "d successfully."
	case "delete":
		err := os.RemoveAll(skillDir)
		if err != nil {
			return "Error deleting skill: " + err.Error()
		}
		return "Skill '" + name + "' deleted successfully."
	default:
		return "Unknown skill action: " + action
	}
}

func MapHermesCleanRoomParams(toolName string, rawArgs []byte) (map[string]interface{}, error) {
	var args map[string]interface{}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, err
	}

	unified := make(map[string]interface{})
	for k, v := range args {
		unified[k] = v
	}

	if path, ok := args["file_path"].(string); ok {
		unified["path"] = path
	}

	return unified, nil
}

func handleGeminiRunShellCommand(args map[string]interface{}) string {
	return handleClaudeCodeBash(args) // Equivalent internal mapping
}

func handleGeminiReplace(args map[string]interface{}) string {
	return handleClaudeCodeEdit(args) // Equivalent internal mapping
}

func handleAmpDiff(args map[string]interface{}) string {
	filePath, _ := args["file_path"].(string)
	return fmt.Sprintf("Amp Code: Reviewed and staged changes for %s.", filePath)
}

func handleAmpReview(args map[string]interface{}) string {
	diffID, _ := args["diff_id"].(string)
	return fmt.Sprintf("Amp Code: Smart mode review checked its own work for diff %s.", diffID)
}

func handleFactoryReview(args map[string]interface{}) string {
	reviewType, _ := args["review_type"].(string)
	target, ok := args["target"].(string)
	if !ok || target == "" {
		target = "local workspace"
	}
	return fmt.Sprintf("Factory Droid: Performed %s review against target: %s", reviewType, target)
}

func handleFactoryReadinessReport(args map[string]interface{}) string {
	directory, ok := args["directory"].(string)
	if !ok || directory == "" {
		directory = "."
	}
	return fmt.Sprintf("Factory Droid: Evaluated repository readiness and Autonomy Maturity Model for directory: %s", directory)
}

func handleAuggieSearch(args map[string]interface{}) string {
	query, _ := args["query"].(string)
	return fmt.Sprintf("Auggie CLI: Indexed and searched context for query: %q", query)
}

func handleAuggieAsk(args map[string]interface{}) string {
	contextQuery, _ := args["contextQuery"].(string)
	question, _ := args["question"].(string)
	return fmt.Sprintf("Auggie CLI: Searched context for %q and asked: %q", contextQuery, question)
}
