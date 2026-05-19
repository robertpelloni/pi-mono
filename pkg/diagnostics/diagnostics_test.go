package diagnostics

import (
	"strings"
	"testing"
)

func TestFormatDiagnostics_Empty(t *testing.T) {
	result := FormatDiagnostics(nil)
	if result != "" {
		t.Errorf("Expected empty string for nil diagnostics, got %q", result)
	}
}

func TestFormatDiagnostics_Single(t *testing.T) {
	diags := []ResourceDiagnostic{
		{Path: "/tmp/test.txt", Message: "file not found", Source: "skill"},
	}
	result := FormatDiagnostics(diags)
	if !strings.Contains(result, "Warnings") {
		t.Error("Expected 'Warnings' header")
	}
	if !strings.Contains(result, "file not found") {
		t.Error("Expected diagnostic message")
	}
	if !strings.Contains(result, "[skill]") {
		t.Error("Expected source prefix")
	}
}

func TestFormatDiagnostics_Multiple(t *testing.T) {
	diags := []ResourceDiagnostic{
		{Path: "/a", Message: "error 1", Source: "skill"},
		{Message: "error 2"},
	}
	result := FormatDiagnostics(diags)
	if !strings.Contains(result, "error 1") || !strings.Contains(result, "error 2") {
		t.Error("Expected both diagnostics in output")
	}
}

func TestResourceDiagnostic_Fields(t *testing.T) {
	d := ResourceDiagnostic{Path: "/tmp/x", Message: "test", Source: "prompt"}
	if d.Path != "/tmp/x" {
		t.Error("Path mismatch")
	}
	if d.Source != "prompt" {
		t.Error("Source mismatch")
	}
}
