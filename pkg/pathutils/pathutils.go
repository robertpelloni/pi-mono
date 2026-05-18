package pathutils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Unicode space characters that macOS uses in screenshot filenames
// These are handled in isUnicodeSpace() below.

const narrowNoBreakSpace = "\u202F"

// normalizeUnicodeSpaces replaces Unicode space characters with regular spaces.
func normalizeUnicodeSpaces(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if isUnicodeSpace(r) {
			b.WriteRune(' ')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isUnicodeSpace(r rune) bool {
	switch r {
	case '\u00A0', '\u2000', '\u2001', '\u2002', '\u2003', '\u2004',
		'\u2005', '\u2006', '\u2007', '\u2008', '\u2009', '\u200A',
		'\u202F', '\u205F', '\u3000':
		return true
	}
	return false
}

// tryMacOSScreenshotPath replaces space before AM/PM with narrow no-break space.
func tryMacOSScreenshotPath(filePath string) string {
	result := strings.ReplaceAll(filePath, " AM.", narrowNoBreakSpace+"AM.")
	result = strings.ReplaceAll(result, " PM.", narrowNoBreakSpace+"PM.")
	return result
}

// tryNFDVariant returns the NFD (decomposed) form of the path.
func tryNFDVariant(filePath string) string {
	// Go doesn't have built-in NFC/NFD normalization
	// On macOS, filenames are stored in NFD form
	// This is a simplified implementation
	return filePath // TODO: implement proper NFD normalization
}

// tryCurlyQuoteVariant replaces straight apostrophe with right single quotation mark.
func tryCurlyQuoteVariant(filePath string) string {
	return strings.ReplaceAll(filePath, "'", "\u2019")
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// normalizeAtPrefix removes @ prefix from file paths.
func normalizeAtPrefix(filePath string) string {
	if strings.HasPrefix(filePath, "@") {
		return filePath[1:]
	}
	return filePath
}

// ExpandPath expands ~ and normalizes Unicode spaces in a file path.
func ExpandPath(filePath string) string {
	normalized := normalizeUnicodeSpaces(normalizeAtPrefix(filePath))
	if normalized == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(normalized, "~/") {
		home, _ := os.UserHomeDir()
		return home + normalized[1:]
	}
	return normalized
}

// ResolveToCwd resolves a path relative to the current working directory.
func ResolveToCwd(path, cwd string) string {
	expanded := ExpandPath(path)
	if filepath.IsAbs(expanded) {
		return expanded
	}
	return filepath.Join(cwd, expanded)
}

// ResolveReadPath resolves a file path for reading, trying macOS variants.
func ResolveReadPath(filePath, cwd string) string {
	resolved := ResolveToCwd(filePath, cwd)
	if fileExists(resolved) {
		return resolved
	}

	// Try macOS AM/PM variant
	amPmVariant := tryMacOSScreenshotPath(resolved)
	if amPmVariant != resolved && fileExists(amPmVariant) {
		return amPmVariant
	}

	// Try NFD variant
	nfdVariant := tryNFDVariant(resolved)
	if nfdVariant != resolved && fileExists(nfdVariant) {
		return nfdVariant
	}

	// Try curly quote variant
	curlyVariant := tryCurlyQuoteVariant(resolved)
	if curlyVariant != resolved && fileExists(curlyVariant) {
		return curlyVariant
	}

	// Try combined NFD + curly quote
	nfdCurlyVariant := tryCurlyQuoteVariant(nfdVariant)
	if nfdCurlyVariant != resolved && fileExists(nfdCurlyVariant) {
		return nfdCurlyVariant
	}

	return resolved
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
	if dir := os.Getenv("PI_AGENT_DIR"); dir != "" {
		return dir
	}
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
