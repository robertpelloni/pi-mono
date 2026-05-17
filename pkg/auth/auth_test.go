package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewAuthStorage_NoFile(t *testing.T) {
	a := NewAuthStorage("/nonexistent/path/auth.json")
	if a == nil {
		t.Fatal("Expected non-nil AuthStorage")
	}
}

func TestInMemoryAuthStorage(t *testing.T) {
	data := AuthStorageData{
		"openai": {Type: CredentialTypeAPIKey, Key: "sk-test123"},
	}
	a := InMemoryAuthStorage(data)
	if a == nil {
		t.Fatal("Expected non-nil AuthStorage")
	}

	cred := a.Get("openai")
	if cred == nil {
		t.Fatal("Expected credential for openai")
	}
	if cred.Key != "sk-test123" {
		t.Errorf("Expected key sk-test123, got %s", cred.Key)
	}
}

func TestAuthStorage_SetGet(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "auth_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	authPath := filepath.Join(tmpDir, "auth.json")
	a := NewAuthStorage(authPath)

	cred := AuthCredential{Type: CredentialTypeAPIKey, Key: "sk-test"}
	err = a.Set("openai", cred)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Reload and verify
	a2 := NewAuthStorage(authPath)
	got := a2.Get("openai")
	if got == nil {
		t.Fatal("Expected credential after reload")
	}
	if got.Key != "sk-test" {
		t.Errorf("Expected key sk-test, got %s", got.Key)
	}
}

func TestAuthStorage_Remove(t *testing.T) {
	a := InMemoryAuthStorage(AuthStorageData{
		"openai": {Type: CredentialTypeAPIKey, Key: "sk-test"},
	})

	a.Remove("openai")
	if a.Has("openai") {
		t.Error("Expected openai to be removed")
	}
}

func TestAuthStorage_RuntimeOverride(t *testing.T) {
	a := InMemoryAuthStorage(AuthStorageData{
		"openai": {Type: CredentialTypeAPIKey, Key: "sk-from-file"},
	})

	a.SetRuntimeAPIKey("openai", "sk-from-cli")

	key := a.GetAPIKey("openai")
	if key != "sk-from-cli" {
		t.Errorf("Expected runtime override key, got %s", key)
	}
}

func TestAuthStorage_EnvVar(t *testing.T) {
	a := InMemoryAuthStorage(nil)

	os.Setenv("OPENAI_API_KEY", "sk-from-env")
	defer os.Unsetenv("OPENAI_API_KEY")

	key := a.GetAPIKey("openai")
	if key != "sk-from-env" {
		t.Errorf("Expected env var key, got %s", key)
	}
}

func TestAuthStorage_HasAuth(t *testing.T) {
	a := InMemoryAuthStorage(nil)

	if a.HasAuth("openai") {
		t.Error("Expected no auth for unknown provider")
	}

	os.Setenv("OPENAI_API_KEY", "sk-test")
	defer os.Unsetenv("OPENAI_API_KEY")

	if !a.HasAuth("openai") {
		t.Error("Expected auth via env var")
	}
}

func TestAuthStorage_List(t *testing.T) {
	a := InMemoryAuthStorage(AuthStorageData{
		"openai":    {Type: CredentialTypeAPIKey, Key: "sk-1"},
		"anthropic": {Type: CredentialTypeAPIKey, Key: "sk-2"},
	})

	list := a.List()
	if len(list) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(list))
	}
}

func TestAuthStorage_GetAll(t *testing.T) {
	data := AuthStorageData{
		"openai": {Type: CredentialTypeAPIKey, Key: "sk-1"},
	}
	a := InMemoryAuthStorage(data)

	all := a.GetAll()
	if len(all) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(all))
	}

	// Verify it's a copy
	all["anthropic"] = AuthCredential{Type: CredentialTypeAPIKey, Key: "sk-2"}
	if a.Has("anthropic") {
		t.Error("GetAll should return a copy")
	}
}

func TestAuthStorage_PersistAndReload(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "auth_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	authPath := filepath.Join(tmpDir, "auth.json")

	// Write initial data
	a := NewAuthStorage(authPath)
	a.Set("openai", AuthCredential{Type: CredentialTypeAPIKey, Key: "sk-first"})

	// Verify file exists
	if _, err := os.Stat(authPath); os.IsNotExist(err) {
		t.Fatal("auth.json should exist after Set")
	}

	// Read file directly
	content, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatal(err)
	}

	var data AuthStorageData
	if err := json.Unmarshal(content, &data); err != nil {
		t.Fatal(err)
	}

	if data["openai"].Key != "sk-first" {
		t.Errorf("Expected sk-first, got %s", data["openai"].Key)
	}
}

func TestResolveConfigValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		env   map[string]string
		want  string
	}{
		{"plain string", "hello", nil, "hello"},
		{"empty string", "", nil, ""},
		{"env var", "${MY_KEY}", map[string]string{"MY_KEY": "resolved"}, "resolved"},
		{"env var not set", "${MISSING}", nil, "${MISSING}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}
			got := ResolveConfigValue(tt.input)
			if got != tt.want {
				t.Errorf("ResolveConfigValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetEnvAPIKey(t *testing.T) {
	tests := []struct {
		provider string
		envVar   string
		envVal   string
	}{
		{"anthropic", "ANTHROPIC_API_KEY", "sk-ant"},
		{"openai", "OPENAI_API_KEY", "sk-oai"},
		{"google", "GEMINI_API_KEY", "sk-gem"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			os.Setenv(tt.envVar, tt.envVal)
			defer os.Unsetenv(tt.envVar)

			got := getEnvAPIKey(tt.provider)
			if got != tt.envVal {
				t.Errorf("getEnvAPIKey(%q) = %q, want %q", tt.provider, got, tt.envVal)
			}
		})
	}
}
