package plugins

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestLinterPlugin_Lifecycle(t *testing.T) {
	// Skip test if gofmt is not installed in the testing environment
	if _, err := exec.LookPath("gofmt"); err != nil {
		t.Skip("skipping linter plugin test because 'gofmt' is not installed")
	}

	plugin := NewLinterPlugin()

	// Test 1: Validate
	if err := plugin.Validate(); err != nil {
		t.Fatalf("expected validation to pass, got error: %v", err)
	}

	// Test 2: Tools Registration
	tools := plugin.GetTools()
	if _, ok := tools["run_linter"]; !ok {
		t.Fatalf("expected 'run_linter' to be present in tool schemas")
	}

	// Test 3: Execute HandleToolCall
	tmpFile, err := os.CreateTemp("", "test_format_*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// Write unformatted Go code
	unformatted := "package main\n\nfunc   main()   {   }\n"
	tmpFile.Write([]byte(unformatted))
	tmpFile.Close()

	ctx := context.Background()
	args := map[string]interface{}{"file_path": tmpFile.Name()}

	resp, err := plugin.HandleToolCall(ctx, "run_linter", args)
	if err != nil {
		t.Fatalf("expected successful linting, got error: %v", err)
	}

	if !strings.Contains(resp, "Successfully formatted") {
		t.Errorf("expected success message, got: %s", resp)
	}

	// Verify the file was actually formatted
	content, _ := os.ReadFile(tmpFile.Name())
	if strings.Contains(string(content), "func   main()") {
		t.Errorf("expected file to be formatted, but spacing remained: %s", string(content))
	}
}

func TestPluginManager_ValidationFailure(t *testing.T) {
	// We dynamically swap out the lookpath for the test or create a MockPlugin
	// Since we are unit testing the manager, a mock is safer.

	pm := &PluginManager{
		plugins:      make(map[string]Plugin),
		toolToPlugin: make(map[string]string),
	}

	err := pm.RegisterPlugin(&mockFailingPlugin{})
	if err == nil {
		t.Fatal("expected manager to return an error when plugin validation fails")
	}
	if !strings.Contains(err.Error(), "failed validation") {
		t.Errorf("unexpected error message: %v", err)
	}
}

type mockFailingPlugin struct{}

func (m *mockFailingPlugin) Name() string { return "failing_mock" }
func (m *mockFailingPlugin) Init(ctx context.Context) error { return nil }
func (m *mockFailingPlugin) GetTools() map[string]interface{} { return nil }
func (m *mockFailingPlugin) HandleToolCall(ctx context.Context, toolName string, args map[string]interface{}) (string, error) { return "", nil }

// Validate simulates a failure for the test.
func (m *mockFailingPlugin) Validate() error {
	return os.ErrNotExist
}
