package filesafety

import (
	"os"
	"path/filepath"
	"strings"
)

// IsWriteDenied returns true if the given path is on the denylist for writes.
// It mirrors the logic from Hermes‑agent's file_safety.py.
func IsWriteDenied(p string) bool {
	home, _ := os.UserHomeDir()
	home = filepath.Clean(home)
	resolved := filepath.Join(home, p)
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return false
	}
	resolved = filepath.Clean(abs)

	// Exact denied paths
	for _, p := range writeDeniedPaths(home) {
		if resolved == p {
			return true
		}
	}

	// Denied prefixes
	for _, prefix := range writeDeniedPrefixes(home) {
		if strings.HasPrefix(resolved, prefix) {
			return true
		}
	}

	// HERMES_WRITE_SAFE_ROOT handling – if set, restrict to that root
	if safeRoot := getSafeWriteRoot(); safeRoot != "" {
		if !(resolved == safeRoot || strings.HasPrefix(resolved, safeRoot+string(os.PathSeparator))) {
			return true
		}
	}

	return false
}

// GetReadBlockError returns an error string if reading the path is blocked, otherwise empty.
func GetReadBlockError(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return ""
	}
	resolved := filepath.Clean(abs)

	// Block internal Hermes cache & credential files – simplified set
	blocked := blockedReadPaths()
	for _, p := range blocked {
		if resolved == p {
			return "read denied: protected Hermes file"
		}
	}

	// Block project env files
	base := filepath.Base(resolved)
	for _, name := range blockedProjectEnvBasenames() {
		if base == name {
			return "read denied: project .env files"
		}
	}

	return ""
}

// Helper functions – simplified versions of the Python equivalents.

func writeDeniedPaths(home string) []string {
	return []string{
		filepath.Join(home, ".ssh", "authorized_keys"),
		filepath.Join(home, ".ssh", "id_rsa"),
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "config"),
		filepath.Join(home, ".netrc"),
		filepath.Join(home, ".pgpass"),
		filepath.Join(home, ".npmrc"),
		filepath.Join(home, ".pypirc"),
		filepath.Join(home, ".git-credentials"),
		"/etc/sudoers",
		"/etc/passwd",
		"/etc/shadow",
	}
}

func writeDeniedPrefixes(home string) []string {
	return []string{
		filepath.Join(home, ".ssh") + string(os.PathSeparator),
		filepath.Join(home, ".aws") + string(os.PathSeparator),
		filepath.Join(home, ".gnupg") + string(os.PathSeparator),
		filepath.Join(home, ".kube") + string(os.PathSeparator),
		"/etc/sudoers.d/",
		"/etc/systemd/",
		filepath.Join(home, ".docker") + string(os.PathSeparator),
		filepath.Join(home, ".azure") + string(os.PathSeparator),
		filepath.Join(home, ".config", "gh") + string(os.PathSeparator),
		filepath.Join(home, ".config", "gcloud") + string(os.PathSeparator),
	}
}

func getSafeWriteRoot() string {
	if v := os.Getenv("HERMES_WRITE_SAFE_ROOT"); v != "" {
		if abs, err := filepath.Abs(v); err == nil {
			return abs
		}
	}
	return ""
}

func blockedReadPaths() []string {
	// Simplified: block Hermes cache and credential files under HOME/.hermes
	home, _ := os.UserHomeDir()
	hermesHome := filepath.Join(home, ".hermes")
	return []string{
		filepath.Join(hermesHome, "auth.json"),
		filepath.Join(hermesHome, "auth.lock"),
		filepath.Join(hermesHome, ".anthropic_oauth.json"),
		filepath.Join(hermesHome, ".env"),
		filepath.Join(hermesHome, "skills", ".hub"),
	}
}

func blockedProjectEnvBasenames() []string {
	return []string{
		".env",
		".env.local",
		".env.development",
		".env.production",
		".env.test",
		".env.staging",
		".envrc",
	}
}
