package printmode

import (
	"bytes"
	"testing"

	"github.com/badlogic/pi-mono/pkg/ai"
)

func TestPrintModeOptions_Fields(t *testing.T) {
	opts := PrintModeOptions{
		Mode:           "text",
		Messages:       []string{"hello"},
		InitialMessage: "test",
		Writer:         &bytes.Buffer{},
	}
	if opts.Mode != "text" {
		t.Error("Mode mismatch")
	}
	if len(opts.Messages) != 1 {
		t.Error("Messages mismatch")
	}
	if opts.InitialMessage != "test" {
		t.Error("InitialMessage mismatch")
	}
}

func TestPrintModeOptions_DefaultMode(t *testing.T) {
	opts := PrintModeOptions{}
	if opts.Mode != "" {
		// Empty mode defaults to "text" in RunPrintMode
		t.Error("Expected empty default mode")
	}
}

func TestPrintModeOptions_WithImages(t *testing.T) {
	opts := PrintModeOptions{
		InitialImages: []ai.ImageContent{
			{Data: "base64data", MimeType: "image/png"},
		},
	}
	if len(opts.InitialImages) != 1 {
		t.Error("Expected 1 image")
	}
}
