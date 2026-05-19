package interactive

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/badlogic/pi-mono/pkg/agentsession"
	"github.com/badlogic/pi-mono/pkg/sessionruntime"
	"github.com/badlogic/pi-mono/pkg/slashcommands"
	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// Interactive Mode Types
// ---------------------------------------------------------------------------

// InteractiveModeOptions configures the interactive session.
type InteractiveModeOptions struct {
	// Providers that were migrated to auth.json (shows warning)
	MigratedProviders []string
	// Warning message if session model couldn't be restored
	ModelFallbackMessage *string
	// Initial message to send on startup
	InitialMessage *string
	// Additional messages to send after the initial message
	InitialMessages []string
	// Force verbose startup
	Verbose bool
	// Output writer (defaults to os.Stdout)
	Writer io.Writer
	// Error writer (defaults to os.Stderr)
	ErrWriter io.Writer
}

// CompactionQueuedMessage represents a message queued during compaction.
type CompactionQueuedMessage struct {
	Text string
	Mode string // "steer" or "followUp"
}

// InteractiveMode handles the main interactive REPL loop.
// It delegates business logic to AgentSession and handles
// event rendering, slash command dispatch, and user input.
type InteractiveMode struct {
	runtime    *sessionruntime.AgentSessionRuntime
	session    *agentsession.AgentSession
	slashReg   *slashcommands.Registry
	options    InteractiveModeOptions
	writer     io.Writer
	errWriter  io.Writer

	mu         sync.Mutex
	quitting   bool
	isRunning  bool

	// Compaction queue
	compactionQueue []CompactionQueuedMessage

	// Event listener cancel
	unsubscribe func()
}

// NewInteractiveMode creates a new interactive mode.
func NewInteractiveMode(runtime *sessionruntime.AgentSessionRuntime, options InteractiveModeOptions) *InteractiveMode {
	w := options.Writer
	if w == nil {
		w = os.Stdout
	}
	ew := options.ErrWriter
	if ew == nil {
		ew = os.Stderr
	}

	var sess *agentsession.AgentSession
	if runtime != nil && runtime.AgentSession() != nil {
		sess = runtime.AgentSession()
	}

	return &InteractiveMode{
		runtime:    runtime,
		session:    sess,
		options:    options,
		writer:     w,
		errWriter:  ew,
	}
}

// Run starts the interactive loop. Blocks until the user quits.
func (im *InteractiveMode) Run(ctx context.Context) error {
	if im.session == nil {
		return fmt.Errorf("no agent session available")
	}

	im.isRunning = true
	defer func() { im.isRunning = false }()

	// Subscribe to session events
	im.unsubscribe = im.session.Subscribe(im.handleEvent)
	defer func() {
		if im.unsubscribe != nil {
			im.unsubscribe()
		}
	}()

	// Show startup warnings
	im.showStartupWarnings()

	// Handle SIGINT for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		im.handleCtrlC()
	}()

	// Process initial messages
	if im.options.InitialMessage != nil && *im.options.InitialMessage != "" {
		if err := im.session.Prompt(ctx, *im.options.InitialMessage); err != nil {
			fmt.Fprintf(im.errWriter, "Error: %v\n", err)
		}
	}
	for _, msg := range im.options.InitialMessages {
		if err := im.session.Prompt(ctx, msg); err != nil {
			fmt.Fprintf(im.errWriter, "Error: %v\n", err)
		}
	}

	// Main interactive loop — read from stdin
	im.printf("\n%s\n", styleHeader.Render("═══ Pi-Go Interactive Mode ═══"))
	if m := im.session.Model(); m.ID != "" {
		im.printf("%s\n", styleSystem.Render(fmt.Sprintf("Model: %s/%s", m.Provider, m.ID)))
	}
	im.printf("%s\n\n", styleSystem.Render("Type your message, / for commands, ! for bash, Ctrl+C to quit"))

	scanner := newLineScanner(im.reader())
	for scanner.scan() {
		if im.quitting {
			break
		}

		text := strings.TrimSpace(scanner.text())
		if text == "" {
			continue
		}

		if err := im.processInput(ctx, text); err != nil {
			if err == errQuit {
				break
			}
			fmt.Fprintf(im.errWriter, "Error: %v\n", err)
		}
	}

	return im.shutdown()
}

