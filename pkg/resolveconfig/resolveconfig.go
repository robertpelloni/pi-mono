package resolveconfig

import (
	"os"
	"os/exec"
	"strings"
	"sync"
)

var commandResultCache sync.Map

// ResolveConfigValue resolves a configuration value.
// - If starts with "!", executes the rest as a shell command and uses stdout (cached)
// - Otherwise checks environment variable first, then treats as literal
func ResolveConfigValue(config string) string {
	if strings.HasPrefix(config, "!") {
		return executeCommand(config)
	}
	envValue := os.Getenv(config)
	if envValue != "" {
		return envValue
	}
	return config
}

// ResolveConfigValueUncached resolves a config value without caching.
func ResolveConfigValueUncached(config string) string {
	if strings.HasPrefix(config, "!") {
		return executeCommandUncached(config)
	}
	envValue := os.Getenv(config)
	if envValue != "" {
		return envValue
	}
	return config
}

// ResolveConfigValueOrThrow resolves a config value or panics.
func ResolveConfigValueOrThrow(config string, description string) string {
	resolvedValue := ResolveConfigValueUncached(config)
	if resolvedValue != "" {
		return resolvedValue
	}
	if strings.HasPrefix(config, "!") {
		panic("Failed to resolve " + description + " from shell command: " + config[1:])
	}
	panic("Failed to resolve " + description)
}

// ResolveHeaders resolves all header values using the same resolution logic as API keys.
func ResolveHeaders(headers map[string]string) map[string]string {
	if headers == nil {
		return nil
	}
	resolved := make(map[string]string)
	for key, value := range headers {
		resolvedValue := ResolveConfigValue(value)
		if resolvedValue != "" {
			resolved[key] = resolvedValue
		}
	}
	if len(resolved) > 0 {
		return resolved
	}
	return nil
}

func executeCommandUncached(commandConfig string) string {
	command := commandConfig[1:]
	out, err := exec.Command("sh", "-c", command).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func executeCommand(commandConfig string) string {
	if cached, ok := commandResultCache.Load(commandConfig); ok {
		if s, ok := cached.(string); ok {
			return s
		}
	}
	result := executeCommandUncached(commandConfig)
	commandResultCache.Store(commandConfig, result)
	return result
}

// ClearConfigValueCache clears the command result cache.
func ClearConfigValueCache() {
	commandResultCache.Range(func(key, value interface{}) bool {
		commandResultCache.Delete(key)
		return true
	})
}
