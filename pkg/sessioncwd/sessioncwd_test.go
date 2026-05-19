package sessioncwd

import (
	"os"
	"strings"
	"testing"
)

type mockSessionCwdSource struct {
	cwd        string
	sessionFile string
}

func (m *mockSessionCwdSource) GetCwd() string        { return m.cwd }
func (m *mockSessionCwdSource) GetSessionFile() string { return m.sessionFile }

func TestGetMissingSessionCwdIssue_NoSessionFile(t *testing.T) {
	src := &mockSessionCwdSource{cwd: "/nonexistent", sessionFile: ""}
	issue := GetMissingSessionCwdIssue(src, "/fallback")
	if issue != nil {
		t.Error("Expected nil when no session file")
	}
}

func TestGetMissingSessionCwdIssue_ExistingCwd(t *testing.T) {
	wd, _ := os.Getwd()
	src := &mockSessionCwdSource{cwd: wd, sessionFile: "test.jsonl"}
	issue := GetMissingSessionCwdIssue(src, "/fallback")
	if issue != nil {
		t.Error("Expected nil when cwd exists")
	}
}

func TestGetMissingSessionCwdIssue_MissingCwd(t *testing.T) {
	src := &mockSessionCwdSource{cwd: "/absolutely/nonexistent/path", sessionFile: "test.jsonl"}
	issue := GetMissingSessionCwdIssue(src, "/fallback")
	if issue == nil {
		t.Fatal("Expected non-nil issue for missing cwd")
	}
	if issue.SessionCwd != "/absolutely/nonexistent/path" {
		t.Errorf("Expected SessionCwd to be set, got %s", issue.SessionCwd)
	}
	if issue.FallbackCwd != "/fallback" {
		t.Errorf("Expected FallbackCwd to be /fallback, got %s", issue.FallbackCwd)
	}
}

func TestFormatMissingSessionCwdError(t *testing.T) {
	issue := &SessionCwdIssue{
		SessionFile: "test.jsonl",
		SessionCwd:  "/missing",
		FallbackCwd: "/current",
	}
	result := FormatMissingSessionCwdError(issue)
	if !strings.Contains(result, "/missing") {
		t.Error("Expected missing cwd in error")
	}
	if !strings.Contains(result, "/current") {
		t.Error("Expected current cwd in error")
	}
	if !strings.Contains(result, "test.jsonl") {
		t.Error("Expected session file in error")
	}
}

func TestMissingSessionCwdError_Error(t *testing.T) {
	issue := &SessionCwdIssue{SessionCwd: "/missing", FallbackCwd: "/current"}
	err := &MissingSessionCwdError{Issue: issue}
	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}
