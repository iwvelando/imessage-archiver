package config

import (
	"os"
	"testing"
)

func TestConfigStruct(t *testing.T) {
	// Test that we can create a Config struct
	cfg := &Config{
		RemoteUser:        "testuser",
		RemoteHost:        "testhost",
		SSHPrivateKeyPath: "/fake/path",
		RemoteArchivePath: "/fake/archive",
		DaysToCheck:       7,
		ExportFormat:      "html",
		CopyMethod:        "basic",
		LoggingLevel:      "info",
	}

	if cfg.RemoteUser != "testuser" {
		t.Errorf("Expected RemoteUser to be 'testuser', got: %s", cfg.RemoteUser)
	}
}

func TestTempFileHandling(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-file-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	testContent := "test content"
	if _, err := tmpFile.WriteString(testContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Verify file exists and contains expected content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("Expected file content to be '%s', got: %s", testContent, string(content))
	}
}
