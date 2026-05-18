package initialmessage

import (
	"testing"
)

func TestBuildInitialMessage_AllParts(t *testing.T) {
	stdin := "stdin content"
	fileText := "file content"
	messages := []string{"cli message"}

	result := BuildInitialMessage(messages, &fileText, nil, &stdin)
	if result.InitialMessage == nil {
		t.Fatal("Expected non-nil initial message")
	}
	expected := "stdin contentfile contentcli message"
	if *result.InitialMessage != expected {
		t.Errorf("Expected %q, got %q", expected, *result.InitialMessage)
	}
}

func TestBuildInitialMessage_NoParts(t *testing.T) {
	result := BuildInitialMessage(nil, nil, nil, nil)
	if result.InitialMessage != nil {
		t.Errorf("Expected nil for no input, got %v", result.InitialMessage)
	}
}

func TestBuildInitialMessage_OnlyStdin(t *testing.T) {
	stdin := "piped input"
	result := BuildInitialMessage(nil, nil, nil, &stdin)
	if result.InitialMessage == nil || *result.InitialMessage != "piped input" {
		t.Errorf("Expected 'piped input', got %v", result.InitialMessage)
	}
}

func TestBuildInitialMessage_OnlyFileText(t *testing.T) {
	fileText := "file content"
	result := BuildInitialMessage(nil, &fileText, nil, nil)
	if result.InitialMessage == nil || *result.InitialMessage != "file content" {
		t.Errorf("Expected 'file content', got %v", result.InitialMessage)
	}
}

func TestBuildInitialMessage_OnlyMessage(t *testing.T) {
	messages := []string{"cli message"}
	result := BuildInitialMessage(messages, nil, nil, nil)
	if result.InitialMessage == nil || *result.InitialMessage != "cli message" {
		t.Errorf("Expected 'cli message', got %v", result.InitialMessage)
	}
}

func TestBuildInitialMessage_WithImages(t *testing.T) {
	images := []map[string]interface{}{
		{"type": "image", "mimeType": "image/png", "data": "base64data"},
	}
	result := BuildInitialMessage([]string{"msg"}, nil, images, nil)
	if len(result.InitialImages) != 1 {
		t.Errorf("Expected 1 image, got %d", len(result.InitialImages))
	}
}
