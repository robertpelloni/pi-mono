package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// CredentialType represents the type of authentication credential.
type CredentialType string

const (
	CredentialTypeAPIKey CredentialType = "api_key"
	CredentialTypeOAuth  CredentialType = "oauth"
)

// AuthCredential represents a stored authentication credential.
type AuthCredential struct {
	Type CredentialType `json:"type"`
	Key  string         `json:"key,omitempty"` // For api_key type

	// OAuth fields
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
	Expires      int64  `json:"expires,omitempty"`
	TokenType    string `json:"tokenType,omitempty"`
}

// AuthStorageData is the full auth.json structure.
type AuthStorageData map[string]AuthCredential

// AuthStorage manages API keys and OAuth tokens.
// It provides a hierarchical key resolution: runtime overrides > auth.json > env vars > fallback.
type AuthStorage struct {
	mu               sync.RWMutex
	data             AuthStorageData
	filePath         string
	runtimeOverrides map[string]string // CLI --api-key overrides
	fallbackResolver func(provider string) string
	loadError        error
}

// NewAuthStorage creates a new AuthStorage backed by a JSON file.
func NewAuthStorage(authPath string) *AuthStorage {
	a := &AuthStorage{
		filePath:         authPath,
		data:             make(AuthStorageData),
		runtimeOverrides: make(map[string]string),
	}
	a.reload()
	return a
}

// InMemoryAuthStorage creates an in-memory auth storage.
func InMemoryAuthStorage(data AuthStorageData) *AuthStorage {
	if data == nil {
		data = make(AuthStorageData)
	}
	return &AuthStorage{
		data:             data,
		runtimeOverrides: make(map[string]string),
	}
}

// reload reads credentials from the auth.json file.
func (a *AuthStorage) reload() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.filePath == "" {
		return
	}

	content, err := os.ReadFile(a.filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			a.loadError = err
		}
		return
	}

	var data AuthStorageData
	if err := json.Unmarshal(content, &data); err != nil {
		a.loadError = fmt.Errorf("failed to parse auth.json: %w", err)
		return
	}

	a.data = data
	a.loadError = nil
}

// persist writes current data to the auth.json file.
func (a *AuthStorage) persist() error {
	if a.filePath == "" {
		return nil
	}

	dir := filepath.Dir(a.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create auth directory: %w", err)
	}

	content, err := json.MarshalIndent(a.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth data: %w", err)
	}

	if err := os.WriteFile(a.filePath, content, 0600); err != nil {
		return fmt.Errorf("failed to write auth.json: %w", err)
	}

	return nil
}

// Get returns the credential for a provider.
func (a *AuthStorage) Get(provider string) *AuthCredential {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if cred, ok := a.data[provider]; ok {
		return &cred
	}
	return nil
}

// Set stores a credential for a provider.
func (a *AuthStorage) Set(provider string, credential AuthCredential) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.data[provider] = credential
	return a.persist()
}

// Remove deletes a credential for a provider.
func (a *AuthStorage) Remove(provider string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.data, provider)
	return a.persist()
}

// List returns all providers with credentials.
func (a *AuthStorage) List() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	providers := make([]string, 0, len(a.data))
	for p := range a.data {
		providers = append(providers, p)
	}
	return providers
}

// Has checks if credentials exist in auth.json for a provider.
func (a *AuthStorage) Has(provider string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	_, ok := a.data[provider]
	return ok
}

// HasAuth checks if any form of auth is configured for a provider.
// Checks: runtime overrides > auth.json > env vars > fallback resolver.
func (a *AuthStorage) HasAuth(provider string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if _, ok := a.runtimeOverrides[provider]; ok {
		return true
	}
	if _, ok := a.data[provider]; ok {
		return true
	}
	if envKey := getEnvAPIKey(provider); envKey != "" {
		return true
	}
	if a.fallbackResolver != nil {
		if key := a.fallbackResolver(provider); key != "" {
			return true
		}
	}
	return false
}

