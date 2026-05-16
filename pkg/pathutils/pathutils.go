package pathutils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ResolveToCwd resolves a path relative to the current working directory.
func ResolveToCwd(path, cwd string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(cwd, path)
}

// ShortenPath shortens a path for display by replacing the home directory with ~.
func ShortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// GetAgentDir returns the path to the pi agent configuration directory.
func GetAgentDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".pi"
	}
	return filepath.Join(home, ".pi")
}

// GetSessionsDir returns the path to the sessions directory.
func GetSessionsDir() string {
	return filepath.Join(GetAgentDir(), "sessions")
}

// GetShellConfig returns the shell and arguments for the current platform.
func GetShellConfig() (shell string, args []string) {
	if runtime.GOOS == "windows" {
		if _, err := os.Stat("powershell"); err == nil {
			return "powershell", []string{"-Command"}
		}
		return "cmd", []string{"/C"}
	}
	shell = os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	if strings.Contains(shell, "zsh") {
		return shell, []string{"-c"}
	}
	return shell, []string{"-c"}
}

// IsLocalPath checks if a path looks like a local file path.
func IsLocalPath(path string) bool {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return false
	}
	return true
}

