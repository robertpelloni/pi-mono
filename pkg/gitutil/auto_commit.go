package gitutil

import (
	"fmt"
	"os/exec"
	"strings"
)

// AutoCommitResult represents the result of an auto-commit operation.
type AutoCommitResult struct {
	Committed bool   `json:"committed"`
	Message   string `json:"message"`
	Files     int    `json:"files"`
	Error     string `json:"error,omitempty"`
}

// AutoCommitOptions controls auto-commit behavior.
type AutoCommitOptions struct {
	// Enabled enables auto-commit after file changes.
	Enabled bool
	// MaxDiffLength limits the diff used for commit message generation (0 = no limit).
	MaxDiffLength int
	// DryRun if true, reports what would be committed without actually committing.
	DryRun bool
}

// DefaultAutoCommitOptions returns sensible defaults for auto-commit.
func DefaultAutoCommitOptions() AutoCommitOptions {
	return AutoCommitOptions{
		Enabled:       true,
		MaxDiffLength: 4000,
		DryRun:        false,
	}
}

// AutoCommit stages all changes and commits them with an auto-generated message.
// Returns a result indicating whether a commit was made and the commit message.
func AutoCommit(cwd string, opts AutoCommitOptions, context string) AutoCommitResult {
	if !opts.Enabled {
		return AutoCommitResult{Committed: false}
	}

	if !IsGitRepo(cwd) {
		return AutoCommitResult{Committed: false}
	}

	// Check if there are any changes to commit
	hasChanges, err := hasUncommittedChanges(cwd)
	if err != nil || !hasChanges {
		return AutoCommitResult{Committed: false}
	}

	// Get diff for commit message generation
	diffContent, err := getDiff(cwd, opts.MaxDiffLength)
	if err != nil {
		return AutoCommitResult{Committed: false, Error: err.Error()}
	}

	// Count changed files
	files := countChangedFiles(cwd)

	if opts.DryRun {
		msg := generateCommitMessage(diffContent, context)
		return AutoCommitResult{
			Committed: true,
			Message:   msg,
			Files:     files,
		}
	}

	// Stage all changes
	if err := stageAll(cwd); err != nil {
		return AutoCommitResult{
			Committed: false,
			Error:     fmt.Sprintf("stage failed: %v", err),
		}
	}

	// Generate commit message from diff
	msg := generateCommitMessage(diffContent, context)

	// Commit
	if err := commit(cwd, msg); err != nil {
		return AutoCommitResult{
			Committed: false,
			Error:     fmt.Sprintf("commit failed: %v", err),
		}
	}

	return AutoCommitResult{
		Committed: true,
		Message:   msg,
		Files:     files,
	}
}

// hasUncommittedChanges checks if there are any uncommitted changes in the working tree.
func hasUncommittedChanges(cwd string) (bool, error) {
	// Check for unstaged changes
	cmd := exec.Command("git", "--no-optional-locks", "diff", "--quiet", "--exit-code")
	cmd.Dir = cwd
	unstagedErr := cmd.Run()

	// Check for staged but uncommitted changes
	cmd2 := exec.Command("git", "--no-optional-locks", "diff", "--cached", "--quiet", "--exit-code")
	cmd2.Dir = cwd
	stagedErr := cmd2.Run()

	// If either has changes, exit code is non-zero
	return unstagedErr != nil || stagedErr != nil, nil
}

// getDiff returns the diff against HEAD, optionally truncated.
func getDiff(cwd string, maxLen int) (string, error) {
	cmd := exec.Command("git", "--no-optional-locks", "diff", "HEAD")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}

	diff := strings.TrimSpace(string(out))
	if maxLen > 0 && len(diff) > maxLen {
		diff = diff[:maxLen] + "\n... (diff truncated)"
	}
	return diff, nil
}

// countChangedFiles counts the number of changed files (staged + unstaged).
func countChangedFiles(cwd string) int {
	cmd := exec.Command("git", "--no-optional-locks", "diff", "--name-only", "HEAD")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0
	}
	return len(lines)
}