// processInput handles a single line of user input.
func (im *InteractiveMode) processInput(ctx context.Context, text string) error {
	// Handle quit commands
	if text == "/quit" || text == "/exit" {
		return errQuit
	}

	// Handle slash commands
	if strings.HasPrefix(text, "/") {
		return im.handleSlashCommand(ctx, text)
	}

	// Handle bash command (! prefix)
	if strings.HasPrefix(text, "!") {
		isExcluded := strings.HasPrefix(text, "!!")
		command := text
		if isExcluded {
			command = strings.TrimPrefix(text, "!!")
		} else {
			command = strings.TrimPrefix(text, "!")
		}
		command = strings.TrimSpace(command)
		if command == "" {
			return nil
		}
		return im.handleBashCommand(ctx, command, isExcluded)
	}

	// Queue input during compaction
	if im.session.IsCompacting() {
		im.mu.Lock()
		im.compactionQueue = append(im.compactionQueue, CompactionQueuedMessage{
			Text: text,
			Mode: "steer",
		})
		im.mu.Unlock()
		im.printf("%s\n", styleSystem.Render("(Queued — compaction in progress)"))
		return nil
	}

	// If streaming, use steer behavior
	if im.session.IsStreaming() {
		im.printf("%s\n", styleSystem.Render("(Steering current response...)"))
		return im.session.Prompt(ctx, text)
	}

	// Normal message
	return im.session.Prompt(ctx, text)
}

// handleSlashCommand processes a slash command.
func (im *InteractiveMode) handleSlashCommand(ctx context.Context, text string) error {
	parts := strings.SplitN(text, " ", 2)
	cmd := parts[0]
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	switch cmd {
	case "/help":
		im.showHelp()
		return nil

	case "/new":
		im.printf("%s\n", styleSystem.Render("Creating new session..."))
		return im.runtime.NewSession(nil)

	case "/compact":
		im.printf("%s\n", styleCompaction.Render("Starting compaction..."))
		return im.session.Compact(ctx)

	case "/reload":
		im.printf("%s\n", styleSystem.Render("Reloading..."))
		return im.session.Reload()

	case "/model":
		if args == "" {
			m := im.session.Model()
			im.printf("Current model: %s/%s\n", m.Provider, m.ID)
			return nil
		}
		im.printf("%s\n", styleSystem.Render(fmt.Sprintf("Switching model to: %s", args)))
		return im.session.SwitchModel(args)

	case "/clear":
		im.printf("\033[2J\033[H") // Clear screen
		return nil

	case "/quit", "/exit":
		return errQuit

	case "/resume":
		im.printf("%s\n", styleSystem.Render("Session resume not yet implemented in terminal mode"))
		return nil

	case "/debug":
		im.showDebugInfo()
		return nil
	}

	// Try the slash command registry
	if im.slashReg != nil {
		result, isCommand, err := im.slashReg.Execute(text)
		if isCommand {
			if err != nil {
				fmt.Fprintf(im.errWriter, "%s\n", styleError.Render(fmt.Sprintf("Error: %v", err)))
				return nil
			}
			im.handleSlashResult(ctx, result)
			return nil
		}
	}

	im.printf("%s\n", styleError.Render(fmt.Sprintf("Unknown command: %s. Type /help for available commands.", cmd)))
	return nil
}

// handleBashCommand executes a bash command.
func (im *InteractiveMode) handleBashCommand(ctx context.Context, command string, excluded bool) error {
	im.printf("%s\n", styleTool.Render(fmt.Sprintf("! %s", command)))

	result, err := im.session.ExecuteBash(ctx, command)
	if err != nil {
		fmt.Fprintf(im.errWriter, "%s\n", styleError.Render(fmt.Sprintf("Bash error: %v", err)))
		return nil
	}

	if result != "" {
		im.printf("%s\n", result)
	}
	return nil
}

