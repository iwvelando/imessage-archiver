package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary SSH key file for testing
	tmpKeyFile, err := os.CreateTemp("", "test-ssh-key-*")
	if err != nil {
		t.Fatalf("Failed to create temp SSH key file: %v", err)
	}
	defer os.Remove(tmpKeyFile.Name())
	tmpKeyFile.WriteString("test-ssh-key-content")
	tmpKeyFile.Close()

	// Create a temporary config file for testing
	configContent := `
logging_level: "debug"
date_from: "2023-01-01T00:00:00Z"
date_to: "2023-12-31T23:59:59Z"
output_directory: "/tmp/test-output"
export_format: "html"
remote_user: "testuser"
ssh_private_key_path: "` + tmpKeyFile.Name() + `"
remote_host: "test.example.com"
remote_archive_path: "/backup/imessages"
ssh:
  enabled: false
`

	// Write config to temporary file
	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// Test loading the config
	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify config values
	if cfg.LoggingLevel != "debug" {
		t.Errorf("Expected logging level 'debug', got '%s'", cfg.LoggingLevel)
	}

	if cfg.ExportFormat != "html" {
		t.Errorf("Expected export format 'html', got '%s'", cfg.ExportFormat)
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error when loading nonexistent file, got nil")
	}
}
