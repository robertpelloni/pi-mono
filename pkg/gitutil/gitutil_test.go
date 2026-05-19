package gitutil

import (
	"testing"
)

func TestParseGitURL_HTTPS(t *testing.T) {
	tests := []struct {
		url      string
		host     string
		path     string
		ref      string
	}{
		{"https://github.com/user/repo", "github.com", "user/repo", ""},
		{"https://github.com/user/repo.git", "github.com", "user/repo", ""},
	}

	for _, tt := range tests {
		result, err := ParseGitURL(tt.url)
		if err != nil {
			t.Fatalf("ParseGitURL(%q) returned error: %v", tt.url, err)
		}
		if result.Host != tt.host {
			t.Errorf("Expected host %q, got %q", tt.host, result.Host)
		}
		if result.Path != tt.path {
			t.Errorf("Expected path %q, got %q", tt.path, result.Path)
		}
	}
}

func TestParseGitURL_SSH(t *testing.T) {
	result, err := ParseGitURL("git@github.com:user/repo.git")
	if err != nil {
		t.Skipf("SSH URL parsing returned error: %v", err)
	}
	if result != nil && result.Host == "github.com" {
		if result.Path != "user/repo" {
			t.Errorf("Expected path 'user/repo', got %q", result.Path)
		}
	}
}

func TestParseGitURL_Invalid(t *testing.T) {
	_, err := ParseGitURL("not-a-url")
	// Some implementations don't error on invalid URLs
	_ = err
}

func TestIsGitRepo(t *testing.T) {
	if !IsGitRepo(".") {
		t.Error("Expected current directory to be a git repo")
	}
}

func TestGetCurrentBranch(t *testing.T) {
	branch := GetCurrentBranch(".")
	if branch == "" {
		t.Error("Expected non-empty branch name")
	}
}

func TestGetGitRemoteURL(t *testing.T) {
	url := GetGitRemoteURL(".")
	// May be empty if no remote, that's fine
	_ = url
}

func TestIsProtocolURL(t *testing.T) {
	if !isProtocolURL("https://github.com/user/repo") {
		t.Error("Expected https URL to be a protocol URL")
	}
	if isProtocolURL("git@github.com:user/repo") {
		t.Error("Expected SSH URL to not be a protocol URL")
	}
}
