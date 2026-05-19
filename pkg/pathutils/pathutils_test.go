package pathutils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandPath_Home(t *testing.T) {
	result := ExpandPath("~")
	home, _ := os.UserHomeDir()
	if result != home {
		t.Errorf("Expected %q, got %q", home, result)
	}
}

func TestExpandPath_HomeSubdir(t *testing.T) {
	result := ExpandPath("~/Documents")
	home, _ := os.UserHomeDir()
	// On Windows, pathutils uses home + path[1:] which gives forward slash
	if !strings.HasPrefix(result, home) {
		t.Errorf("Expected result to start with home dir %q, got %q", home, result)
	}
	if !strings.Contains(result, "Documents") {
		t.Errorf("Expected result to contain 'Documents', got %q", result)
	}
}

func TestExpandPath_AtPrefix(t *testing.T) {
	result := ExpandPath("@file.txt")
	if result != "file.txt" {
		t.Errorf("Expected file.txt, got %q", result)
	}
}

func TestExpandPath_UnicodeSpaces(t *testing.T) {
	result := ExpandPath("hello\u00A0world")
	if result != "hello world" {
		t.Errorf("Expected normalized spaces, got %q", result)
	}
}

func TestResolveToCwd_Relative(t *testing.T) {
	result := ResolveToCwd("file.txt", "/home/user")
	if !strings.Contains(result, "file.txt") {
		t.Errorf("Expected result containing 'file.txt', got %q", result)
	}
}

func TestResolveToCwd_Absolute(t *testing.T) {
	// Use a path that is actually absolute on the current platform
	absPath, _ := filepath.Abs("/some/absolute/path")
	result := ResolveToCwd(absPath, "/home/user")
	if result != absPath {
		t.Errorf("Expected %q, got %q", absPath, result)
	}
}

func TestShortenPath_HomeDir(t *testing.T) {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, "Documents", "file.txt")
	result := ShortenPath(path)
	if !strings.HasPrefix(result, "~") {
		t.Errorf("Expected path starting with ~, got %q", result)
	}
}

func TestShortenPath_OtherDir(t *testing.T) {
	result := ShortenPath("/other/path/file.txt")
	if !strings.Contains(result, "other") {
		t.Errorf("Expected path containing 'other', got %q", result)
	}
}

func TestIsLocalPath(t *testing.T) {
	if !IsLocalPath("/home/user") {
		t.Error("Expected local path")
	}
	if IsLocalPath("https://example.com") {
		t.Error("Expected URL to not be local path")
	}
	if IsLocalPath("http://example.com") {
		t.Error("Expected URL to not be local path")
	}
}

func TestGetAgentDir(t *testing.T) {
	dir := GetAgentDir()
	if dir == "" {
		t.Error("Expected non-empty agent dir")
	}
}

func TestGetSessionsDir(t *testing.T) {
	dir := GetSessionsDir()
	if dir == "" {
		t.Error("Expected non-empty sessions dir")
	}
}

func TestGetShellConfig(t *testing.T) {
	shell, args := GetShellConfig()
	if shell == "" {
		t.Error("Expected non-empty shell")
	}
	_ = args
}

func TestNormalizeAtPrefix(t *testing.T) {
	if normalizeAtPrefix("@file.txt") != "file.txt" {
		t.Error("Expected @ prefix removed")
	}
	if normalizeAtPrefix("file.txt") != "file.txt" {
		t.Error("Expected no change without @")
	}
}

func TestNormalizeUnicodeSpaces(t *testing.T) {
	result := normalizeUnicodeSpaces("hello\u2000world\u3000test")
	if result != "hello world test" {
		t.Errorf("Expected normalized, got %q", result)
	}
}
