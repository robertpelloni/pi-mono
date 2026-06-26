package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/badlogic/pi-mono/pkg/agentsession"
)

func TestSystemStatusEndpoint(t *testing.T) {
	config := agentsession.AgentSessionConfig{}
	server := NewServer("", config)

	req := httptest.NewRequest(http.MethodGet, "/system/status", nil)
	rr := httptest.NewRecorder()

	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK, got %d", rr.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	status, ok := response["status"].(string)
	if !ok || status != "Healthy" {
		t.Errorf("expected status 'Healthy', got %v", status)
	}

	submodules, ok := response["submodules"].([]interface{})
	if !ok || len(submodules) == 0 {
		t.Errorf("expected submodules list, got %v", submodules)
	}

	// Verify we are returning the expected mock data format
	bodyStr := rr.Body.String()
	if !strings.Contains(bodyStr, "assimilated") {
		t.Errorf("expected 'assimilated' inside submodule json, got: %s", bodyStr)
	}
}
