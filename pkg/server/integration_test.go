package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/badlogic/pi-mono/pkg/agentsession"
)

func TestServer_AssimilatedEndpoints(t *testing.T) {
	s := NewServer("", agentsession.AgentSessionConfig{})

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

		// Expect 500 because no models are registered, but the route should be found
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rr.Code)
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
	})
}
