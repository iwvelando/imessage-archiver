package config

import (
	"os"
	"path/filepath"
	"strings"
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

func TestLoad_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		expectedError string
	}{
		{
			name: "missing remote_user",
			configContent: `
logging_level: "info"
ssh_private_key_path: "/tmp/test-key"
remote_host: "example.com"
remote_archive_path: "/backup"
`,
			expectedError: "remote_user is required in config",
		},
		{
			name: "missing ssh_private_key_path",
			configContent: `
logging_level: "info"
remote_user: "testuser"
remote_host: "example.com"
remote_archive_path: "/backup"
`,
			expectedError: "ssh_private_key_path is required in config",
		},
		{
			name: "missing remote_host",
			configContent: `
logging_level: "info"
remote_user: "testuser"
ssh_private_key_path: "/tmp/test-key"
remote_archive_path: "/backup"
`,
			expectedError: "remote_host is required in config",
		},
		{
			name: "missing remote_archive_path",
			configContent: `
logging_level: "info"
remote_user: "testuser"
ssh_private_key_path: "/tmp/test-key"
remote_host: "example.com"
`,
			expectedError: "remote_archive_path is required in config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.configContent); err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}
			tmpFile.Close()

			_, err = Load(tmpFile.Name())
			if err == nil {
				t.Errorf("Expected error for missing field, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}

func TestLoad_InvalidFieldValues(t *testing.T) {
	// Create a temporary SSH key file
	tmpKeyFile, err := os.CreateTemp("", "test-ssh-key-*")
	if err != nil {
		t.Fatalf("Failed to create temp SSH key file: %v", err)
	}
	defer os.Remove(tmpKeyFile.Name())
	tmpKeyFile.WriteString("test-ssh-key-content")
	tmpKeyFile.Close()

	tests := []struct {
		name          string
		configContent string
		expectedError string
	}{
		{
			name: "invalid logging_level",
			configContent: `
logging_level: "invalid"
remote_user: "testuser"
ssh_private_key_path: "` + tmpKeyFile.Name() + `"
remote_host: "example.com"
remote_archive_path: "/backup"
`,
			expectedError: "invalid logging_level: invalid",
		},
		{
			name: "invalid export_format",
			configContent: `
logging_level: "info"
export_format: "invalid"
remote_user: "testuser"
ssh_private_key_path: "` + tmpKeyFile.Name() + `"
remote_host: "example.com"
remote_archive_path: "/backup"
`,
			expectedError: "invalid export_format: invalid",
		},
		{
			name: "invalid copy_method",
			configContent: `
logging_level: "info"
copy_method: "invalid"
remote_user: "testuser"
ssh_private_key_path: "` + tmpKeyFile.Name() + `"
remote_host: "example.com"
remote_archive_path: "/backup"
`,
			expectedError: "invalid copy_method: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.configContent); err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}
			tmpFile.Close()

			_, err = Load(tmpFile.Name())
			if err == nil {
				t.Errorf("Expected error for invalid field value, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}

func TestLoad_NonexistentSSHKey(t *testing.T) {
	configContent := `
logging_level: "info"
remote_user: "testuser"
ssh_private_key_path: "/nonexistent/ssh/key"
remote_host: "example.com"
remote_archive_path: "/backup"
`

	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	_, err = Load(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for nonexistent SSH key file, got nil")
		return
	}

	if !strings.Contains(err.Error(), "ssh private key file does not exist") {
		t.Errorf("Expected error about SSH key file, got: %s", err.Error())
	}
}

func TestLoad_TildeExpansion(t *testing.T) {
	// Create SSH key in temp directory to simulate home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Skipping test due to inability to get home directory: %v", err)
	}

	// Create a temporary SSH key file in home directory
	sshKeyPath := filepath.Join(homeDir, ".ssh", "test-key-"+filepath.Base(os.TempDir()))
	if err := os.MkdirAll(filepath.Dir(sshKeyPath), 0700); err != nil {
		t.Fatalf("Failed to create SSH directory: %v", err)
	}
	defer os.Remove(sshKeyPath)

	if err := os.WriteFile(sshKeyPath, []byte("test-ssh-key"), 0600); err != nil {
		t.Fatalf("Failed to create SSH key file: %v", err)
	}

	// Use tilde notation in config
	relativePath := "~/.ssh/" + filepath.Base(sshKeyPath)
	configContent := `
logging_level: "info"
remote_user: "testuser"
ssh_private_key_path: "` + relativePath + `"
remote_host: "example.com"
remote_archive_path: "/backup"
`

	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// This should successfully load and expand the tilde
	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config with tilde expansion: %v", err)
	}

	if cfg.SSHPrivateKeyPath != relativePath {
		t.Errorf("Expected SSH key path to remain as configured: %s, got: %s",
			relativePath, cfg.SSHPrivateKeyPath)
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Create a temporary SSH key file
	tmpKeyFile, err := os.CreateTemp("", "test-ssh-key-*")
	if err != nil {
		t.Fatalf("Failed to create temp SSH key file: %v", err)
	}
	defer os.Remove(tmpKeyFile.Name())
	tmpKeyFile.WriteString("test-ssh-key-content")
	tmpKeyFile.Close()

	// Config with only required fields
	configContent := `
remote_user: "testuser"
ssh_private_key_path: "` + tmpKeyFile.Name() + `"
remote_host: "example.com"
remote_archive_path: "/backup"
`

	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify defaults are set
	if cfg.LoggingLevel != "info" {
		t.Errorf("Expected default logging level 'info', got '%s'", cfg.LoggingLevel)
	}

	if cfg.ExportFormat != "txt" {
		t.Errorf("Expected default export format 'txt', got '%s'", cfg.ExportFormat)
	}

	if cfg.CopyMethod != "basic" {
		t.Errorf("Expected default copy method 'basic', got '%s'", cfg.CopyMethod)
	}

	if cfg.DaysToCheck != 7 {
		t.Errorf("Expected default days to check '7', got %d", cfg.DaysToCheck)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	configContent := `
invalid_yaml: [
  missing_closing_bracket
`

	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	_, err = Load(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
		return
	}

	if !strings.Contains(err.Error(), "failed to parse config file") {
		t.Errorf("Expected parse error, got: %s", err.Error())
	}
}

func TestContains(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}

	tests := []struct {
		item     string
		expected bool
	}{
		{"apple", true},
		{"banana", true},
		{"cherry", true},
		{"grape", false},
		{"", false},
		{"APPLE", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.item, func(t *testing.T) {
			result := contains(slice, tt.item)
			if result != tt.expected {
				t.Errorf("contains(%v, %q) = %v, expected %v", slice, tt.item, result, tt.expected)
			}
		})
	}
}

func TestValidate_EmptySlice(t *testing.T) {
	emptySlice := []string{}
	result := contains(emptySlice, "anything")
	if result != false {
		t.Errorf("contains(empty_slice, 'anything') = %v, expected false", result)
	}
}
