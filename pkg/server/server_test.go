package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/badlogic/pi-mono/pkg/agentsession"
)

func TestHealthEndpoint(t *testing.T) {
	s := NewServer("", agentsession.AgentSessionConfig{})
	req, _ := http.NewRequest("GET", "/api/health", nil)
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK, got %v", rr.Code)
	}

	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %v", resp["status"])
	}
}

func TestChatEndpoint_InvalidMethod(t *testing.T) {
	s := NewServer("", agentsession.AgentSessionConfig{})
	req, _ := http.NewRequest("GET", "/api/chat", nil)
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected method not allowed, got %v", rr.Code)
	}
}

func TestListSessionsEndpoint(t *testing.T) {
	s := NewServer("", agentsession.AgentSessionConfig{})
	req, _ := http.NewRequest("GET", "/api/sessions", nil)
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK, got %v", rr.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if _, ok := resp["sessions"]; !ok {
		t.Errorf("expected sessions list in response")
	}
}
