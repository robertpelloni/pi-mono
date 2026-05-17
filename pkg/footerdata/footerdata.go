package footerdata

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

// FooterDataProvider provides git branch and extension status data.
type FooterDataProvider struct {
	cwd                    string
	cachedBranch           atomic.Value // string | nil
	extensionStatuses      map[string]string
	extensionStatusesMu    sync.RWMutex
	availableProviderCount int
	branchChangeCallbacks  []func()
	branchChangeMu        sync.Mutex
}

// NewFooterDataProvider creates a new FooterDataProvider for the given cwd.
func NewFooterDataProvider(cwd string) *FooterDataProvider {
	fdp := &FooterDataProvider{
		cwd:               cwd,
		extensionStatuses: make(map[string]string),
	}
	fdp.cachedBranch.Store((*string)(nil))
	return fdp
}

// GitPaths holds paths to git metadata.
type GitPaths struct {
	RepoDir      string
	CommonGitDir string
	HeadPath     string
}

// FindGitPaths finds git metadata paths by walking up from cwd.
func FindGitPaths(cwd string) *GitPaths {
	dir := cwd
	for {
		gitPath := filepath.Join(dir, ".git")
		info, err := os.Stat(gitPath)
		if err == nil {
			if !info.IsDir() {
				// Worktree: .git is a file pointing to gitdir
				content, err := os.ReadFile(gitPath)
				if err == nil && strings.HasPrefix(string(content), "gitdir: ") {
					gitDir := strings.TrimSpace(string(content)[8:])
					if !filepath.IsAbs(gitDir) {
						gitDir = filepath.Join(dir, gitDir)
					}
					headPath := filepath.Join(gitDir, "HEAD")
					if _, err := os.Stat(headPath); err != nil {
						return nil
					}
					commonDirPath := filepath.Join(gitDir, "commondir")
					commonGitDir := gitDir
					if content, err := os.ReadFile(commonDirPath); err == nil {
						relDir := strings.TrimSpace(string(content))
						if !filepath.IsAbs(relDir) {
							commonGitDir = filepath.Join(gitDir, relDir)
						} else {
							commonGitDir = relDir
						}
					}
					return &GitPaths{
						RepoDir:      dir,
						CommonGitDir: commonGitDir,
						HeadPath:     headPath,
					}
				}
			} else {
				// Regular repo
				headPath := filepath.Join(gitPath, "HEAD")
				if _, err := os.Stat(headPath); err != nil {
					return nil
				}
				return &GitPaths{
					RepoDir:      dir,
					CommonGitDir: gitPath,
					HeadPath:     headPath,
				}
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
		dir = parent
	}
}

// GetGitBranch returns the current git branch, nil if not in repo, "detached" if detached HEAD.
func (f *FooterDataProvider) GetGitBranch() *string {
	cached := f.cachedBranch.Load()
	if cached != nil {
		if s, ok := cached.(*string); ok && s != nil {
			return s
		}
	}

	branch := f.resolveGitBranchSync()
	f.cachedBranch.Store(branch)
	return branch
}

// GetExtensionStatuses returns a copy of the extension status map.
func (f *FooterDataProvider) GetExtensionStatuses() map[string]string {
	f.extensionStatusesMu.RLock()
	defer f.extensionStatusesMu.RUnlock()

	result := make(map[string]string, len(f.extensionStatuses))
	for k, v := range f.extensionStatuses {
		result[k] = v
	}
	return result
}

// SetExtensionStatus sets an extension status.
func (f *FooterDataProvider) SetExtensionStatus(key, text string) {
	f.extensionStatusesMu.Lock()
	defer f.extensionStatusesMu.Unlock()

	if text == "" {
		delete(f.extensionStatuses, key)
	} else {
		f.extensionStatuses[key] = text
	}
}

// ClearExtensionStatuses clears all extension statuses.
func (f *FooterDataProvider) ClearExtensionStatuses() {
	f.extensionStatusesMu.Lock()
	defer f.extensionStatusesMu.Unlock()
	f.extensionStatuses = make(map[string]string)
}

// GetAvailableProviderCount returns the number of providers with available models.
func (f *FooterDataProvider) GetAvailableProviderCount() int {
	return f.availableProviderCount
}

// SetAvailableProviderCount sets the available provider count.
func (f *FooterDataProvider) SetAvailableProviderCount(count int) {
	f.availableProviderCount = count
}

// OnBranchChange subscribes to git branch changes. Returns unsubscribe function.
func (f *FooterDataProvider) OnBranchChange(callback func()) func() {
	f.branchChangeMu.Lock()
	defer f.branchChangeMu.Unlock()

	f.branchChangeCallbacks = append(f.branchChangeCallbacks, callback)
	return func() {
		f.branchChangeMu.Lock()
		defer f.branchChangeMu.Unlock()
		for i, cb := range f.branchChangeCallbacks {
			if &cb == &callback {
				f.branchChangeCallbacks = append(f.branchChangeCallbacks[:i], f.branchChangeCallbacks[i+1:]...)
				break
			}
		}
	}
}

// SetCwd updates the working directory and refreshes git info.
func (f *FooterDataProvider) SetCwd(cwd string) {
	if f.cwd == cwd {
		return
	}
	f.cwd = cwd
	f.cachedBranch.Store((*string)(nil))
	f.notifyBranchChange()
}

func (f *FooterDataProvider) notifyBranchChange() {
	f.branchChangeMu.Lock()
	callbacks := make([]func(), len(f.branchChangeCallbacks))
	copy(callbacks, f.branchChangeCallbacks)
	f.branchChangeMu.Unlock()

	for _, cb := range callbacks {
		cb()
	}
}

func (f *FooterDataProvider) resolveGitBranchSync() *string {
	gitPaths := FindGitPaths(f.cwd)
	if gitPaths == nil {
		return nil
	}

	content, err := os.ReadFile(gitPaths.HeadPath)
	if err != nil {
		return nil
	}

	headContent := strings.TrimSpace(string(content))
	if strings.HasPrefix(headContent, "ref: refs/heads/") {
		branch := headContent[16:]
		if branch == ".invalid" {
			resolved := resolveBranchWithGitSync(gitPaths.RepoDir)
			if resolved != nil {
				return resolved
			}
			detached := "detached"
			return &detached
		}
		return &branch
	}

	detached := "detached"
	return &detached
}

func resolveBranchWithGitSync(repoDir string) *string {
	cmd := exec.Command("git", "--no-optional-locks", "symbolic-ref", "--quiet", "--short", "HEAD")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return nil
	}
	return &branch
}
