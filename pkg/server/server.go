package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

// Server represents the HTTP web server for the Pi Web UI.
type Server struct {
	mux       *http.ServeMux
	staticDir string
}

// NewServer initializes a new Web UI Server with default routes.
// It accepts a path to the static web-ui dist directory.
func NewServer(staticDir string) *Server {
	if staticDir == "" {
		staticDir = "packages/web-ui/dist" // Default legacy workspace fallback
	}

	mux := http.NewServeMux()
	s := &Server{
		mux:       mux,
		staticDir: staticDir,
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
