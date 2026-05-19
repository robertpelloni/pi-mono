package rpctypes

import (
	"encoding/json"
	"testing"
)

func TestRpcError(t *testing.T) {
	resp := RpcError("test-1", "prompt", "something went wrong")
	if resp.ID != "test-1" {
		t.Errorf("Expected ID 'test-1', got %q", resp.ID)
	}
	if resp.Command != "prompt" {
		t.Errorf("Expected command 'prompt', got %q", resp.Command)
	}
	if resp.Error != "something went wrong" {
		t.Errorf("Expected error 'something went wrong', got %q", resp.Error)
	}
	if resp.Success {
		t.Error("Expected Success=false for error response")
	}
}

func TestRpcSuccess(t *testing.T) {
	resp := RpcSuccess("test-2", "compact", map[string]interface{}{"tokens": 1000})
	if resp.ID != "test-2" {
		t.Errorf("Expected ID 'test-2', got %q", resp.ID)
	}
	if resp.Command != "compact" {
		t.Errorf("Expected command 'compact', got %q", resp.Command)
	}
	if resp.Error != "" {
		t.Errorf("Expected empty error for success, got %q", resp.Error)
	}
	if !resp.Success {
		t.Error("Expected Success=true for success response")
	}
	if resp.Data == nil {
		t.Error("Expected non-nil data for success")
	}
}

func TestRpcError_JSONSerialization(t *testing.T) {
	resp := RpcError("test-3", "prompt", "test error")
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "" {
		t.Error("Expected non-empty JSON output")
	}
}

func TestRpcSuccess_JSONSerialization(t *testing.T) {
	resp := RpcSuccess("test-4", "list", []string{"a", "b"})
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "" {
		t.Error("Expected non-empty JSON output")
	}
}

func TestRpcResponse_Fields(t *testing.T) {
	resp := RpcResponse{ID: "test-5", Command: "test", Type: "response"}
	if resp.ID != "test-5" {
		t.Error("ID mismatch")
	}
	if resp.Command != "test" {
		t.Error("Command mismatch")
	}
	if resp.Type != "response" {
		t.Error("Type mismatch")
	}
}
