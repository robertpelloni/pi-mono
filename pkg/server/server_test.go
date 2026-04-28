package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServer_HealthEndpoint(t *testing.T) {
	srv := NewServer("") // Uses default path

	req, err := http.NewRequest("GET", "/api/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to decode json response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%v'", response["status"])
	}
}

func TestServer_StaticAssetEndpoint(t *testing.T) {
	// Create a temporary directory and an index.html file to test static serving
	tempDir, err := os.MkdirTemp("", "pi-server-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	expectedBody := "<html><body><h1>Test SPA</h1></body></html>"
	indexPath := filepath.Join(tempDir, "index.html")
	err = os.WriteFile(indexPath, []byte(expectedBody), 0644)
	if err != nil {
		t.Fatal(err)
	}

	srv := NewServer(tempDir)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if rr.Body.String() != expectedBody {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expectedBody)
	}
}

func TestServer_SPARoutingEndpoint(t *testing.T) {
	// Ensure that requesting a non-existent path falls back to index.html
	tempDir, err := os.MkdirTemp("", "pi-server-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	expectedBody := "<html><body><h1>Test SPA Fallback</h1></body></html>"
	indexPath := filepath.Join(tempDir, "index.html")
	err = os.WriteFile(indexPath, []byte(expectedBody), 0644)
	if err != nil {
		t.Fatal(err)
	}

	srv := NewServer(tempDir)

	req, err := http.NewRequest("GET", "/some/nonexistent/client/route", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if rr.Body.String() != expectedBody {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expectedBody)
	}
}
