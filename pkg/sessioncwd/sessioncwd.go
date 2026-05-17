package sessioncwd

import (
	"fmt"
	"os"
)

// SessionCwdIssue describes a missing session working directory.
type SessionCwdIssue struct {
	SessionFile string `json:"sessionFile,omitempty"`
	SessionCwd  string `json:"sessionCwd"`
	FallbackCwd string `json:"fallbackCwd"`
}

// SessionCwdSource provides session directory info.
type SessionCwdSource interface {
	GetCwd() string
	GetSessionFile() string
}

// GetMissingSessionCwdIssue checks if the session's cwd exists.
func GetMissingSessionCwdIssue(sessionManager SessionCwdSource, fallbackCwd string) *SessionCwdIssue {
	sessionFile := sessionManager.GetSessionFile()
	if sessionFile == "" {
		return nil
	}
	sessionCwd := sessionManager.GetCwd()
	if sessionCwd == "" {
		return nil
	}
	if _, err := os.Stat(sessionCwd); err == nil {
		return nil
	}
	return &SessionCwdIssue{
		SessionFile: sessionFile,
		SessionCwd:  sessionCwd,
		FallbackCwd: fallbackCwd,
	}
}

// FormatMissingSessionCwdError formats the error message for a missing cwd.
func FormatMissingSessionCwdError(issue *SessionCwdIssue) string {
	sessionFile := ""
	if issue.SessionFile != "" {
		sessionFile = "\nSession file: " + issue.SessionFile
	}
	return fmt.Sprintf("Stored session working directory does not exist: %s%s\nCurrent working directory: %s",
		issue.SessionCwd, sessionFile, issue.FallbackCwd)
}

// MissingSessionCwdError is returned when a session's cwd doesn't exist.
type MissingSessionCwdError struct {
	Issue *SessionCwdIssue
}

func (e *MissingSessionCwdError) Error() string {
	return FormatMissingSessionCwdError(e.Issue)
}

// AssertSessionCwdExists panics if the session's cwd doesn't exist.
func AssertSessionCwdExists(sessionManager SessionCwdSource, fallbackCwd string) {
	issue := GetMissingSessionCwdIssue(sessionManager, fallbackCwd)
	if issue != nil {
		panic(&MissingSessionCwdError{Issue: issue})
	}
}
