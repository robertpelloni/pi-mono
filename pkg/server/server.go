package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

// Server represents the HTTP web server for the Pi Web UI.
type Server struct {
	mux       *http.ServeMux
	staticDir string
	ag        *agent.Agent
}

// NewServer initializes a new Web UI Server with default routes.
// It accepts a path to the static web-ui dist directory and the core Agent.
func NewServer(staticDir string, ag *agent.Agent) *Server {
	if staticDir == "" {
		staticDir = "packages/web-ui/dist" // Default legacy workspace fallback
	}

	mux := http.NewServeMux()
	s := &Server{
		mux:       mux,
		staticDir: staticDir,
		ag:        ag,
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

	// Serve static files (React frontend)
	fileServer := http.FileServer(http.Dir(s.staticDir))
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Basic SPA routing: if file doesn't exist, serve index.html
		path := filepath.Join(s.staticDir, r.URL.Path)
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(s.staticDir, "index.html"))
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

// handleHealth returns a basic status OK JSON response.
func (s *Server) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

// chatRequest models the incoming prompt from the web frontend
type chatRequest struct {
	Message string `json:"message"`
}

// handleChat exposes a Server-Sent Events (SSE) endpoint driving the agent loop.
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

		if s.ag == nil {
			http.Error(w, "Agent not initialized", http.StatusInternalServerError)
			return
		}

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

		// Create a local channel to capture events for this specific request loop
		eventsChan := make(chan agent.AgentEvent, 100)

		// Subscribe the local channel to the global agent
		subFunc := func(e agent.AgentEvent) {
			eventsChan <- e
		}
		s.ag.Subscribe(subFunc)

		// Run prompt in background
		userMsg := ai.UserMessage{
			Content: []ai.Content{
				ai.TextContent{Text: req.Message},
			},
			Timestamp: time.Now().UnixMilli(),
		}

		go func() {
			err := s.ag.Prompt(r.Context(), userMsg)
			if err != nil {
				// Send an error event over the channel
				errMsg := err.Error()
				reason := ai.StopReasonError
				eventsChan <- agent.AgentEvent{
					Type: agent.EventMessageUpdate,
					AssistantMessageEvent: &ai.AssistantMessageEvent{
						Type: ai.EventError,
						Reason: &reason,
						Error: &ai.AssistantMessage{ErrorMessage: &errMsg},
					},
				}
			}
			// Let the client know execution finished so they can drop the connection
			eventsChan <- agent.AgentEvent{Type: agent.EventAgentEnd}
		}()

		// Stream events to HTTP client
		for {
			select {
			case <-r.Context().Done():
				return // Client disconnected
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
