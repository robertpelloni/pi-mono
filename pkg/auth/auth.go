package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuthToken represents a stored authentication token.
type AuthToken struct {
	Provider    string    `json:"provider"`
	AccessToken string    `json:"accessToken"`
	RefreshToken string   `json:"refreshToken,omitempty"`
	ExpiresAt   time.Time `json:"expiresAt,omitempty"`
	Scope       string    `json:"scope,omitempty"`
	ObtainedAt  time.Time `json:"obtainedAt"`
}

// IsExpired checks if the token has expired.
func (t *AuthToken) IsExpired() bool {
	if t.ExpiresAt.IsZero() {
		return false // No expiration set
	}
	return time.Now().After(t.ExpiresAt)
}

// AuthStorage manages authentication tokens on disk.
type AuthStorage struct {
	mu   sync.RWMutex
	dir  string
	file string
}

// NewAuthStorage creates an auth storage in the agent directory.
func NewAuthStorage(agentDir string) *AuthStorage {
	dir := filepath.Join(agentDir, "auth")
	return &AuthStorage{
		dir:  dir,
		file: filepath.Join(dir, "tokens.json"),
	}
}

// SaveToken stores a token for a provider.
func (s *AuthStorage) SaveToken(token AuthToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	os.MkdirAll(s.dir, 0700) // Restrict permissions for security

	tokens, err := s.loadTokens()
	if err != nil {
		tokens = make(map[string]AuthToken)
	}

	token.ObtainedAt = time.Now()
	tokens[token.Provider] = token

	return s.saveTokens(tokens)
}

// GetToken retrieves a token for a provider.
func (s *AuthStorage) GetToken(provider string) (*AuthToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tokens, err := s.loadTokens()
	if err != nil {
		return nil, err
	}

	token, ok := tokens[provider]
	if !ok {
		return nil, fmt.Errorf("no token found for provider: %s", provider)
	}

	return &token, nil
}

// DeleteToken removes a token for a provider.
func (s *AuthStorage) DeleteToken(provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokens, err := s.loadTokens()
	if err != nil {
		return err
	}

	delete(tokens, provider)
	return s.saveTokens(tokens)
}

// ListProviders returns all providers with stored tokens.
func (s *AuthStorage) ListProviders() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tokens, err := s.loadTokens()
	if err != nil {
		return nil, err
	}

	providers := make([]string, 0, len(tokens))
	for p := range tokens {
		providers = append(providers, p)
	}
	return providers, nil
}

// Logout removes all stored tokens.
func (s *AuthStorage) Logout() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return os.Remove(s.file)
}

func (s *AuthStorage) loadTokens() (map[string]AuthToken, error) {
	data, err := os.ReadFile(s.file)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]AuthToken), nil
		}
		return nil, err
	}

	tokens := make(map[string]AuthToken)
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, fmt.Errorf("invalid token file: %w", err)
	}

	return tokens, nil
}

func (s *AuthStorage) saveTokens(tokens map[string]AuthToken) error {
	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return err
	}

	// Write with restricted permissions (only owner can read/write)
	if err := os.WriteFile(s.file, data, 0600); err != nil {
		return err
	}

	return nil
}
