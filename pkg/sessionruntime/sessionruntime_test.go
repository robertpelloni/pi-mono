package sessionruntime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/badlogic/pi-mono/pkg/ai"
)

func TestCreateAgentSessionRuntime(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sessionruntime_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	agentDir := filepath.Join(tmpDir, ".pi")
	os.MkdirAll(filepath.Join(agentDir, "sessions"), 0755)

	result, err := DefaultCreateRuntime(CreateAgentSessionRuntimeOptions{
		CWD:      tmpDir,
		AgentDir: agentDir,
		Model: &ai.ModelInfo{
			ID:       "test-model",
			Provider: ai.ProviderOpenAI,
			API:      ai.ApiOpenAICompletions,
		},
	})
	if err != nil {
		t.Fatalf("DefaultCreateRuntime failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.AgentSession == nil {
		t.Error("Expected AgentSession to be non-nil")
	}
	if result.Services.CWD != tmpDir {
		t.Errorf("Expected CWD=%s, got %s", tmpDir, result.Services.CWD)
	}
}

func TestAgentSessionRuntime_NewSession(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sessionruntime_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	agentDir := filepath.Join(tmpDir, ".pi")
	os.MkdirAll(filepath.Join(agentDir, "sessions"), 0755)

	runtime, err := CreateAgentSessionRuntime(DefaultCreateRuntime, CreateAgentSessionRuntimeOptions{
		CWD:      tmpDir,
		AgentDir: agentDir,
		Model: &ai.ModelInfo{
			ID:       "test-model",
			Provider: ai.ProviderOpenAI,
			API:      ai.ApiOpenAICompletions,
		},
	})
	if err != nil {
		t.Fatalf("CreateAgentSessionRuntime failed: %v", err)
	}
	if runtime == nil {
		t.Fatal("Expected non-nil runtime")
	}
	if runtime.AgentSession() == nil {
		t.Error("Expected AgentSession to be non-nil")
	}
	if runtime.CWD() != tmpDir {
		t.Errorf("Expected CWD=%s, got %s", tmpDir, runtime.CWD())
	}

	// Test NewSession
	err = runtime.NewSession(nil)
	if err != nil {
		t.Errorf("NewSession failed: %v", err)
	}

	// Test Dispose
	runtime.Dispose()
}

func TestSessionStartReason(t *testing.T) {
	if ReasonStartup != "startup" {
		t.Errorf("Expected ReasonStartup='startup', got %s", ReasonStartup)
	}
	if ReasonReload != "reload" {
		t.Errorf("Expected ReasonReload='reload', got %s", ReasonReload)
	}
	if ReasonNew != "new" {
		t.Errorf("Expected ReasonNew='new', got %s", ReasonNew)
	}
	if ReasonResume != "resume" {
		t.Errorf("Expected ReasonResume='resume', got %s", ReasonResume)
	}
	if ReasonFork != "fork" {
		t.Errorf("Expected ReasonFork='fork', got %s", ReasonFork)
	}
}

func TestDiagnosticTypes(t *testing.T) {
	if DiagInfo != "info" {
		t.Errorf("Expected DiagInfo='info', got %s", DiagInfo)
	}
	if DiagWarning != "warning" {
		t.Errorf("Expected DiagWarning='warning', got %s", DiagWarning)
	}
	if DiagError != "error" {
		t.Errorf("Expected DiagError='error', got %s", DiagError)
	}
}
