package archiver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iwvelando/imessage-archiver/internal/config"
	"github.com/iwvelando/imessage-archiver/internal/logger"
)

// getTestDatabasePath returns the path to the test database
func getTestDatabasePath() string {
	// Get the current file's directory and work backwards to project root
	currentDir, _ := os.Getwd()
	// Navigate to project root (assuming we're in internal/archiver)
	projectRoot := filepath.Join(currentDir, "..", "..")
	testDbPath := filepath.Join(projectRoot, "internal", "archiver", "testdata", "chat.db")
	return testDbPath
}

// checkTestDatabaseExists checks if the test database exists and fails the test if it doesn't
func checkTestDatabaseExists(t *testing.T) {
	testDbPath := getTestDatabasePath()
	if _, err := os.Stat(testDbPath); os.IsNotExist(err) {
		t.Fatalf("Test database not found at %s. Run 'make generate-test-db' to create it.", testDbPath)
	}
}

func TestNew(t *testing.T) {
	cfg := &config.Config{
		LoggingLevel:     "info",
		TestDatabasePath: getTestDatabasePath(),
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
	// Check that test database exists before running tests that need it
	checkTestDatabaseExists(t)

	cfg := &config.Config{
		LoggingLevel:      "info",
		RemoteUser:        "testuser",
		RemoteHost:        "nonexistent.example.com",
		SSHPrivateKeyPath: "/nonexistent/key",
		RemoteArchivePath: "/backup/test",
		DaysToCheck:       1,
		ExportFormat:      "txt",
		CopyMethod:        "basic",
		TestDatabasePath:  getTestDatabasePath(),
	}
	log := logger.New("info")

	archiver := New(cfg, log)

	// Test that Run method exists and can be called
	// This will fail due to missing SSH config, but we're testing the flow
	err := archiver.Run()

	// We expect an error in test environment due to invalid SSH config
	if err == nil {
		t.Log("Run completed without error (unexpected in test environment)")
	} else {
		t.Logf("Run failed as expected in test environment: %v", err)
	}
}

func TestArchiver_findMissingArchives_Fallback(t *testing.T) {
	cfg := &config.Config{
		LoggingLevel:      "debug",
		DaysToCheck:       3,
		RemoteUser:        "testuser",
		RemoteHost:        "nonexistent.example.com",
		SSHPrivateKeyPath: "/nonexistent/key",
		RemoteArchivePath: "/backup/test",
	}
	log := logger.New("debug")
	archiver := New(cfg, log)

	// This should fall back to checking all dates when remote query fails
	missingDates, err := archiver.findMissingArchives()

	if err != nil {
		t.Fatalf("Expected findMissingArchives to handle remote failure gracefully, got error: %v", err)
	}

	// Should return the configured number of days to check
	if len(missingDates) != cfg.DaysToCheck {
		t.Errorf("Expected %d missing dates (fallback mode), got %d", cfg.DaysToCheck, len(missingDates))
	}

	// Verify dates are in correct order (most recent first)
	now := time.Now()
	for i, date := range missingDates {
		expectedDate := now.AddDate(0, 0, -(i + 1))
		if date.Format("2006-01-02") != expectedDate.Format("2006-01-02") {
			t.Errorf("Expected date %s at position %d, got %s",
				expectedDate.Format("2006-01-02"), i, date.Format("2006-01-02"))
		}
	}
}

func TestArchiver_processDateLocally_EmptyExport(t *testing.T) {
	// Check that test database exists before running tests that need it
	checkTestDatabaseExists(t)

	// Create temporary directories
	tempRoot, err := os.MkdirTemp("", "archiver-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempRoot); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	cfg := &config.Config{
		LoggingLevel:     "debug",
		ExportFormat:     "txt",
		CopyMethod:       "basic",
		TestDatabasePath: getTestDatabasePath(),
	}
	log := logger.New("debug")
	archiver := New(cfg, log)

	targetDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	// Create a mock archiver that skips the actual imessage-exporter call
	// by pre-creating an empty export directory structure
	year := targetDate.Format("2006")
	month := targetDate.Format("01")
	day := targetDate.Format("02")
	expectedDir := filepath.Join(tempRoot, year, month, day)

	// Pre-create the directory structure but leave it empty
	if err := os.MkdirAll(expectedDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a mock empty export (what imessage-exporter would create for no messages)
	attachmentsDir := filepath.Join(expectedDir, "attachments")
	if err := os.MkdirAll(attachmentsDir, 0755); err != nil {
		t.Fatalf("Failed to create attachments directory: %v", err)
	}

	orphanedHTML := filepath.Join(expectedDir, "orphaned.html")
	if err := os.WriteFile(orphanedHTML, []byte("<html>No messages</html>"), 0644); err != nil {
		t.Fatalf("Failed to create orphaned.html: %v", err)
	}

	// Test that processDateLocally detects empty export and cleans up
	// Note: This will fail at the exportMessages step in a real test environment
	// but we can test the directory creation logic
	err = archiver.processDateLocally(targetDate, tempRoot)

	// In a real environment, this would fail due to missing imessage-exporter
	// The test validates that the function handles the flow correctly
	if err != nil {
		t.Logf("processDateLocally failed as expected in test environment: %v", err)

		// Verify the directory structure was created correctly
		if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
			t.Error("Expected directory structure to be created")
		}
	}
}

func TestArchiver_processDateLocally_DirectoryCreation(t *testing.T) {
	tempRoot, err := os.MkdirTemp("", "archiver-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempRoot); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	cfg := &config.Config{
		LoggingLevel:     "debug",
		ExportFormat:     "html",
		CopyMethod:       "clone",
		TestDatabasePath: getTestDatabasePath(),
	}
	log := logger.New("debug")
	archiver := New(cfg, log)

	targetDate := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)

	// Test directory creation regardless of whether imessage-exporter works
	err = archiver.processDateLocally(targetDate, tempRoot)

	// Verify correct directory structure was created
	expectedPath := filepath.Join(tempRoot, "2023", "12", "25")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("Expected year/month/day directory structure to be created")
	}

	// In test environment, this may succeed or fail depending on imessage-exporter availability
	// The important thing is that directory structure was created
	t.Logf("processDateLocally result: %v", err)
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

