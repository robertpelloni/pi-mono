package cleanroomtools

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// --- Handler Functions ---

// HandleHermesPatch applies a find/replace patch to a file.
func HandleHermesPatch(args map[string]interface{}) string {
	filePath, _ := args["file_path"].(string)
	find, _ := args["find"].(string)
	replace, _ := args["replace"].(string)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err)
	}

	if !strings.Contains(string(content), find) {
		return "Error: Target string not found in file."
	}

	newContent := strings.Replace(string(content), find, replace, 1)
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return fmt.Sprintf("Error writing file: %v", err)
	}
	return "Patch applied successfully"
}

// HandleHermesWriteFile writes content to a file.
func HandleHermesWriteFile(args map[string]interface{}) string {
	filePath, _ := args["file_path"].(string)
	content, _ := args["content"].(string)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Sprintf("Error writing file: %v", err)
	}
	return "File written successfully"
}

// HandleHermesTerminal executes a command.
func HandleHermesTerminal(args map[string]interface{}) string {
	command, _ := args["command"].(string)
	background, _ := args["background"].(bool)
	_ = background // TODO: implement background mode

	out, err := exec.Command("sh", "-c", command).CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error executing command: %v\n%s", err, string(out))
	}
	output := strings.TrimSpace(string(out))
	if output == "" {
		return "Command executed successfully with no output."
	}
	return output
}

// HandleHermesMemory saves a key-value pair to persistent memory.
func HandleHermesMemory(args map[string]interface{}) string {
	key, _ := args["key"].(string)
	value, _ := args["value"].(string)

	memoryDir := ".pi_memory"
	if err := os.MkdirAll(memoryDir, 0755); err != nil {
		return fmt.Sprintf("Error creating memory directory: %v", err)
	}
	if err := os.WriteFile(memoryDir+"/"+key+".txt", []byte(value), 0644); err != nil {
		return fmt.Sprintf("Error saving memory: %v", err)
	}
	return fmt.Sprintf("Memory saved successfully for key: %s", key)
}

// HandleClineExecuteCommand executes a CLI command.
func HandleClineExecuteCommand(args map[string]interface{}) string {
	command, _ := args["command"].(string)
	out, err := exec.Command("sh", "-c", command).CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error executing command: %v\n%s", err, string(out))
	}
	output := strings.TrimSpace(string(out))
	if output == "" {
		return "Command executed successfully with no output."
	}
	return output
}

// HandleClineWriteToFile writes content to a file.
func HandleClineWriteToFile(args map[string]interface{}) string {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Sprintf("Error writing file: %v", err)
	}
	return "File written successfully"
}

// HandleClineAskFollowup returns a follow-up question.
func HandleClineAskFollowup(args map[string]interface{}) string {
	question, _ := args["question"].(string)
	return fmt.Sprintf("[Follow-up Question Sent to User]: %s", question)
}

// HandleClineListCodeDefinitionNames lists code definitions.
func HandleClineListCodeDefinitionNames(args map[string]interface{}) string {
	path, _ := args["path"].(string)
	out, err := exec.Command("grep", "-roh", `func \|class \|type `, path).CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error executing command: %v", err)
	}
	output := strings.TrimSpace(string(out))
	if output == "" {
		return "No definitions found."
	}
	return output
}

// HandleClineBrowserAction simulates browser actions.
func HandleClineBrowserAction(args map[string]interface{}) string {
	action, _ := args["action"].(string)
	url, _ := args["url"].(string)
	coordinate, _ := args["coordinate"].(string)
	text, _ := args["text"].(string)

	switch action {
	case "launch":
		return fmt.Sprintf("Browser launched at: %s", url)
	case "click":
		return fmt.Sprintf("Clicked at coordinate: %s", coordinate)
	case "type":
		return fmt.Sprintf("Typed text: %s", text)
	case "scroll_down", "scroll_up":
		return fmt.Sprintf("Scrolled browser: %s", action)
	case "close":
		return "Browser closed."
	default:
		return "Unknown browser action."
	}
}

// HandleOpenInterpreterComputerUse simulates computer actions.
func HandleOpenInterpreterComputerUse(args map[string]interface{}) string {
	action, _ := args["action"].(string)
	return fmt.Sprintf("Simulated computer action: %s executed.", action)
}

// HandleAiderRunCommand executes a command.
func HandleAiderRunCommand(args map[string]interface{}) string {
	cmd, _ := args["cmd"].(string)
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error executing command: %v\n%s", err, string(out))
	}
	output := strings.TrimSpace(string(out))
	if output == "" {
		return "Command executed successfully with no output."
	}
	return output
}

// HandleAiderReplaceLines replaces lines in a file (stub).
func HandleAiderReplaceLines(args map[string]interface{}) string {
	return "Simulated replacement logic in legacy TS layer."
}