// handleEvent processes AgentSession events.
func (im *InteractiveMode) handleEvent(event agentsession.AgentSessionEvent) {
	switch event.Type {
	case "agent_start":
		im.printf("%s\n", styleSystem.Render("⏳ Working..."))

	case "message_start":
		if msg, ok := event.Data.(map[string]interface{}); ok {
			if role, ok := msg["role"].(string); ok && role == "assistant" {
				im.printf("%s\n", styleAssistant.Render("🤖 "))
			}
		}

	case "message_update":
		// Streaming updates are handled by the agent event layer

	case "message_end":
		im.printf("\n")

	case "tool_execution_start":
		if data, ok := event.Data.(map[string]interface{}); ok {
			name, _ := data["toolName"].(string)
			im.printf("%s\n", styleTool.Render(fmt.Sprintf("  🔧 Running: %s", name)))
		}

	case "tool_execution_end":
		if data, ok := event.Data.(map[string]interface{}); ok {
			isErr, _ := data["isError"].(bool)
			if isErr {
				im.printf("%s\n", styleError.Render("  ✗ Tool failed"))
			} else {
				im.printf("%s\n", styleTool.Render("  ✓ Tool completed"))
			}
		}

	case "agent_end":
		im.printf("%s\n", styleSystem.Render("✓ Done."))

	case "compaction_start":
		im.printf("%s\n", styleCompaction.Render("📦 Compacting context... (Esc to cancel)"))

	case "compaction_end":
		if data, ok := event.Data.(map[string]interface{}); ok {
			if aborted, _ := data["aborted"].(bool); aborted {
				im.printf("%s\n", styleSystem.Render("Compaction cancelled"))
			} else {
				im.printf("%s\n", styleCompaction.Render("✓ Compaction complete"))
				im.flushCompactionQueue()
			}
		}

	case "auto_retry_start":
		if data, ok := event.Data.(map[string]interface{}); ok {
			attempt, _ := data["attempt"].(int)
			maxAttempts, _ := data["maxAttempts"].(int)
			delayMs, _ := data["delayMs"].(int)
			im.printf("%s\n", styleRetry.Render(fmt.Sprintf("🔄 Retrying (%d/%d) in %ds...",
				attempt, maxAttempts, delayMs/1000)))
		}

	case "auto_retry_end":
		if data, ok := event.Data.(map[string]interface{}); ok {
			success, _ := data["success"].(bool)
			if !success {
				attempt, _ := data["attempt"].(int)
				finalErr, _ := data["finalError"].(string)
				im.printf("%s\n", styleError.Render(fmt.Sprintf("Retry failed after %d attempts: %s", attempt, finalErr)))
			}
		}

	case "model_select":
		if m := im.session.Model(); m.ID != "" {
			im.printf("%s\n", styleSystem.Render(fmt.Sprintf("Model: %s/%s", m.Provider, m.ID)))
		}

	case "new_session":
		im.printf("%s\n", styleSystem.Render("📄 New session started"))

	case "reload":
		im.printf("%s\n", styleSystem.Render("🔄 Reloaded"))
	}
}

// handleSlashResult processes a slash command result.
func (im *InteractiveMode) handleSlashResult(ctx context.Context, result slashcommands.SlashCommandResult) {
	if result.Error != "" {
		fmt.Fprintf(im.errWriter, "%s\n", styleError.Render(result.Error))
	}
	if result.Info != "" {
		im.printf("%s\n", styleSlashInfo.Render(result.Info))
	}
	if result.Message != "" {
		_ = im.session.Prompt(ctx, result.Message)
	}
	if result.SwitchModel != "" {
		_ = im.session.SwitchModel(result.SwitchModel)
	}
	if result.SwitchProvider != "" {
		_ = im.session.SwitchProvider(result.SwitchProvider)
	}
	if result.Compact {
		_ = im.session.Compact(ctx)
	}
	if result.Quit {
		im.quitting = true
	}
}

// flushCompactionQueue sends queued messages after compaction ends.
func (im *InteractiveMode) flushCompactionQueue() {
	im.mu.Lock()
	queue := im.compactionQueue
	im.compactionQueue = nil
	im.mu.Unlock()

	for _, msg := range queue {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		_ = im.session.Prompt(ctx, msg.Text)
		cancel()
	}
}

