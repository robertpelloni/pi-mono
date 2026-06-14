package browser

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
)

// OpenURL opens a URL in the default system browser.
// Only http/https URLs are allowed for security.
func OpenURL(rawURL string) error {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return fmt.Errorf("only http/https URLs are allowed")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Host == "" {
		return fmt.Errorf("URL must have a host")
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", parsed.String())
	case "windows":
		// Use rundll32 to avoid shell injection with cmd /c start
		cmd = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", parsed.String())
	default: // linux, others
		cmd = exec.Command("xdg-open", parsed.String())
	}
	if cmd == nil {
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}
