package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

type Config struct {
	RemoteUser        string `yaml:"remote_user"`
	SSHPrivateKeyPath string `yaml:"ssh_private_key_path"`
	RemoteHost        string `yaml:"remote_host"`
	LoggingLevel      string `yaml:"logging_level"`
	RemoteArchivePath string `yaml:"remote_archive_path"`
	LocalExportPath   string `yaml:"local_export_path,omitempty"`
	ExportFormat      string `yaml:"export_format,omitempty"`
	CopyMethod        string `yaml:"copy_method,omitempty"`
	DaysToCheck       int    `yaml:"days_to_check,omitempty"`
}

func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if config.LoggingLevel == "" {
		config.LoggingLevel = "info"
	}
	if config.LocalExportPath == "" {
		homeDir, _ := os.UserHomeDir()
		config.LocalExportPath = filepath.Join(homeDir, "Library", "Application Support", "iMessage Archive")
	}
	if config.ExportFormat == "" {
		config.ExportFormat = "txt"
	}
	if config.CopyMethod == "" {
		config.CopyMethod = "basic"
	}
	if config.DaysToCheck == 0 {
		config.DaysToCheck = 7
	}

	// Validate required fields
	if config.RemoteUser == "" {
		return nil, fmt.Errorf("remote_user is required in config")
	}
	if config.SSHPrivateKeyPath == "" {
		return nil, fmt.Errorf("ssh_private_key_path is required in config")
	}
	if config.RemoteHost == "" {
		return nil, fmt.Errorf("remote_host is required in config")
	}
	if config.RemoteArchivePath == "" {
		return nil, fmt.Errorf("remote_archive_path is required in config")
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func (c *Config) validate() error {
	if c.RemoteUser == "" {
		return fmt.Errorf("remote_user is required")
	}
	if c.SSHPrivateKeyPath == "" {
		return fmt.Errorf("ssh_private_key_path is required")
	}

	// Expand tilde in SSH key path
	sshKeyPath := c.SSHPrivateKeyPath
	if strings.HasPrefix(sshKeyPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		sshKeyPath = filepath.Join(homeDir, sshKeyPath[2:])
	}

	if _, err := os.Stat(sshKeyPath); os.IsNotExist(err) {
		return fmt.Errorf("ssh private key file does not exist: %s", c.SSHPrivateKeyPath)
	}
	if c.RemoteHost == "" {
		return fmt.Errorf("remote_host is required")
	}
	if c.RemoteArchivePath == "" {
		return fmt.Errorf("remote_archive_path is required")
	}

	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLogLevels, c.LoggingLevel) {
		return fmt.Errorf("invalid logging_level: %s (must be one of: %s)", c.LoggingLevel, strings.Join(validLogLevels, ", "))
	}

	validFormats := []string{"txt", "html"}
	if !contains(validFormats, c.ExportFormat) {
		return fmt.Errorf("invalid export_format: %s (must be one of: %s)", c.ExportFormat, strings.Join(validFormats, ", "))
	}

	validCopyMethods := []string{"clone", "basic", "full", "disabled"}
	if !contains(validCopyMethods, c.CopyMethod) {
		return fmt.Errorf("invalid copy_method: %s (must be one of: %s)", c.CopyMethod, strings.Join(validCopyMethods, ", "))
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
