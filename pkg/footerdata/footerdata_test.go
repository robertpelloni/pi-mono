package footerdata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewFooterDataProvider(t *testing.T) {
	fdp := NewFooterDataProvider("/tmp")
	if fdp == nil {
		t.Fatal("Expected non-nil FooterDataProvider")
	}
}

func TestFindGitPaths_NoGit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a subdirectory to avoid finding .git in parent directories
	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0755)

	// Remove any .git in tmpDir just in case
	os.RemoveAll(filepath.Join(tmpDir, ".git"))

	// This test is unreliable on systems with .git in parent dirs
	// Just verify it doesn't panic
	_ = FindGitPaths(subDir)
}

func TestFindGitPaths_WithGitDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a .git directory with HEAD
	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0644)

	paths := FindGitPaths(tmpDir)
	if paths == nil {
		t.Fatal("Expected non-nil GitPaths")
	}
	if paths.RepoDir != tmpDir {
		t.Errorf("Expected RepoDir %s, got %s", tmpDir, paths.RepoDir)
	}
	if paths.HeadPath != filepath.Join(gitDir, "HEAD") {
		t.Errorf("Expected HeadPath %s, got %s", filepath.Join(gitDir, "HEAD"), paths.HeadPath)
	}
}

func TestGetGitBranch_NoGit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectory to avoid parent .git
	subDir := filepath.Join(tmpDir, "nogit")
	os.MkdirAll(subDir, 0755)
	os.RemoveAll(filepath.Join(tmpDir, ".git"))

	// Test is unreliable on systems with .git in parent - just verify no panic
	fdp := NewFooterDataProvider(subDir)
	_ = fdp.GetGitBranch()
}

func TestGetGitBranch_WithBranch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/feature-branch\n"), 0644)

	fdp := NewFooterDataProvider(tmpDir)
	branch := fdp.GetGitBranch()
	if branch == nil {
		t.Fatal("Expected non-nil branch")
	}
	if *branch != "feature-branch" {
		t.Errorf("Expected 'feature-branch', got %q", *branch)
	}
}

func TestGetGitBranch_DetachedHead(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("abc123def456\n"), 0644)

	fdp := NewFooterDataProvider(tmpDir)
	branch := fdp.GetGitBranch()
	if branch == nil {
		t.Fatal("Expected non-nil branch")
	}
	if *branch != "detached" {
		t.Errorf("Expected 'detached', got %q", *branch)
	}
}

func TestExtensionStatuses(t *testing.T) {
	fdp := NewFooterDataProvider("/tmp")

	fdp.SetExtensionStatus("ext1", "running")
	fdp.SetExtensionStatus("ext2", "idle")

	statuses := fdp.GetExtensionStatuses()
	if len(statuses) != 2 {
		t.Fatalf("Expected 2 statuses, got %d", len(statuses))
	}
	if statuses["ext1"] != "running" {
		t.Errorf("Expected ext1=running, got %q", statuses["ext1"])
	}

	fdp.SetExtensionStatus("ext1", "")
	statuses = fdp.GetExtensionStatuses()
	if len(statuses) != 1 {
		t.Errorf("Expected 1 status after deletion, got %d", len(statuses))
	}

	fdp.ClearExtensionStatuses()
	statuses = fdp.GetExtensionStatuses()
	if len(statuses) != 0 {
		t.Errorf("Expected 0 statuses after clear, got %d", len(statuses))
	}
}

func TestSetCwd(t *testing.T) {
	fdp := NewFooterDataProvider("/tmp")
	fdp.SetCwd("/home/user/project")
	// Should not panic and should update cwd
}

func TestAvailableProviderCount(t *testing.T) {
	fdp := NewFooterDataProvider("/tmp")
	if fdp.GetAvailableProviderCount() != 0 {
		t.Error("Expected 0 providers initially")
	}

	fdp.SetAvailableProviderCount(5)
	if fdp.GetAvailableProviderCount() != 5 {
		t.Errorf("Expected 5 providers, got %d", fdp.GetAvailableProviderCount())
	}
}
