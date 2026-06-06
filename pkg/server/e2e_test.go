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

func TestServer_E2E(t *testing.T) {
	s := NewServer("", agentsession.AgentSessionConfig{})

	t.Run("E2E - Tabby Flow", func(t *testing.T) {
		payload := map[string]interface{}{
			"segments": map[string]interface{}{
				"prefix": "func main() {",
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/v1/completions", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, req)

		// Expect 500 (no models) but verifies the entire route stack
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
		payload := map[string]interface{}{
			"type": "readfile",
			"params": map[string]interface{}{
				"path": "../../go.mod",
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
		if resp.Status != "success" {
			t.Errorf("expected success status, got %s, error: %s", resp.Status, resp.Error)
		}
		if !strings.Contains(resp.Output, "module github.com/badlogic/pi-mono") {
			t.Errorf("unexpected output: %s", resp.Output)
		}
	})
}
