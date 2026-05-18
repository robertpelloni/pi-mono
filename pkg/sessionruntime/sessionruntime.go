package sessionruntime

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/badlogic/pi-mono/pkg/session"
)

// DiagnosticType classifies non-fatal issues collected during runtime creation.
type DiagnosticType string

const (
	DiagInfo    DiagnosticType = "info"
	DiagWarning DiagnosticType = "warning"
	DiagError   DiagnosticType = "error"
)

// AgentSessionRuntimeDiagnostic represents a non-fatal issue.
type AgentSessionRuntimeDiagnostic struct {
	Type    DiagnosticType `json:"type"`
	Message string         `json:"message"`
}

// CreateAgentSessionRuntimeResult holds the result of creating a runtime.
type CreateAgentSessionRuntimeResult struct {
	Session            interface{} // *agentsession.AgentSession
	Services           AgentSessionServices
	Diagnostics        []AgentSessionRuntimeDiagnostic
	ModelFallbackMessage *string
}

// AgentSessionServices holds cwd-bound runtime services.
type AgentSessionServices struct {
	CWD      string `json:"cwd"`
	AgentDir string `json:"agentDir"`
	// AuthStorage, SettingsManager, ModelRegistry, ResourceLoader
	// would be typed here when those packages are fully integrated
	Diagnostics []AgentSessionRuntimeDiagnostic `json:"diagnostics"`
}

// CreateAgentSessionRuntimeFactory creates a full runtime for a target cwd.
type CreateAgentSessionRuntimeFactory func(options CreateAgentSessionRuntimeOptions) (*CreateAgentSessionRuntimeResult, error)

// CreateAgentSessionRuntimeOptions configures runtime creation.
type CreateAgentSessionRuntimeOptions struct {
	CWD               string
	AgentDir          string
	SessionManager    *session.SessionManager
	SessionStartEvent *SessionStartEvent
}

// SessionStartEvent is fired when a session starts.
type SessionStartEvent struct {
	Type               SessionStartReason `json:"type"`
	PreviousSessionFile *string            `json:"previousSessionFile,omitempty"`
}

// SessionStartReason indicates why a session started.
type SessionStartReason string

const (
	ReasonStartup SessionStartReason = "startup"
	ReasonReload  SessionStartReason = "reload"
	ReasonNew     SessionStartReason = "new"
	ReasonResume  SessionStartReason = "resume"
	ReasonFork    SessionStartReason = "fork"
)

// AgentSessionRuntime owns the current AgentSession plus its cwd-bound services.
type AgentSessionRuntime struct {
	session             interface{} // *agentsession.AgentSession
	services            AgentSessionServices
	createRuntime       CreateAgentSessionRuntimeFactory
	diagnostics         []AgentSessionRuntimeDiagnostic
	modelFallbackMessage *string
}

// NewAgentSessionRuntime creates a new runtime wrapper.
func NewAgentSessionRuntime(
	sess interface{},
	services AgentSessionServices,
	createRuntime CreateAgentSessionRuntimeFactory,
	diagnostics []AgentSessionRuntimeDiagnostic,
	modelFallbackMessage *string,
) *AgentSessionRuntime {
	return &AgentSessionRuntime{
		session:             sess,
		services:            services,
		createRuntime:       createRuntime,
		diagnostics:         diagnostics,
		modelFallbackMessage: modelFallbackMessage,
	}
}

// Services returns the cwd-bound services.
func (r *AgentSessionRuntime) Services() *AgentSessionServices {
	return &r.services
}

// Session returns the current agent session.
func (r *AgentSessionRuntime) Session() interface{} {
	return r.session
}

// CWD returns the current working directory.
func (r *AgentSessionRuntime) CWD() string {
	return r.services.CWD
}

// Diagnostics returns collected diagnostics.
func (r *AgentSessionRuntime) Diagnostics() []AgentSessionRuntimeDiagnostic {
	return r.diagnostics
}

// ModelFallbackMessage returns the model fallback message if any.
func (r *AgentSessionRuntime) ModelFallbackMessage() *string {
	return r.modelFallbackMessage
}

// SwitchSession switches to a different session file.
func (r *AgentSessionRuntime) SwitchSession(sessionPath string, cwdOverride *string) error {
	sm := session.OpenSession(sessionPath, r.services.AgentDir, cwdOverride)
	// Verify CWD exists
	if sm.GetCWD() == "" {
		// Could set to current CWD
	}

	result, err := r.createRuntime(CreateAgentSessionRuntimeOptions{
		CWD:            sm.GetCWD(),
		AgentDir:       r.services.AgentDir,
		SessionManager: sm,
		SessionStartEvent: &SessionStartEvent{
			Type: ReasonResume,
		},
	})
	if err != nil {
		return err
	}

	r.apply(result)
	return nil
}

// NewSession creates a new session.
func (r *AgentSessionRuntime) NewSession(parentSession *string) error {
	sessionDir := "" // Would get from session manager
	sm := session.CreateSession(r.services.CWD, sessionDir)
	if parentSession != nil {
		sm.NewSession(&session.NewSessionOptions{ParentSession: parentSession})
	}

	result, err := r.createRuntime(CreateAgentSessionRuntimeOptions{
		CWD:            r.services.CWD,
		AgentDir:       r.services.AgentDir,
		SessionManager: sm,
		SessionStartEvent: &SessionStartEvent{
			Type: ReasonNew,
		},
	})
	if err != nil {
		return err
	}

	r.apply(result)
	return nil
}

// ImportFromJsonl imports a session from a JSONL file.
func (r *AgentSessionRuntime) ImportFromJsonl(inputPath string, cwdOverride *string) error {
	resolvedPath, _ := filepath.Abs(inputPath)
	if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", resolvedPath)
	}

	sessionDir := "" // Would get from session manager
	sm := session.OpenSession(resolvedPath, sessionDir, cwdOverride)

	result, err := r.createRuntime(CreateAgentSessionRuntimeOptions{
		CWD:            sm.GetCWD(),
		AgentDir:       r.services.AgentDir,
		SessionManager: sm,
		SessionStartEvent: &SessionStartEvent{
			Type: ReasonResume,
		},
	})
	if err != nil {
		return err
	}

	r.apply(result)
	return nil
}

// Dispose shuts down the runtime.
func (r *AgentSessionRuntime) Dispose() {
	// Would emit session_shutdown event and dispose agent session
}

func (r *AgentSessionRuntime) apply(result *CreateAgentSessionRuntimeResult) {
	if cwd := result.Services.CWD; cwd != "" {
		if wd, _ := os.Getwd(); wd != cwd {
			os.Chdir(cwd)
		}
	}
	r.session = result.Session
	r.services = result.Services
	r.diagnostics = result.Diagnostics
	r.modelFallbackMessage = result.ModelFallbackMessage
}

// CreateAgentSessionRuntime creates the initial runtime.
func CreateAgentSessionRuntime(
	createRuntime CreateAgentSessionRuntimeFactory,
	options CreateAgentSessionRuntimeOptions,
) (*AgentSessionRuntime, error) {
	result, err := createRuntime(options)
	if err != nil {
		return nil, err
	}

	if cwd := result.Services.CWD; cwd != "" {
		if wd, _ := os.Getwd(); wd != cwd {
			os.Chdir(cwd)
		}
	}

	return NewAgentSessionRuntime(
		result.Session,
		result.Services,
		createRuntime,
		result.Diagnostics,
		result.ModelFallbackMessage,
	), nil
}