// GetAPIKey returns the API key for a provider.
// Priority: runtime override > auth.json api_key > auth.json oauth > env var > fallback.
func (a *AuthStorage) GetAPIKey(provider string) string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Runtime override
	if key, ok := a.runtimeOverrides[provider]; ok {
		return key
	}

	// Auth.json credential
	if cred, ok := a.data[provider]; ok {
		if cred.Type == CredentialTypeAPIKey {
			return ResolveConfigValue(cred.Key)
		}
		if cred.Type == CredentialTypeOAuth {
			if cred.Expires > 0 && cred.AccessToken != "" {
				return cred.AccessToken
			}
		}
	}

	// Env var
	if envKey := getEnvAPIKey(provider); envKey != "" {
		return envKey
	}

	// Fallback resolver
	if a.fallbackResolver != nil {
		return a.fallbackResolver(provider)
	}

	return ""
}

// SetRuntimeAPIKey sets a runtime API key override (not persisted).
func (a *AuthStorage) SetRuntimeAPIKey(provider, apiKey string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.runtimeOverrides[provider] = apiKey
}

// RemoveRuntimeAPIKey removes a runtime API key override.
func (a *AuthStorage) RemoveRuntimeAPIKey(provider string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.runtimeOverrides, provider)
}

// SetFallbackResolver sets a fallback resolver for API keys.
func (a *AuthStorage) SetFallbackResolver(resolver func(provider string) string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.fallbackResolver = resolver
}

// GetAll returns all credentials.
func (a *AuthStorage) GetAll() AuthStorageData {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(AuthStorageData, len(a.data))
	for k, v := range a.data {
		result[k] = v
	}
	return result
}

// DrainErrors returns any load errors and clears them.
func (a *AuthStorage) DrainErrors() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	err := a.loadError
	a.loadError = nil
	return err
}

// getEnvAPIKey returns the API key from environment variables for a provider.
func getEnvAPIKey(provider string) string {
	switch provider {
	case "anthropic":
		return os.Getenv("ANTHROPIC_API_KEY")
	case "openai":
		return os.Getenv("OPENAI_API_KEY")
	case "google", "gemini":
		return os.Getenv("GEMINI_API_KEY")
	case "xai":
		return os.Getenv("XAI_API_KEY")
	case "groq":
		return os.Getenv("GROQ_API_KEY")
	case "openrouter":
		return os.Getenv("OPENROUTER_API_KEY")
	case "mistral":
		return os.Getenv("MISTRAL_API_KEY")
	case "ollama-cloud", "ollama":
		return os.Getenv("OLLAMA_API_KEY")
	case "azure-openai", "azure-openai-responses":
		key := os.Getenv("AZURE_OPENAI_API_KEY")
		if key == "" {
			key = os.Getenv("AZURE_API_KEY")
		}
		return key
	default:
		envName := toEnvName(provider) + "_API_KEY"
		return os.Getenv(envName)
	}
}

// toEnvName converts a provider name to an environment variable name prefix.
func toEnvName(provider string) string {
	var b strings.Builder
	for i := 0; i < len(provider); i++ {
		c := provider[i]
		if c == '-' || c == ' ' || c == '.' {
			b.WriteByte('_')
		} else {
			b.WriteByte(c)
		}
	}
	return b.String()
}

// ResolveConfigValue resolves a config value that may reference an env var or command.
// Supports: plain string, ${ENV_VAR}, $(command).
func ResolveConfigValue(value string) string {
	if value == "" {
		return ""
	}

	// ${ENV_VAR} pattern
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		envName := value[2 : len(value)-1]
		if envVal := os.Getenv(envName); envVal != "" {
			return envVal
		}
		return value
	}

	// $(command) pattern - execute command and use output
	if strings.HasPrefix(value, "$(") && strings.HasSuffix(value, ")") {
		cmd := value[2 : len(value)-1]
		result, err := runShellCommand(cmd)
		if err == nil {
			return strings.TrimSpace(result)
		}
		return value
	}

	return value
}

// runShellCommand runs a shell command and returns its stdout.
func runShellCommand(cmd string) (string, error) {
	var shell string
	var args []string

	if runtime.GOOS == "windows" {
		shell = "cmd"
		args = []string{"/C", cmd}
	} else {
		shell = "/bin/sh"
		args = []string{"-c", cmd}
	}

	out, err := exec.Command(shell, args...).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
