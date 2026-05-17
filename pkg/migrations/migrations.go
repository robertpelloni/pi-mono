package migrations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MigrationResult holds the results of running migrations.
type MigrationResult struct {
	MigratedAuthProviders []string
	DeprecationWarnings   []string
}

// RunMigrations runs all one-time startup migrations.
func RunMigrations(cwd, agentDir string) MigrationResult {
	result := MigrationResult{}

	// Migrate oauth.json and settings.json apiKeys to auth.json
	result.MigratedAuthProviders = migrateAuthToAuthJSON(agentDir)

	// Migrate sessions from agent root to proper session directories
	migrateSessionsFromAgentRoot(agentDir)

	// Migrate commands/ to prompts/
	result.DeprecationWarnings = migrateExtensionSystem(cwd, agentDir)

	return result
}

// migrateAuthToAuthJSON migrates legacy oauth.json and settings.json apiKeys to auth.json.
func migrateAuthToAuthJSON(agentDir string) []string {
	authPath := filepath.Join(agentDir, "auth.json")
	oauthPath := filepath.Join(agentDir, "oauth.json")
	settingsPath := filepath.Join(agentDir, "settings.json")

	// Skip if auth.json already exists
	if _, err := os.Stat(authPath); err == nil {
		return nil
	}

	migrated := make(map[string]interface{})
	var providers []string

	// Migrate oauth.json
	if content, err := os.ReadFile(oauthPath); err == nil {
		var oauth map[string]interface{}
		if err := json.Unmarshal(content, &oauth); err == nil {
			for provider, cred := range oauth {
				migrated[provider] = map[string]interface{}{
					"type": "oauth",
				}
				// Copy credential fields
				if credMap, ok := cred.(map[string]interface{}); ok {
					for k, v := range credMap {
						migrated[provider].(map[string]interface{})[k] = v
					}
				}
				providers = append(providers, provider)
			}
			// Rename old file
			os.Rename(oauthPath, oauthPath+".migrated")
		}
	}

	// Migrate settings.json apiKeys
	if content, err := os.ReadFile(settingsPath); err == nil {
		var settings map[string]interface{}
		if err := json.Unmarshal(content, &settings); err == nil {
			if apiKeys, ok := settings["apiKeys"].(map[string]interface{}); ok {
				for provider, key := range apiKeys {
					if _, exists := migrated[provider]; !exists {
						if keyStr, ok := key.(string); ok {
							migrated[provider] = map[string]interface{}{
								"type": "api_key",
								"key":  keyStr,
							}
							providers = append(providers, provider)
						}
					}
				}
				// Remove apiKeys from settings
				delete(settings, "apiKeys")
				if newContent, err := json.MarshalIndent(settings, "", "  "); err == nil {
					os.WriteFile(settingsPath, newContent, 0644)
				}
			}
		}
	}

	// Write auth.json if we migrated anything
	if len(migrated) > 0 {
		os.MkdirAll(filepath.Dir(authPath), 0700)
		content, _ := json.MarshalIndent(migrated, "", "  ")
		os.WriteFile(authPath, content, 0600)
	}

	return providers
}

// migrateSessionsFromAgentRoot moves .jsonl files from agent root to session dirs.
func migrateSessionsFromAgentRoot(agentDir string) {
	entries, err := os.ReadDir(agentDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		filePath := filepath.Join(agentDir, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		// Read first line for session header
		lines := strings.Split(string(content), "\n")
		if len(lines) == 0 {
			continue
		}

		var header map[string]interface{}
		if err := json.Unmarshal([]byte(lines[0]), &header); err != nil {
			continue
		}

		headerType, _ := header["type"].(string)
		sessionCwd, _ := header["cwd"].(string)
		if headerType != "session" || sessionCwd == "" {
			continue
		}

		// Compute correct session directory
		safePath := fmt.Sprintf("--%s--", strings.ReplaceAll(
			strings.TrimPrefix(strings.TrimPrefix(sessionCwd, "/"), "\\"), "/", "-"))
		safePath = strings.ReplaceAll(safePath, "\\", "-")
		safePath = strings.ReplaceAll(safePath, ":", "-")

		correctDir := filepath.Join(agentDir, "sessions", safePath)
		os.MkdirAll(correctDir, 0755)

		newPath := filepath.Join(correctDir, entry.Name())
		if _, err := os.Stat(newPath); err == nil {
			continue // Target exists
		}

		os.Rename(filePath, newPath)
	}
}

// migrateExtensionSystem migrates commands/ to prompts/ and checks for deprecated dirs.
func migrateExtensionSystem(cwd, agentDir string) []string {
	var warnings []string

	// Migrate commands/ to prompts/
	migrateDir := func(baseDir, label string) {
		commandsDir := filepath.Join(baseDir, "commands")
		promptsDir := filepath.Join(baseDir, "prompts")

		if _, err := os.Stat(commandsDir); err == nil {
			if _, err := os.Stat(promptsDir); err != nil {
				if err := os.Rename(commandsDir, promptsDir); err == nil {
					fmt.Fprintf(os.Stderr, "Migrated %s commands/ → prompts/\n", label)
				}
			}
		}
	}

	migrateDir(agentDir, "Global")
	migrateDir(filepath.Join(cwd, ".pi"), "Project")

	// Check for deprecated directories
	checkDeprecated := func(baseDir, label string) {
		hooksDir := filepath.Join(baseDir, "hooks")
		if _, err := os.Stat(hooksDir); err == nil {
			warnings = append(warnings, fmt.Sprintf("%s hooks/ directory found. Hooks have been renamed to extensions.", label))
		}
	}

	checkDeprecated(agentDir, "Global")
	checkDeprecated(filepath.Join(cwd, ".pi"), "Project")

	return warnings
}
