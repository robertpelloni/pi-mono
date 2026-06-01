package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/agentsession"
	"github.com/badlogic/pi-mono/pkg/ai"
)

// Server represents the HTTP web server for the Pi Web UI.
type Server struct {
	mu        sync.RWMutex
	mux       *http.ServeMux
	staticDir string
	sessions  map[string]*agentsession.AgentSession
	config    agentsession.AgentSessionConfig // Template config for new sessions
}

// NewServer initializes a new Web UI Server with multi-session support.
func NewServer(staticDir string, config agentsession.AgentSessionConfig) *Server {
	if staticDir == "" {
		staticDir = "packages/web-ui/dist"
	}

	mux := http.NewServeMux()
	s := &Server{
		mux:       mux,
		staticDir: staticDir,
		sessions:  make(map[string]*agentsession.AgentSession),
		config:    config,
	}
	s.routes()
	return s
}

// ServeHTTP implements the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// routes registers all the HTTP endpoints.
func (s *Server) routes() {
	s.mux.HandleFunc("/api/health", s.handleHealth())
	s.mux.HandleFunc("/api/chat", s.handleChat())
	s.mux.HandleFunc("/api/sessions", s.handleListSessions())

	// Serve static files
	fileServer := http.FileServer(http.Dir(s.staticDir))
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(s.staticDir, r.URL.Path)
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(s.staticDir, "index.html"))
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "0.86.0"})
	}
}

type chatRequest struct {
	SessionID string `json:"sessionId"`
	Message   string `json:"message"`
}

func (s *Server) handleChat() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		sess, ok := s.sessions[req.SessionID]
		if !ok {
			// Create new session if ID not found or empty
			if req.SessionID == "" {
				req.SessionID = fmt.Sprintf("sess_%d", time.Now().UnixNano())
			}
			// In a real implementation we would clone the template agent and session manager
			sess = agentsession.NewAgentSession(s.config)
			s.sessions[req.SessionID] = sess
		}
		s.mu.Unlock()

		// Setup SSE Headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		eventsChan := make(chan agent.AgentEvent, 100)
		unsubscribe := sess.Agent().Subscribe(func(e agent.AgentEvent) {
			eventsChan <- e
		})
		defer unsubscribe()

		go func() {
			err := sess.Prompt(r.Context(), req.Message)
			if err != nil {
				errMsg := err.Error()
				reason := ai.StopReasonError
				eventsChan <- agent.AgentEvent{
					Type: agent.EventMessageUpdate,
					AssistantMessageEvent: &ai.AssistantMessageEvent{
						Type:   ai.EventError,
						Reason: &reason,
						Error:  &ai.AssistantMessage{ErrorMessage: &errMsg},
					},
				}
			}
			eventsChan <- agent.AgentEvent{Type: agent.EventAgentEnd}
		}()

		for {
			select {
			case <-r.Context().Done():
				return
			case event := <-eventsChan:
				data, err := json.Marshal(event)
				if err == nil {
					fmt.Fprintf(w, "data: %s\n\n", string(data))
					flusher.Flush()
				}
				if event.Type == agent.EventAgentEnd {
					return
				}
			}
		}
	}
}

func (s *Server) handleListSessions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		ids := make([]string, 0, len(s.sessions))
		for id := range s.sessions {
			ids = append(ids, id)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"sessions": ids})
	}
}
