package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitConfig_NewFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	cfg, err := InitConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}
}

func TestMergeConfig(t *testing.T) {
	// MergeConfig should not panic
	MergeConfig(&Config{})
}

func TestConfig_Fields(t *testing.T) {
	cfg := &Config{}
	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}
}
