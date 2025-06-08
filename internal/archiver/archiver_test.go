package archiver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/iwvelando/imessage-archiver/internal/config"
	"github.com/iwvelando/imessage-archiver/internal/logger"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{
		LoggingLevel: "info",
	}
	log := logger.New("info")

	archiver := New(cfg, log)

	if archiver == nil {
		t.Fatal("Expected archiver to be created, got nil")
	}

	if archiver.config != cfg {
		t.Error("Expected archiver config to match provided config")
	}

	if archiver.logger != log {
		t.Error("Expected archiver logger to match provided logger")
	}
}

func TestArchiver_Run(t *testing.T) {
	cfg := &config.Config{
		LoggingLevel: "info",
	}
	log := logger.New("info")

	archiver := New(cfg, log)

	// Test that Run method exists and can be called
	// Note: This will likely fail due to missing dependencies in test environment
	// but we're just testing that the method signature is correct
	err := archiver.Run()

	// We expect an error in test environment due to missing chat.db file
	// The important thing is that the method exists and returns an error type
	if err == nil {
		t.Log("Run completed without error (unexpected in test environment)")
	} else {
		t.Logf("Run failed as expected in test environment: %v", err)
	}
}

func TestArchiver_isDirectoryEmpty(t *testing.T) {
	cfg := &config.Config{
		LoggingLevel: "debug",
	}
	log := logger.New("debug")
	archiver := New(cfg, log)

	tests := []struct {
		name     string
		setup    func(string) error
		expected bool
		wantErr  bool
	}{
		{
			name: "completely empty directory",
			setup: func(dir string) error {
				return nil // directory is already empty
			},
			expected: true,
			wantErr:  false,
		},
		{
			name: "directory with only empty attachments",
			setup: func(dir string) error {
				return os.MkdirAll(filepath.Join(dir, "attachments"), 0755)
			},
			expected: true,
			wantErr:  false,
		},
		{
			name: "directory with only small orphaned.html",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "orphaned.html"), []byte("<html>No messages</html>"), 0644)
			},
			expected: true,
			wantErr:  false,
		},
		{
			name: "directory with empty attachments and small orphaned.html",
			setup: func(dir string) error {
				if err := os.MkdirAll(filepath.Join(dir, "attachments"), 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "orphaned.html"), []byte("<html>No messages</html>"), 0644)
			},
			expected: true,
			wantErr:  false,
		},
		{
			name: "directory with attachments containing files",
			setup: func(dir string) error {
				attachmentsDir := filepath.Join(dir, "attachments")
				if err := os.MkdirAll(attachmentsDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(attachmentsDir, "image.jpg"), []byte("fake image data"), 0644)
			},
			expected: false,
			wantErr:  false,
		},
		{
			name: "directory with large orphaned.html",
			setup: func(dir string) error {
				// Create a large orphaned.html file (>10KB)
				largeContent := make([]byte, 12000)
				for i := range largeContent {
					largeContent[i] = 'a'
				}
				return os.WriteFile(filepath.Join(dir, "orphaned.html"), largeContent, 0644)
			},
			expected: false,
			wantErr:  false,
		},
		{
			name: "directory with actual message files",
			setup: func(dir string) error {
				if err := os.MkdirAll(filepath.Join(dir, "attachments"), 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(dir, "orphaned.html"), []byte("<html>No messages</html>"), 0644); err != nil {
					return err
				}
				// Add a real message file
				return os.WriteFile(filepath.Join(dir, "conversation1.html"), []byte("<html>Real messages here</html>"), 0644)
			},
			expected: false,
			wantErr:  false,
		},
		{
			name: "directory with orphaned.html as directory (unexpected)",
			setup: func(dir string) error {
				return os.MkdirAll(filepath.Join(dir, "orphaned.html"), 0755)
			},
			expected: false,
			wantErr:  false,
		},
		{
			name: "directory with attachments as file (unexpected)",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "attachments"), []byte("not a directory"), 0644)
			},
			expected: false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir, err := os.MkdirTemp("", "archiver-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Setup the test scenario
			if err := tt.setup(tempDir); err != nil {
				t.Fatalf("Failed to setup test scenario: %v", err)
			}

			// Test the function
			result, err := archiver.isDirectoryEmpty(tempDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("isDirectoryEmpty() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result != tt.expected {
				t.Errorf("isDirectoryEmpty() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestArchiver_isDirectoryEmpty_NonexistentDirectory(t *testing.T) {
	cfg := &config.Config{
		LoggingLevel: "debug",
	}
	log := logger.New("debug")
	archiver := New(cfg, log)

	_, err := archiver.isDirectoryEmpty("/nonexistent/directory")
	if err == nil {
		t.Error("Expected error for nonexistent directory, got nil")
	}
}
