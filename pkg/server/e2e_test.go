package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/agentsession"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/settings"
)

func TestServer_E2E(t *testing.T) {
	s := NewServer("", agentsession.AgentSessionConfig{})

	t.Run("E2E - Tabby Flow", func(t *testing.T) {
		// Mock a model in the registry to avoid 500
		s.registry.RegisterModel(ai.ModelInfo{
			ID:       "mock-model",
			Provider: "mock",
			API:      "mock",
		})

		payload := map[string]interface{}{
			"segments": map[string]interface{}{
				"prefix": "func main() {",
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/v1/completions", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, req)

		// It still fails in model.Stream because no provider is registered, but it gets further
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rr.Code)
		}
	})

	t.Run("E2E - Warp Flow", func(t *testing.T) {
		payload := map[string]interface{}{
			"type": "RequestCommandOutput",
			"params": map[string]interface{}{
				"command": "hostname",
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/warp/action", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		var resp ai.WarpActionResponse
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp.Status != "success" {
			t.Errorf("expected success status, got %s, error: %s", resp.Status, resp.Error)
		}
	})

	t.Run("E2E - Wave Flow", func(t *testing.T) {
		// Create a file in current test dir
		os.WriteFile("test_wave.txt", []byte("module github.com/badlogic/pi-mono"), 0644)
		defer os.Remove("test_wave.txt")

		payload := map[string]interface{}{
			"type": "readfile",
			"params": map[string]interface{}{
				"path": "test_wave.txt",
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/wave/action", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
		}

		var resp ai.WaveActionResponse
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp.Status != "success" {
			t.Errorf("expected success status, got %s, error: %s", resp.Status, resp.Error)
		}
		if !strings.Contains(resp.Output, "module github.com/badlogic/pi-mono") {
			t.Errorf("unexpected output: %s", resp.Output)
		}
	})
}

func TestServer_ComplexMutations(t *testing.T) {
	s := NewServer("", agentsession.AgentSessionConfig{})

	t.Run("E2E - OpenCode MultiEdit", func(t *testing.T) {
		tmpFile := "e2e_multiedit_test.txt"
		os.WriteFile(tmpFile, []byte("original text"), 0644)
		defer os.Remove(tmpFile)

		h := ai.NewHarness(s.registry)
		args := map[string]interface{}{
			"filePath": tmpFile,
			"edits": []interface{}{
				map[string]interface{}{"oldString": "original", "newString": "modified"},
			},
		}

		resp, err := h.ExecuteTool(context.Background(), "multiedit", args)
		if err != nil {
			t.Errorf("multiedit failed: %v", err)
		}
		if !strings.Contains(resp, "Success") {
			t.Errorf("unexpected response: %s", resp)
		}

		content, _ := os.ReadFile(tmpFile)
		if string(content) != "modified text" {
			t.Errorf("file not updated correctly: %s", string(content))
		}
	})

	t.Run("E2E - RepoMap Generation", func(t *testing.T) {
		h := ai.NewHarness(s.registry)
		args := map[string]interface{}{
			"base_dir": ".",
		}

		resp, err := h.ExecuteTool(context.Background(), "repo_map", args)
		if err != nil {
			t.Errorf("repo_map failed: %v", err)
		}
		if !strings.Contains(resp, "<repo_map>") {
			t.Errorf("invalid repo_map response: %s", resp)
		}
	})
}

func TestServer_LLMToolCoordination(t *testing.T) {
	// 1. Setup server with mock stream
	model := ai.ModelInfo{
		ID: "coord-model",
		Provider: "mock-prov",
		API: ai.Api("mock-api"),
	}

	// Manual NewAgent
	ag := agent.NewAgent(model, nil, mockToolCoordinationStream, agent.AgentLoopConfig{})

	// Create minimal settings manager
	tmpDir, _ := os.MkdirTemp("", "settings-test")
	defer os.RemoveAll(tmpDir)
	sm := settings.Create(".", tmpDir)

	cfg := agentsession.AgentSessionConfig{
		Agent:    ag,
		Settings: sm,
	}
	s := NewServer("", cfg)

	// Register the mock API provider
	apiProv := ai.APIProvider{
		API: ai.Api("mock-api"),
		Stream: mockToolCoordinationStream,
		StreamSimple: mockToolCoordinationStream,
	}
	ai.RegisterAPIProvider(apiProv, "test")

	// Pre-populate the registry
	s.registry.RegisterModel(model)

	// 2. Start Chat
	payload := chatRequest{
		Message: "Please read go.mod",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/chat", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	// 3. Verify interaction (SSE stream contains tool call and final answer)
	output := rr.Body.String()
	if !strings.Contains(output, "toolcall_start") {
		t.Error("expected tool call in response stream")
	}
	if !strings.Contains(output, "I have read the file.") {
		t.Error("expected final LLM answer in response stream")
	}
}
