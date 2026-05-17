package migrations

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunMigrations_EmptyDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "migration_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	result := RunMigrations(tmpDir, tmpDir)
	if len(result.MigratedAuthProviders) != 0 {
		t.Errorf("Expected no migrated providers, got %v", result.MigratedAuthProviders)
	}
}

func TestMigrateAuthToAuthJSON_NoFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "migration_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	providers := migrateAuthToAuthJSON(tmpDir)
	if len(providers) != 0 {
		t.Errorf("Expected no providers, got %v", providers)
	}

	// auth.json should not be created when there's nothing to migrate
	authPath := filepath.Join(tmpDir, "auth.json")
	if _, err := os.Stat(authPath); err == nil {
		t.Error("auth.json should not be created when there's nothing to migrate")
	}
}

func TestMigrateAuthToAuthJSON_WithOauthJson(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "migration_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create oauth.json
	oauthContent := `{"anthropic": {"accessToken": "test-token", "refreshToken": "refresh", "expires": 1234567890}}`
	oauthPath := filepath.Join(tmpDir, "oauth.json")
	os.WriteFile(oauthPath, []byte(oauthContent), 0644)

	// Also create auth.json to skip migration
	authPath := filepath.Join(tmpDir, "auth.json")

	// First test: migrate when auth.json doesn't exist
	providers := migrateAuthToAuthJSON(tmpDir)
	if len(providers) != 1 || providers[0] != "anthropic" {
		t.Errorf("Expected [anthropic], got %v", providers)
	}

	// auth.json should now exist
	if _, err := os.Stat(authPath); os.IsNotExist(err) {
		t.Fatal("auth.json should exist after migration")
	}

	// oauth.json should be renamed
	if _, err := os.Stat(oauthPath); err == nil {
		t.Error("oauth.json should be renamed to .migrated")
	}
	if _, err := os.Stat(oauthPath + ".migrated"); os.IsNotExist(err) {
		t.Error("oauth.json.migrated should exist")
	}

	// Second test: don't migrate when auth.json already exists
	providers2 := migrateAuthToAuthJSON(tmpDir)
	if len(providers2) != 0 {
		t.Errorf("Expected no migration when auth.json exists, got %v", providers2)
	}
}

func TestMigrateSessionsFromAgentRoot_NoFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "migration_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Should not panic with empty directory
	migrateSessionsFromAgentRoot(tmpDir)
}

func TestMigrateExtensionSystem_NoDirs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "migration_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	warnings := migrateExtensionSystem(tmpDir, tmpDir)
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings, got %v", warnings)
	}
}

func TestMigrateExtensionSystem_DeprecatedHooksDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "migration_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a hooks directory
	hooksDir := filepath.Join(tmpDir, "hooks")
	os.MkdirAll(hooksDir, 0755)

	warnings := migrateExtensionSystem(tmpDir, tmpDir)
	if len(warnings) == 0 {
		t.Error("Expected warning about hooks/ directory")
	}
}
