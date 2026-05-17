package footerdata

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// FooterDataProvider provides git branch and extension statuses.
type FooterDataProvider struct {
	mu                sync.RWMutex
	cwd               string
	cachedBranch      string // "" = not in repo, "detached" = detached HEAD, else branch name
	gitPaths          *gitPaths
	extensionStatuses map[string]string
	availableProviders int

	onBranchChange []func()
}

type gitPaths struct {
	repoDir      string
	commonGitDir string
	headPath     string
}

// NewFooterDataProvider creates a new footer data provider.
func NewFooterDataProvider(cwd string) *FooterDataProvider {
	p := &FooterDataProvider{
		cwd:               cwd,
		extensionStatuses: make(map[string]string),
	}
	p.gitPaths = p.findGitPaths(cwd)
	p.cachedBranch = p.resolveGitBranch()
	return p
}

// GetGitBranch returns the current git branch.
// Returns "" if not in a repo, "detached" if detached HEAD.
func (p *FooterDataProvider) GetGitBranch() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cachedBranch
}

// GetExtensionStatuses returns extension status texts.
func (p *FooterDataProvider) GetExtensionStatuses() map[string]string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make(map[string]string)
	for k, v := range p.extensionStatuses {
		result[k] = v
	}
	return result
}

// SetExtensionStatus sets an extension status.
func (p *FooterDataProvider) SetExtensionStatus(key, text string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if text == "" {
		delete(p.extensionStatuses, key)
	} else {
		p.extensionStatuses[key] = text
	}
}

// GetAvailableProviderCount returns the number of available providers.
func (p *FooterDataProvider) GetAvailableProviderCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.availableProviders
}

// SetAvailableProviderCount sets the available provider count.
func (p *FooterDataProvider) SetAvailableProviderCount(count int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.availableProviders = count
}

// OnBranchChange subscribes to git branch changes. Returns unsubscribe function.
func (p *FooterDataProvider) OnBranchChange(callback func()) func() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onBranchChange = append(p.onBranchChange, callback)
	return func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		for i, cb := range p.onBranchChange {
			if &cb == &callback {
				p.onBranchChange = append(p.onBranchChange[:i], p.onBranchChange[i+1:]...)
				break
			}
		}
	}
}

// SetCwd changes the current working directory.
func (p *FooterDataProvider) SetCwd(cwd string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cwd == cwd {
		return
	}
	p.cwd = cwd
	p.gitPaths = p.findGitPaths(cwd)
	p.cachedBranch = p.resolveGitBranch()
	for _, cb := range p.onBranchChange {
		cb()
	}
}

// RefreshBranch re-reads the git branch.
func (p *FooterDataProvider) RefreshBranch() {
	p.mu.Lock()
	defer p.mu.Unlock()
	newBranch := p.resolveGitBranch()
	if p.cachedBranch != newBranch {
		p.cachedBranch = newBranch
		for _, cb := range p.onBranchChange {
			cb()
		}
	}
}

// Dispose cleans up the provider.
func (p *FooterDataProvider) Dispose() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onBranchChange = nil
}

func (p *FooterDataProvider) findGitPaths(cwd string) *gitPaths {
	dir := cwd
	for {
		gitPath := filepath.Join(dir, ".git")
		info, err := os.Stat(gitPath)
		if err == nil {
			if info.IsDir() {
				headPath := filepath.Join(gitPath, "HEAD")
				if _, err := os.Stat(headPath); err == nil {
					return &gitPaths{
						repoDir:      dir,
						commonGitDir: gitPath,
						headPath:     headPath,
					}
				}
			} else {
				// Worktree: .git is a file
				content, err := os.ReadFile(gitPath)
				if err == nil {
					line := strings.TrimSpace(string(content))
					if strings.HasPrefix(line, "gitdir: ") {
						gitDir := filepath.Join(dir, strings.TrimSpace(line[8:]))
						headPath := filepath.Join(gitDir, "HEAD")
						if _, err := os.Stat(headPath); err == nil {
							return &gitPaths{
								repoDir:      dir,
								commonGitDir: gitDir,
								headPath:     headPath,
							}
						}
					}
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

func (p *FooterDataProvider) resolveGitBranch() string {
	if p.gitPaths == nil {
		return ""
	}

	// Read HEAD file
	content, err := os.ReadFile(p.gitPaths.headPath)
	if err != nil {
		return ""
	}

	line := strings.TrimSpace(string(content))
	if strings.HasPrefix(line, "ref: refs/heads/") {
		branch := line[16:]
		if branch == ".invalid" {
			// Try git command
			cmd := exec.Command("git", "--no-optional-locks", "symbolic-ref", "--quiet", "--short", "HEAD")
			cmd.Dir = p.gitPaths.repoDir
			out, err := cmd.Output()
			if err != nil {
				return "detached"
			}
			result := strings.TrimSpace(string(out))
			if result == "" {
				return "detached"
			}
			return result
		}
		return branch
	}

	return "detached"
}

// Ensure fmt and bufio are used
var _ = fmt.Sprintf
var _ = bufio.Scanner{}