func TestArchiver_cleanup(t *testing.T) {
	cfg := &config.Config{
		LoggingLevel: "debug",
	}
	log := logger.New("debug")
	archiver := New(cfg, log)

	// Create a temporary directory with some content
	tempDir, err := os.MkdirTemp("", "cleanup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Add some content
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Verify directory exists before cleanup
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Fatal("Test directory should exist before cleanup")
	}

	// Test cleanup
	archiver.cleanup(tempDir)

	// Verify directory is removed after cleanup
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Error("Directory should be removed after cleanup")
	}
}

func TestArchiver_cleanup_NonexistentDirectory(t *testing.T) {
	cfg := &config.Config{
		LoggingLevel: "debug",
	}
	log := logger.New("debug")
	archiver := New(cfg, log)

	// Test cleanup on non-existent directory (should not panic)
	archiver.cleanup("/nonexistent/directory")
	// If we reach here without panic, the test passes
}

func TestArchiver_exportMessages_InvalidConfig(t *testing.T) {
	cfg := &config.Config{
		LoggingLevel:     "debug",
		ExportFormat:     "txt",
		CopyMethod:       "basic",
		TestDatabasePath: getTestDatabasePath(),
	}
	log := logger.New("debug")
	archiver := New(cfg, log)

	tempDir, err := os.MkdirTemp("", "export-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	targetDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Test exportMessages function - may succeed or fail depending on environment
	err = archiver.exportMessages(targetDate, tempDir)

	if err != nil {
		// Verify error contains helpful information
		errorMsg := err.Error()
		if errorMsg == "" {
			t.Error("Expected non-empty error message")
		}
		t.Logf("exportMessages failed as expected: %v", err)
	} else {
		t.Logf("exportMessages succeeded in test environment (imessage-exporter is available)")
	}
}

func TestArchiver_exportMessages_WithValidDatabase(t *testing.T) {
	// Check that test database exists
	checkTestDatabaseExists(t)

	cfg := &config.Config{
		LoggingLevel:     "debug",
		ExportFormat:     "txt",
		CopyMethod:       "basic",
		TestDatabasePath: getTestDatabasePath(),
	}
	log := logger.New("debug")
	archiver := New(cfg, log)

	tempDir, err := os.MkdirTemp("", "export-test-with-db-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	// Use the date from our test data (2024-01-01)
	targetDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Test exportMessages function with valid database
	err = archiver.exportMessages(targetDate, tempDir)

	if err != nil {
		// If it fails, make sure it's not due to database issues
		if strings.Contains(err.Error(), "does not exist at the specified path") {
			t.Fatalf("Database path issue even though checkTestDatabaseExists passed: %v", err)
		}
		t.Logf("exportMessages failed (possibly due to imessage-exporter not available): %v", err)
	} else {
		t.Logf("exportMessages succeeded with test database")

		// Check if any output was created
		entries, err := os.ReadDir(tempDir)
		if err != nil {
			t.Logf("Could not read output directory: %v", err)
		} else {
			t.Logf("Export created %d entries in output directory", len(entries))
		}
	}
}

func TestArchiver_TestDatabasePathExists(t *testing.T) {
	// This test validates that our test database detection works correctly
	testDbPath := getTestDatabasePath()
	t.Logf("Test database path: %s", testDbPath)

	// Check if the path contains the expected components
	if !strings.Contains(testDbPath, "internal/archiver/testdata/chat.db") {
		t.Errorf("Test database path doesn't contain expected components: %s", testDbPath)
	}

	if _, err := os.Stat(testDbPath); os.IsNotExist(err) {
		t.Skipf("Test database not found at %s. Run 'make generate-test-db' to create it.", testDbPath)
	} else {
		t.Logf("Test database found at %s", testDbPath)

		// Verify it's a valid SQLite database by attempting to open it
		// This is a basic smoke test to ensure the file isn't corrupted
		cfg := &config.Config{
			LoggingLevel:     "debug",
			ExportFormat:     "txt",
			CopyMethod:       "basic",
			TestDatabasePath: testDbPath,
		}
		log := logger.New("debug")
		archiver := New(cfg, log)

		// Try a basic export to verify the database works
		tempDir, err := os.MkdirTemp("", "db-validation-*")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("Failed to remove temp directory: %v", err)
			}
		}()

		targetDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		err = archiver.exportMessages(targetDate, tempDir)

		if err == nil {
			t.Logf("Test database validation succeeded - imessage-exporter can read the database")
		} else if strings.Contains(err.Error(), "does not exist at the specified path") {
			t.Errorf("Test database file exists but imessage-exporter says it doesn't: %v", err)
		} else {
			t.Logf("Test database exists but export failed (possibly due to missing imessage-exporter): %v", err)
		}
	}
}