// stageAll stages all changes.
func stageAll(cwd string) error {
	cmd := exec.Command("git", "--no-optional-locks", "add", "-A")
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add -A failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// commit creates a commit with the given message.
func commit(cwd string, msg string) error {
	cmd := exec.Command("git", "--no-optional-locks", "commit", "-m", msg)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// generateCommitMessage creates a commit message from the diff content and optional context.
// This mimics aider's approach of generating concise commit messages from diffs.
func generateCommitMessage(diff, context string) string {
	var sb strings.Builder

	// Extract file names from diff
	files := extractChangedFiles(diff)
	if len(files) > 0 {
		if len(files) <= 3 {
			sb.WriteString(strings.Join(files, ", "))
		} else {
			sb.WriteString(fmt.Sprintf("%s and %d more", files[0], len(files)-1))
		}
	} else {
		sb.WriteString("auto-commit")
	}

	if context != "" {
		// Clean up context for use in commit message
		ctx := strings.TrimSpace(context)
		if len(ctx) > 100 {
			ctx = ctx[:100] + "..."
		}
		sb.WriteString(fmt.Sprintf(": %s", ctx))
	}

	msg := sb.String()
	if len(msg) > 200 {
		msg = msg[:200]
	}

	return msg
}

// extractChangedFiles parses a git diff and returns the list of changed files.
func extractChangedFiles(diff string) []string {
	lines := strings.Split(diff, "\n")
	var files []string
	seen := make(map[string]bool)

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			// Extract file path from "diff --git a/file b/file"
			parts := strings.Split(line, " ")
			if len(parts) >= 4 {
				// Take the "b/file" path
				filePath := strings.TrimPrefix(parts[len(parts)-1], "b/")
				if !seen[filePath] {
					files = append(files, filePath)
					seen[filePath] = true
				}
			}
		}
	}
	return files
}

// GetGitContext returns git-aware context for injection into system prompts.
// This provides the model with awareness of the current git state.
func GetGitContext(workingDir string) (string, error) {
	if !IsGitRepo(workingDir) {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString("<git_context>\n")

	// Current branch
	branch := GetCurrentBranch(workingDir)
	if branch != "" {
		sb.WriteString(fmt.Sprintf("Current Branch: %s\n\n", branch))
	}

	// Uncommitted changes
	diff, err := getDiff(workingDir, 4000)
	if err == nil && diff != "" {
		sb.WriteString("Uncommitted Changes (Diff against HEAD):\n```diff\n")
		sb.WriteString(diff)
		sb.WriteString("\n```\n\n")
	} else if err == nil {
		sb.WriteString("Working tree is clean. No uncommitted changes.\n\n")
	}

	// Recent commits (last 5)
	cmd := exec.Command("git", "--no-optional-locks", "log", "-n", "5", "--oneline")
	cmd.Dir = workingDir
	logOut, logErr := cmd.Output()
	if logErr == nil {
		logText := strings.TrimSpace(string(logOut))
		if logText != "" {
			sb.WriteString("Recent Commits:\n```\n")
			sb.WriteString(logText)
			sb.WriteString("\n```\n")
		}
	}

	// Untracked files
	cmd2 := exec.Command("git", "--no-optional-locks", "ls-files", "--others", "--exclude-standard")
	cmd2.Dir = workingDir
	untrackedOut, _ := cmd2.Output()
	untrackedStr := strings.TrimSpace(string(untrackedOut))
	if untrackedStr != "" {
		untrackedFiles := strings.Split(untrackedStr, "\n")
		if len(untrackedFiles) > 0 {
			sb.WriteString("Untracked Files:\n")
			for _, f := range untrackedFiles {
				if f != "" {
					sb.WriteString(fmt.Sprintf("- %s\n", f))
				}
			}
		}
	}

	sb.WriteString("</git_context>\n")
	return sb.String(), nil
}
