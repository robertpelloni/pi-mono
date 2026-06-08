package ai

import (
	"strings"
	"testing"
)

func TestExternalServices_Lynx(t *testing.T) {
	// Verify lynx is installed
	args := map[string]interface{}{"url": "https://example.com"}
	resp := handleHermesBrowserNavigate(args)

	if strings.Contains(resp, "Error") {
		t.Skip("lynx probably not installed or no internet access")
	}

	if !strings.Contains(resp, "Example Domain") {
		t.Errorf("unexpected lynx output: %s", resp)
	}
}

func TestExternalServices_WebSearch(t *testing.T) {
	args := map[string]interface{}{"query": "test query"}
	resp := handleHermesWebSearch(args)

	if strings.Contains(resp, "Error") {
		t.Skip("web search failed (lynx/network issues)")
	}

	// DuckDuckGo lite/html output usually contains 'test query' or 'DuckDuckGo'
	if !strings.Contains(strings.ToLower(resp), "test") && !strings.Contains(resp, "DuckDuckGo") {
		t.Errorf("unexpected web search output: %s", resp)
	}
}
