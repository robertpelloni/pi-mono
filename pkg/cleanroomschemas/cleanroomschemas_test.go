package cleanroomschemas

import (
	"testing"
)

func TestAllCleanRoomToolSchemas(t *testing.T) {
	schemas := AllCleanRoomToolSchemas()
	if len(schemas) == 0 {
		t.Error("Expected non-empty schemas map")
	}

	// Check key schemas exist
	expectedTools := []string{
		"claude_code_read",
		"claude_code_bash",
		"claude_code_grep",
		"hermes_patch",
		"hermes_terminal",
		"hermes_write_file",
		"cline_execute_command",
		"cline_write_to_file",
		"cline_browser_action",
		"open_interpreter_computer",
	}

	for _, name := range expectedTools {
		if _, ok := schemas[name]; !ok {
			t.Errorf("Expected schema for %s", name)
		}
	}
}

func TestClaudeCodeReadSchema(t *testing.T) {
	schema := ClaudeCodeReadSchema
	if schema.Type != "object" {
		t.Errorf("Expected type 'object', got %s", schema.Type)
	}
	if _, ok := schema.Properties["file_path"]; !ok {
		t.Error("Expected file_path property")
	}
	if len(schema.Required) == 0 {
		t.Error("Expected at least one required field")
	}
}

func TestClineBrowserActionSchema(t *testing.T) {
	schema := ClineBrowserActionSchema
	if _, ok := schema.Properties["action"]; !ok {
		t.Error("Expected action property")
	}
	if _, ok := schema.Properties["url"]; !ok {
		t.Error("Expected url property")
	}
}

func TestHermesPatchSchema(t *testing.T) {
	schema := HermesPatchSchema
	expectedProps := []string{"file_path", "find", "replace"}
	for _, prop := range expectedProps {
		if _, ok := schema.Properties[prop]; !ok {
			t.Errorf("Expected %s property", prop)
		}
	}
}
