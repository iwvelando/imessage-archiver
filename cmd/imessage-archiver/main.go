package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/iwvelando/imessage-archiver/internal/archiver"
	"github.com/iwvelando/imessage-archiver/internal/config"
	"github.com/iwvelando/imessage-archiver/internal/logger"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to configuration file")
	flag.Parse()

	// Default config path if not provided
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}
		configPath = filepath.Join(homeDir, ".config", "imessage-archiver", "config.yaml")
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger with configured level
	log := logger.New(cfg.LoggingLevel)

	log.Info("iMessage Archiver starting up")
	log.Debug(fmt.Sprintf("Configuration loaded from: %s", configPath))

	// Create an instance of the Archiver
	arch := archiver.New(cfg, log)

	// Run the archiving process with fault tolerance
	if err := arch.Run(); err != nil {
		log.Error(fmt.Sprintf("Archiving process failed: %v", err))
		os.Exit(1)
	}
}