// handleCtrlC handles Ctrl+C — double-press quits.
func (im *InteractiveMode) handleCtrlC() {
	im.quitting = true
	if im.session != nil {
		im.session.Abort()
	}
}

// shutdown gracefully stops the interactive mode.
func (im *InteractiveMode) shutdown() error {
	im.printf("%s\n", styleSystem.Render("Goodbye!"))
	if im.runtime != nil {
		im.runtime.Dispose()
	}
	return nil
}

// showStartupWarnings displays warnings at startup.
func (im *InteractiveMode) showStartupWarnings() {
	if len(im.options.MigratedProviders) > 0 {
		im.printf("%s\n", styleError.Render(fmt.Sprintf(
			"Migrated credentials to auth.json: %s",
			strings.Join(im.options.MigratedProviders, ", "))))
	}
	if im.options.ModelFallbackMessage != nil {
		im.printf("%s\n", styleError.Render(*im.options.ModelFallbackMessage))
	}
}

// showHelp displays available commands.
func (im *InteractiveMode) showHelp() {
	help := `
Available Commands:
  /help     - Show this help message
  /new      - Create a new session
  /compact  - Compact conversation context
  /reload   - Reload settings, skills, and system prompt
  /model    - Show or switch model (/model provider/id)
  /clear    - Clear the screen
  /resume   - Resume a previous session
  /debug    - Show debug information
  /quit     - Exit the program

Bash Commands:
  !command  - Execute a bash command (included in context)
  !!command - Execute a bash command (excluded from context)

Keyboard:
  Ctrl+C    - Abort current operation / Quit
`
	im.printf("%s\n", help)
}

// showDebugInfo displays debug information.
func (im *InteractiveMode) showDebugInfo() {
	if im.session == nil {
		im.printf("No active session\n")
		return
	}
	m := im.session.Model()
	im.printf("Model: %s/%s\n", m.Provider, m.ID)
	im.printf("CWD: %s\n", im.runtime.CWD())
	im.printf("Streaming: %v\n", im.session.IsStreaming())
	im.printf("Compacting: %v\n", im.session.IsCompacting())
	if usage := im.session.GetContextUsage(); usage != nil {
		if usage.Tokens != nil && usage.ContextWindow > 0 {
			im.printf("Context: %d/%d tokens (%.1f%%)\n",
				*usage.Tokens, usage.ContextWindow,
				float64(*usage.Tokens)/float64(usage.ContextWindow)*100)
		}
	}
}

// printf writes formatted output.
func (im *InteractiveMode) printf(format string, args ...interface{}) {
	fmt.Fprintf(im.writer, format, args...)
}

// reader returns the stdin reader.
func (im *InteractiveMode) reader() io.Reader {
	return os.Stdin
}

// errQuit is returned when the user requests to quit.
var errQuit = fmt.Errorf("quit requested")

// ---------------------------------------------------------------------------
// Line Scanner (simplified bufio.Scanner wrapper)
// ---------------------------------------------------------------------------

type lineScanner struct {
	reader io.Reader
	line   string
	done   bool
}

func newLineScanner(r io.Reader) *lineScanner {
	return &lineScanner{reader: r}
}

func (s *lineScanner) scan() bool {
	if s.done {
		return false
	}
	buf := make([]byte, 1)
	var line []byte
	for {
		_, err := s.reader.Read(buf)
		if err != nil {
			s.done = true
			s.line = string(line)
			return len(s.line) > 0
		}
		if buf[0] == '\n' {
			s.line = string(line)
			return true
		}
		line = append(line, buf[0])
	}
}

func (s *lineScanner) text() string {
	return s.line
}

// ---------------------------------------------------------------------------
// Styles (lipgloss-based)
// ---------------------------------------------------------------------------


var (
	styleHeader     = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	styleAssistant  = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	styleTool       = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	styleError      = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	styleSystem     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleCompaction = lipgloss.NewStyle().Foreground(lipgloss.Color("228"))
	styleRetry      = lipgloss.NewStyle().Foreground(lipgloss.Color("213"))
	styleSlashInfo  = lipgloss.NewStyle().Foreground(lipgloss.Color("120"))
)