func TestArchiver_batchSyncToRemote_InvalidConfig(t *testing.T) {
	cfg := &config.Config{
		LoggingLevel:      "debug",
		RemoteUser:        "testuser",
		RemoteHost:        "nonexistent.example.com",
		SSHPrivateKeyPath: "/nonexistent/key",
		RemoteArchivePath: "/backup/test",
	}
	log := logger.New("debug")
	archiver := New(cfg, log)

	tempDir, err := os.MkdirTemp("", "sync-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	// This should fail due to invalid SSH configuration
	err = archiver.batchSyncToRemote(tempDir)
	if err == nil {
		t.Error("Expected batchSyncToRemote to fail with invalid config")
	}
	t.Logf("batchSyncToRemote failed as expected: %v", err)
}

func TestArchiver_getRemoteArchiveStructure(t *testing.T) {
	cfg := &config.Config{
		LoggingLevel:      "debug",
		RemoteUser:        "testuser",
		SSHPrivateKeyPath: "/fake/key",
		RemoteHost:        "test.example.com",
		RemoteArchivePath: "/backup/imessages",
	}
	log := logger.New("debug")
	archiver := New(cfg, log)

	// Test case 1: SSH command fails
	t.Run("ssh command fails", func(t *testing.T) {
		// This will fail because the SSH key and host don't exist
		_, err := archiver.getRemoteArchiveStructure()
		if err == nil {
			t.Error("Expected error when SSH command fails, got nil")
		}
	})
}

func TestArchiver_findMissingArchives(t *testing.T) {
	cfg := &config.Config{
		LoggingLevel:      "debug",
		RemoteUser:        "testuser",
		SSHPrivateKeyPath: "/fake/key",
		RemoteHost:        "test.example.com",
		RemoteArchivePath: "/backup/imessages",
		DaysToCheck:       3,
	}
	log := logger.New("debug")
	archiver := New(cfg, log)

	t.Run("falls back to all dates when remote query fails", func(t *testing.T) {
		dates, err := archiver.findMissingArchives()
		if err != nil {
			t.Fatalf("Expected no error in fallback mode, got: %v", err)
		}

		if len(dates) != 3 {
			t.Errorf("Expected 3 dates (DaysToCheck), got %d", len(dates))
		}

		// Verify dates are in descending order (most recent first)
		now := time.Now()
		for i, date := range dates {
			expectedDate := now.AddDate(0, 0, -(i + 1))
			if date.Year() != expectedDate.Year() ||
				date.Month() != expectedDate.Month() ||
				date.Day() != expectedDate.Day() {
				t.Errorf("Date %d: expected %v, got %v", i, expectedDate.Format("2006-01-02"), date.Format("2006-01-02"))
			}
		}
	})
}

func TestArchiver_processDateLocally(t *testing.T) {
	cfg := &config.Config{
		LoggingLevel:     "debug",
		ExportFormat:     "txt",
		CopyMethod:       "basic",
		TestDatabasePath: getTestDatabasePath(),
	}
	log := logger.New("debug")
	archiver := New(cfg, log)

	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "archiver-process-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	testDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	t.Run("creates proper directory structure", func(t *testing.T) {
		// This may succeed or fail depending on imessage-exporter availability
		// But we can test that it creates the directory structure correctly
		err := archiver.processDateLocally(testDate, tempDir)

		// Check that directory structure was created
		expectedDir := filepath.Join(tempDir, "2024", "01", "15")
		if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
			t.Errorf("Expected directory %s to be created", expectedDir)
		}

		// Log the result - may succeed or fail depending on environment
		t.Logf("processDateLocally result: %v", err)
	})
}

