package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"strings"

	"github.com/badlogic/pi-mono/pkg/agentsession"
	"github.com/badlogic/pi-mono/pkg/ai"
)

func TestServer_AssimilatedEndpoints(t *testing.T) {
	s := NewServer("", agentsession.AgentSessionConfig{})

	// Register mock provider for integration tests
	apiProv := ai.APIProvider{
		API: ai.Api("mock-api"),
		Stream: mockStreamIntegration,
		StreamSimple: mockStreamIntegration,
	}
	ai.RegisterAPIProvider(apiProv, "test")

	s.Registry().RegisterModel(ai.ModelInfo{
		ID: "test-model",
		Provider: "mock-prov",
		API: ai.Api("mock-api"),
	})

	t.Run("Tabby Completion Route", func(t *testing.T) {
		payload := map[string]interface{}{
			"segments": map[string]interface{}{
				"prefix": "test",
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/v1/completions", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		s.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
		}

		var resp ai.TabbyCompletionResponse
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if len(resp.Choices) == 0 || resp.Choices[0].Text != "mock-response" {
			t.Errorf("unexpected tabby response text: %v", resp.Choices)
		}
	})

	t.Run("Warp Action Route", func(t *testing.T) {
		payload := map[string]interface{}{
			"type": "RequestCommandOutput",
			"params": map[string]interface{}{
				"command": "echo test",
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
		if resp.Status != "success" || !strings.Contains(resp.Output, "test") {
			t.Errorf("unexpected warp response: %v", resp)
		}
	})

	t.Run("Wave Action Route", func(t *testing.T) {
		payload := map[string]interface{}{
			"type": "term",
			"params": map[string]interface{}{
				"command": "echo wave",
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/wave/action", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		s.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		var resp ai.WaveActionResponse
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp.Status != "success" || !strings.Contains(resp.Output, "wave") {
			t.Errorf("unexpected wave response: %v", resp)
		}
	})

	t.Run("Hyper Theme Sync Route", func(t *testing.T) {
		payload := map[string]interface{}{
			"config": `{"config": {"colors": {"black": "#000"}}}`,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/hyper/theme", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		s.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		var resp map[string]string
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if !strings.Contains(resp["message"], "initialized") {
			t.Errorf("unexpected hyper response: %v", resp)
		}
	})
}
