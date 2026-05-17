package gitutil

import (
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
)

// GitSource represents a parsed git URL.
type GitSource struct {
	Type   string `json:"type"`   // Always "git"
	Repo   string `json:"repo"`  // Clone URL
	Host   string `json:"host"`  // e.g., "github.com"
	Path   string `json:"path"`  // e.g., "user/repo"
	Ref    string `json:"ref"`   // Branch/tag/commit
	Pinned bool   `json:"pinned"` // True if ref was specified
}

// ParseGitURL parses a git URL into its components.
func ParseGitURL(source string) (*GitSource, error) {
	trimmed := strings.TrimSpace(source)
	hasGitPrefix := strings.HasPrefix(trimmed, "git:")
	urlStr := trimmed
	if hasGitPrefix {
		urlStr = strings.TrimSpace(urlStr[4:])
	}

	if !hasGitPrefix && !isProtocolURL(urlStr) {
		return nil, nil
	}

	repo, ref := splitRef(urlStr)

	// Try to parse as URL
	if isProtocolURL(repo) {
		parsed, err := url.Parse(repo)
		if err == nil {
			host := parsed.Hostname()
			path := strings.TrimPrefix(parsed.Path, "/")
			path = strings.TrimSuffix(path, ".git")
			path = strings.TrimPrefix(path, "/")

			if host != "" && path != "" && strings.Contains(path, "/") {
				return &GitSource{
					Type:   "git",
					Repo:   repo,
					Host:   host,
					Path:   path,
					Ref:    ref,
					Pinned: ref != "",
				}, nil
			}
		}
	}

	// Try SCP-like syntax
	scpPattern := regexp.MustCompile(`^git@([^:]+):(.+)$`)
	if scpPattern.MatchString(repo) {
		parts := scpPattern.FindStringSubmatch(repo)
		host := parts[1]
		path := strings.TrimSuffix(parts[2], ".git")
		if host != "" && path != "" && strings.Contains(path, "/") {
			return &GitSource{
				Type:   "git",
				Repo:   repo,
				Host:   host,
				Path:   path,
				Ref:    ref,
				Pinned: ref != "",
			}, nil
		}
	}

	// Try generic shorthand (host/path)
	slashIdx := strings.Index(repo, "/")
	if slashIdx > 0 && strings.Contains(repo[slashIdx:], "/") {
		host := repo[:slashIdx]
		path := repo[slashIdx+1:]
		path = strings.TrimSuffix(path, ".git")
		if strings.Contains(host, ".") || host == "localhost" {
			httpsRepo := "https://" + repo
			return &GitSource{
				Type:   "git",
				Repo:   httpsRepo,
				Host:   host,
				Path:   path,
				Ref:    ref,
				Pinned: ref != "",
			}, nil
		}
	}

	return nil, nil
}

func isProtocolURL(s string) bool {
	return strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "ssh://") ||
		strings.HasPrefix(s, "git://")
}

func splitRef(urlStr string) (repo string, ref string) {
	// Handle SCP-like syntax with ref
	scpPattern := regexp.MustCompile(`^git@([^:]+):(.+)$`)
	if scpPattern.MatchString(urlStr) {
		parts := scpPattern.FindStringSubmatch(urlStr)
		pathPart := parts[2]
		atIdx := strings.LastIndex(pathPart, "@")
		if atIdx > 0 {
			repoPath := pathPart[:atIdx]
			refPart := pathPart[atIdx+1:]
			if repoPath != "" && refPart != "" {
				return "git@" + parts[1] + ":" + repoPath, refPart
			}
		}
		return urlStr, ""
	}

	// Handle URL with ref
	if isProtocolURL(urlStr) {
		parsed, err := url.Parse(urlStr)
		if err == nil {
			pathPart := strings.TrimPrefix(parsed.Path, "/")
			atIdx := strings.LastIndex(pathPart, "@")
			if atIdx > 0 {
				repoPath := pathPart[:atIdx]
				refPart := pathPart[atIdx+1:]
				if repoPath != "" && refPart != "" {
					parsed.Path = "/" + repoPath
					return parsed.String(), refPart
				}
			}
		}
		return urlStr, ""
	}

	return urlStr, ""
}

// GetCurrentBranch returns the current git branch name.
// Returns empty string if not in a git repo.
func GetCurrentBranch(cwd string) string {
	cmd := exec.Command("git", "--no-optional-locks", "symbolic-ref", "--quiet", "--short", "HEAD")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// IsGitRepo checks if the given directory is inside a git repository.
func IsGitRepo(cwd string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = cwd
	err := cmd.Run()
	return err == nil
}

// GetGitRemoteURL returns the URL of the origin remote.
func GetGitRemoteURL(cwd string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// Ensure unused import doesn't break
var _ = fmt.Sprintf