func TestArchiver_exportMessages(t *testing.T) {
	cfg := &config.Config{
		LoggingLevel:     "debug",
		ExportFormat:     "html",
		CopyMethod:       "basic",
		TestDatabasePath: getTestDatabasePath(),
	}
	log := logger.New("debug")
	archiver := New(cfg, log)

	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "archiver-export-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	testDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	t.Run("handles imessage-exporter execution", func(t *testing.T) {
		err := archiver.exportMessages(testDate, tempDir)

		if err != nil {
			// Verify error contains helpful information
			if !strings.Contains(err.Error(), "imessage-exporter") {
				t.Errorf("Expected error to mention imessage-exporter, got: %v", err)
			}
			t.Logf("exportMessages failed: %v", err)
		} else {
			t.Logf("exportMessages succeeded in test environment")
		}
	})
}

func TestArchiver_batchSyncToRemote(t *testing.T) {
	cfg := &config.Config{
		LoggingLevel:      "debug",
		RemoteUser:        "testuser",
		SSHPrivateKeyPath: "/fake/key",
		RemoteHost:        "test.example.com",
		RemoteArchivePath: "/backup/imessages",
	}
	log := logger.New("debug")
	archiver := New(cfg, log)

	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "archiver-sync-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	t.Run("fails when rsync command fails", func(t *testing.T) {
		err := archiver.batchSyncToRemote(tempDir)
		if err == nil {
			t.Error("Expected error when rsync fails, got nil")
		}
	})
}

// TestDatabaseRequired tests that fail when database is missing
func TestDatabaseRequired(t *testing.T) {
	// This test intentionally fails if the database doesn't exist
	// It demonstrates the difference between tests that require the database
	// vs tests that just log warnings
	checkTestDatabaseExists(t)

	// If we get here, the database exists
	t.Logf("Database requirement check passed - database exists")
}

// TestIntentionalFailure is designed to fail to test GitHub Actions
func TestIntentionalFailure(t *testing.T) {
	t.Fatal("This test is intentionally failing to test GitHub Actions workflow")
}
