package resolveconfig

import (
	"os"
	"testing"
)

func TestResolveConfigValue_Literal(t *testing.T) {
	result := ResolveConfigValue("hello")
	if result != "hello" {
		t.Errorf("Expected 'hello', got %q", result)
	}
}

func TestResolveConfigValue_EnvVar(t *testing.T) {
	os.Setenv("PI_TEST_RESOLVE", "resolved_value")
	defer os.Unsetenv("PI_TEST_RESOLVE")

	result := ResolveConfigValue("PI_TEST_RESOLVE")
	if result != "resolved_value" {
		t.Errorf("Expected 'resolved_value', got %q", result)
	}
}

func TestResolveConfigValue_EmptyEnvVar(t *testing.T) {
	os.Unsetenv("PI_TEST_EMPTY_RESOLVE")
	result := ResolveConfigValue("PI_TEST_EMPTY_RESOLVE")
	// Empty env var falls back to literal
	if result != "PI_TEST_EMPTY_RESOLVE" {
		t.Errorf("Expected literal name for empty env var, got %q", result)
	}
}

func TestResolveConfigValue_Command(t *testing.T) {
	ClearConfigValueCache()
	result := ResolveConfigValue("!echo hello")
	if result != "hello" {
		t.Errorf("Expected 'hello', got %q", result)
	}
}

func TestResolveConfigValue_CommandCached(t *testing.T) {
	ClearConfigValueCache()
	result1 := ResolveConfigValue("!echo cached")
	result2 := ResolveConfigValue("!echo cached")
	if result1 != result2 {
		t.Errorf("Cache should return same value: %q vs %q", result1, result2)
	}
}

func TestResolveConfigValueUncached_Literal(t *testing.T) {
	result := ResolveConfigValueUncached("direct")
	if result != "direct" {
		t.Errorf("Expected 'direct', got %q", result)
	}
}

func TestResolveHeaders(t *testing.T) {
	headers := map[string]string{
		"X-Custom": "value",
	}
	result := ResolveHeaders(headers)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result["X-Custom"] != "value" {
		t.Errorf("Expected 'value', got %q", result["X-Custom"])
	}
}

func TestResolveHeaders_Nil(t *testing.T) {
	result := ResolveHeaders(nil)
	if result != nil {
		t.Error("Expected nil for nil input")
	}
}

func TestResolveHeaders_Empty(t *testing.T) {
	result := ResolveHeaders(map[string]string{})
	if result != nil {
		t.Error("Expected nil for empty input")
	}
}

func TestClearConfigValueCache(t *testing.T) {
	// Just test it doesn't panic
	ClearConfigValueCache()
}
