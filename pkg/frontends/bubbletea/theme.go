package bubbletea

import "github.com/charmbracelet/lipgloss"

// Theme colors and styles for the TUI.
var (
	ColorUser       = lipgloss.Color("205")
	ColorAssistant  = lipgloss.Color("39")
	ColorTool       = lipgloss.Color("214")
	ColorError      = lipgloss.Color("196")
	ColorSystem     = lipgloss.Color("240")
	ColorThinking   = lipgloss.Color("63")
	ColorSlashInfo  = lipgloss.Color("120")
	ColorHeader     = lipgloss.Color("86")
	ColorCompaction = lipgloss.Color("228")
	ColorRetry      = lipgloss.Color("213")

	StyleUser       = lipgloss.NewStyle().Foreground(ColorUser).Bold(true)
	StyleAssistant  = lipgloss.NewStyle().Foreground(ColorAssistant).Bold(true)
	StyleToolTitle  = lipgloss.NewStyle().Foreground(ColorTool).Bold(true)
	StyleToolArgs   = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true)
	StyleToolOutput = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	StyleError      = lipgloss.NewStyle().Foreground(ColorError).Bold(true)
	StyleSystem     = lipgloss.NewStyle().Foreground(ColorSystem)
	StyleThinking   = lipgloss.NewStyle().Foreground(ColorThinking).Italic(true)
	StyleSlashInfo  = lipgloss.NewStyle().Foreground(ColorSlashInfo)
	StyleHeader     = lipgloss.NewStyle().Foreground(ColorHeader).Bold(true)
	StyleCompaction = lipgloss.NewStyle().Foreground(ColorCompaction)
	StyleRetry      = lipgloss.NewStyle().Foreground(ColorRetry)

	// Tool status styles
	StyleToolPending = lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(ColorTool)
	StyleToolSuccess = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("120"))
	StyleToolError   = lipgloss.NewStyle().Background(lipgloss.Color("52")).Foreground(ColorError)

	// Diff styles
	StyleDiffAdd = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Background(lipgloss.Color("22"))
	StyleDiffDel = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Background(lipgloss.Color("52"))
)
