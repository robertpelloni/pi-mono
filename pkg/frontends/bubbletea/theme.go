package bubbletea

import "github.com/charmbracelet/lipgloss"

// TUITheme defines the color palette for the terminal interface.
type TUITheme struct {
	Name string

	User       lipgloss.Color
	Assistant  lipgloss.Color
	Tool       lipgloss.Color
	Error      lipgloss.Color
	System     lipgloss.Color
	Thinking   lipgloss.Color
	SlashInfo  lipgloss.Color
	Header     lipgloss.Color
	Compaction lipgloss.Color
	Retry      lipgloss.Color

	Background      lipgloss.Color
	Foreground      lipgloss.Color
	Muted           lipgloss.Color
	SelectionBg    lipgloss.Color
	DiffAddBg      lipgloss.Color
	DiffDelBg      lipgloss.Color
	ToolPendingBg  lipgloss.Color
	ToolSuccessBg  lipgloss.Color
	ToolErrorBg    lipgloss.Color
}

var DarkTheme = TUITheme{
	Name: "dark",

	User:       lipgloss.Color("205"),
	Assistant:  lipgloss.Color("39"),
	Tool:       lipgloss.Color("214"),
	Error:      lipgloss.Color("196"),
	System:     lipgloss.Color("240"),
	Thinking:   lipgloss.Color("63"),
	SlashInfo:  lipgloss.Color("120"),
	Header:     lipgloss.Color("86"),
	Compaction: lipgloss.Color("228"),
	Retry:      lipgloss.Color("213"),

	Background:     lipgloss.Color("234"),
	Foreground:     lipgloss.Color("250"),
	Muted:          lipgloss.Color("244"),
	SelectionBg:    lipgloss.Color("236"),
	DiffAddBg:      lipgloss.Color("22"),
	DiffDelBg:      lipgloss.Color("52"),
	ToolPendingBg:  lipgloss.Color("235"),
	ToolSuccessBg:  lipgloss.Color("236"),
	ToolErrorBg:    lipgloss.Color("52"),
}

var LightTheme = TUITheme{
	Name: "light",

	User:       lipgloss.Color("125"),
	Assistant:  lipgloss.Color("26"),
	Tool:       lipgloss.Color("130"),
	Error:      lipgloss.Color("124"),
	System:     lipgloss.Color("246"),
	Thinking:   lipgloss.Color("55"),
	SlashInfo:  lipgloss.Color("28"),
	Header:     lipgloss.Color("25"),
	Compaction: lipgloss.Color("94"),
	Retry:      lipgloss.Color("90"),

	Background:     lipgloss.Color("255"),
	Foreground:     lipgloss.Color("235"),
	Muted:          lipgloss.Color("248"),
	SelectionBg:    lipgloss.Color("252"),
	DiffAddBg:      lipgloss.Color("194"),
	DiffDelBg:      lipgloss.Color("224"),
	ToolPendingBg:  lipgloss.Color("253"),
	ToolSuccessBg:  lipgloss.Color("194"),
	ToolErrorBg:    lipgloss.Color("224"),
}

var CurrentTheme = DarkTheme

// Theme colors and styles for the TUI.
var (
	StyleUser       lipgloss.Style
	StyleAssistant  lipgloss.Style
	StyleToolTitle  lipgloss.Style
	StyleToolArgs   lipgloss.Style
	StyleToolOutput lipgloss.Style
	StyleError      lipgloss.Style
	StyleSystem     lipgloss.Style
	StyleThinking   lipgloss.Style
	StyleSlashInfo  lipgloss.Style
	StyleHeader     lipgloss.Style
	StyleCompaction lipgloss.Style
	StyleRetry      lipgloss.Style

	StyleToolPending lipgloss.Style
	StyleToolSuccess lipgloss.Style
	StyleToolError   lipgloss.Style

	StyleDiffAdd lipgloss.Style
	StyleDiffDel lipgloss.Style

	StyleCompletionItem     lipgloss.Style
	StyleCompletionSelected lipgloss.Style
	StyleCompletionHeader   lipgloss.Style
)

func init() {
	UpdateStyles(DarkTheme)
}

func UpdateStyles(theme TUITheme) {
	CurrentTheme = theme

	StyleUser = lipgloss.NewStyle().Foreground(theme.User).Bold(true)
	StyleAssistant = lipgloss.NewStyle().Foreground(theme.Assistant).Bold(true)
	StyleToolTitle = lipgloss.NewStyle().Foreground(theme.Tool).Bold(true)
	StyleToolArgs = lipgloss.NewStyle().Foreground(theme.Muted).Italic(true)
	StyleToolOutput = lipgloss.NewStyle().Foreground(theme.Foreground)
	StyleError = lipgloss.NewStyle().Foreground(theme.Error).Bold(true)
	StyleSystem = lipgloss.NewStyle().Foreground(theme.System)
	StyleThinking = lipgloss.NewStyle().Foreground(theme.Thinking).Italic(true)
	StyleSlashInfo = lipgloss.NewStyle().Foreground(theme.SlashInfo)
	StyleHeader = lipgloss.NewStyle().Foreground(theme.Header).Bold(true)
	StyleCompaction = lipgloss.NewStyle().Foreground(theme.Compaction)
	StyleRetry = lipgloss.NewStyle().Foreground(theme.Retry)

	StyleToolPending = lipgloss.NewStyle().Background(theme.ToolPendingBg).Foreground(theme.Tool)
	StyleToolSuccess = lipgloss.NewStyle().Background(theme.ToolSuccessBg).Foreground(theme.SlashInfo)
	StyleToolError = lipgloss.NewStyle().Background(theme.ToolErrorBg).Foreground(theme.Error)

	StyleDiffAdd = lipgloss.NewStyle().Foreground(theme.SlashInfo).Background(theme.DiffAddBg)
	StyleDiffDel = lipgloss.NewStyle().Foreground(theme.Error).Background(theme.DiffDelBg)

	StyleCompletionItem = lipgloss.NewStyle().Foreground(theme.Foreground)
	StyleCompletionSelected = lipgloss.NewStyle().Foreground(theme.Assistant).Bold(true).Background(theme.SelectionBg)
	StyleCompletionHeader = lipgloss.NewStyle().Foreground(theme.Header).Bold(true).Underline(true)
}
