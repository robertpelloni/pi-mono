package bubbletea

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/lipgloss"
)

// HyperConfig parity schema for theme matching.
type HyperConfig struct {
	Config struct {
		FontSize        float64           `json:"fontSize"`
		FontFamily      string            `json:"fontFamily"`
		CursorColor     string            `json:"cursorColor"`
		ForegroundColor string            `json:"foregroundColor"`
		BackgroundColor string            `json:"backgroundColor"`
		Colors          map[string]string `json:"colors"`
		Padding         string            `json:"padding"`
	} `json:"config"`
}

// ApplyHyperTheme applies colors from a Hyper-compatible configuration.
func ApplyHyperTheme(rawConfig string) error {
	var hc HyperConfig
	if err := json.Unmarshal([]byte(rawConfig), &hc); err != nil {
		return fmt.Errorf("failed to parse hyper config: %v", err)
	}

	// Map Hyper colors to Pi styles
	if c, ok := hc.Config.Colors["black"]; ok {
		ColorSystem = lipgloss.Color(c)
		StyleSystem = StyleSystem.Foreground(ColorSystem)
	}
	if c, ok := hc.Config.Colors["red"]; ok {
		ColorError = lipgloss.Color(c)
		StyleError = StyleError.Foreground(ColorError)
	}
	if c, ok := hc.Config.Colors["green"]; ok {
		ColorSlashInfo = lipgloss.Color(c)
		StyleSlashInfo = StyleSlashInfo.Foreground(ColorSlashInfo)
	}
	if c, ok := hc.Config.Colors["blue"]; ok {
		ColorAssistant = lipgloss.Color(c)
		StyleAssistant = StyleAssistant.Foreground(ColorAssistant)
	}
	if c, ok := hc.Config.Colors["yellow"]; ok {
		ColorTool = lipgloss.Color(c)
		StyleToolTitle = StyleToolTitle.Foreground(ColorTool)
	}
	if c, ok := hc.Config.Colors["magenta"]; ok {
		ColorUser = lipgloss.Color(c)
		StyleUser = StyleUser.Foreground(ColorUser)
	}
	if c, ok := hc.Config.Colors["cyan"]; ok {
		ColorHeader = lipgloss.Color(c)
		StyleHeader = StyleHeader.Foreground(ColorHeader)
	}

	return nil
}
