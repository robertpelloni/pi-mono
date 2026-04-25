package server

import (
	"encoding/json"
	"net/http"
)

// Server represents the HTTP web server for the Pi Web UI.
type Server struct {
	mux *http.ServeMux
}

// NewServer initializes a new Web UI Server with default routes.
func NewServer() *Server {
	mux := http.NewServeMux()
	s := &Server{mux: mux}
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
	// In the future, this will serve the static built React/Vite assets from packages/web-ui/dist
	s.mux.HandleFunc("/", s.handleStaticStub())
}

// handleHealth returns a basic status OK JSON response.
func (s *Server) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

// handleStaticStub is a temporary placeholder for serving static UI assets.
func (s *Server) handleStaticStub() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><h1>Pi Web UI (Go Port)</h1></body></html>"))
	}
}
