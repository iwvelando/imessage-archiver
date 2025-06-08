package archiver

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/iwvelando/imessage-archiver/internal/config"
	"github.com/iwvelando/imessage-archiver/internal/logger"
)

type Archiver struct {
	config *config.Config
	logger *logger.Logger
}

func New(cfg *config.Config, log *logger.Logger) *Archiver {
	return &Archiver{
		config: cfg,
		logger: log,
	}
}

func (a *Archiver) Run() error {
	a.logger.Info("Starting iMessage archival process")

	// Find the date range to process
	datesToProcess, err := a.findMissingArchives()
	if err != nil {
		return fmt.Errorf("failed to find missing archives: %w", err)
	}

	if len(datesToProcess) == 0 {
		a.logger.Info("No missing archives found within the specified range")
		return nil
	}

	// Format dates for logging
	dateStrings := make([]string, len(datesToProcess))
	for i, date := range datesToProcess {
		dateStrings[i] = date.Format("2006-01-02")
	}

	a.logger.Info(fmt.Sprintf("Found %d dates to archive: %v", len(datesToProcess), dateStrings))

	// Create a temporary local root directory for all exports
	localRootDir := filepath.Join(os.TempDir(), "imessage-batch-export")
	if err := os.MkdirAll(localRootDir, 0755); err != nil {
		return fmt.Errorf("failed to create local root directory: %w", err)
	}

	// Set up signal handling for cleanup on interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Cleanup function that will be called on defer or signal
	cleanupFunc := func() {
		a.cleanup(localRootDir)
	}
	defer cleanupFunc()

	// Handle signals in a goroutine
	go func() {
		sig := <-sigChan
		a.logger.Info(fmt.Sprintf("Received signal %v, cleaning up and exiting...", sig))
		cleanupFunc()
		os.Exit(1)
	}()

	// Process each date and build local directory structure
	hasDataToSync := false
	for _, targetDate := range datesToProcess {
		if err := a.processDateLocally(targetDate, localRootDir); err != nil {
			a.logger.Error(fmt.Sprintf("Failed to process date %s: %v", targetDate.Format("2006-01-02"), err))
			return fmt.Errorf("failed to process date %s: %w", targetDate.Format("2006-01-02"), err)
		}
		hasDataToSync = true
	}

	// Perform single batch sync if we have data to sync
	if hasDataToSync {
		if err := a.batchSyncToRemote(localRootDir); err != nil {
			a.logger.Error(fmt.Sprintf("Failed to sync batch to remote server: %v", err))
			return fmt.Errorf("batch sync failed: %w", err)
		}
	}

	a.logger.Info("iMessage Archiver completed successfully")
	return nil
}

func (a *Archiver) findMissingArchives() ([]time.Time, error) {
	a.logger.Debug("Finding missing archives to process")

	var missingDates []time.Time
	today := time.Now()

	// Get the remote directory structure in one query
	remoteArchives, err := a.getRemoteArchiveStructure()
	if err != nil {
		a.logger.Warn(fmt.Sprintf("Failed to get remote archive structure: %v", err))
		// Fallback to checking all dates if remote query fails
		for i := 1; i <= a.config.DaysToCheck; i++ {
			checkDate := today.AddDate(0, 0, -i)
			missingDates = append(missingDates, checkDate)
		}
		return missingDates, nil
	}

	// Check each day going back up to days_to_check
	for i := 1; i <= a.config.DaysToCheck; i++ {
		checkDate := today.AddDate(0, 0, -i)
		dateStr := checkDate.Format("2006-01-02")

		if !remoteArchives[dateStr] {
			a.logger.Debug(fmt.Sprintf("Missing archive for date: %s", dateStr))
			missingDates = append(missingDates, checkDate)
		} else {
			a.logger.Debug(fmt.Sprintf("Archive exists for date: %s", dateStr))
		}
	}

	return missingDates, nil
}

// getRemoteArchiveStructure retrieves the entire remote directory structure
// in a single SSH command and returns a map of existing archive dates
func (a *Archiver) getRemoteArchiveStructure() (map[string]bool, error) {
	a.logger.Debug("Retrieving remote archive structure")

	// Use find command to get all directories in the archive path that match the date pattern
	// This finds directories 3 levels deep (year/month/day) and extracts the full date path
	cmd := exec.Command("ssh",
		"-i", a.config.SSHPrivateKeyPath,
		"-o", "ConnectTimeout=30",
		"-o", "ServerAliveInterval=60",
		"-o", "ServerAliveCountMax=3",
		fmt.Sprintf("%s@%s", a.config.RemoteUser, a.config.RemoteHost),
		fmt.Sprintf("find %s -type d -mindepth 3 -maxdepth 3 -path '*/[0-9][0-9][0-9][0-9]/[0-9][0-9]/[0-9][0-9]' 2>/dev/null | while read dir; do if [ -n \"$(ls -A \"$dir\" 2>/dev/null)\" ]; then echo \"$dir\"; fi; done", a.config.RemoteArchivePath),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		a.logger.Debug(fmt.Sprintf("Remote structure query output: %s", string(output)))
		return nil, fmt.Errorf("failed to query remote archive structure: %w", err)
	}

	// Parse the output to build a map of existing archives
	archives := make(map[string]bool)
	outputStr := strings.TrimSpace(string(output))

	if outputStr != "" {
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Extract date from path like /backups/imessages/2024/06/07
			// Remove the base path and convert to YYYY-MM-DD format
			relativePath := strings.TrimPrefix(line, a.config.RemoteArchivePath)
			relativePath = strings.TrimPrefix(relativePath, "/")

			// Split into year/month/day components
			parts := strings.Split(relativePath, "/")
			if len(parts) == 3 {
				dateStr := fmt.Sprintf("%s-%s-%s", parts[0], parts[1], parts[2])
				archives[dateStr] = true
				a.logger.Debug(fmt.Sprintf("Found existing archive: %s", dateStr))
			}
		}
	}

	a.logger.Debug(fmt.Sprintf("Retrieved %d existing archives from remote", len(archives)))
	return archives, nil
}

func (a *Archiver) checkRemoteArchiveExists(date time.Time) (bool, error) {
	remoteDir := a.getRemoteArchivePath(date)

	a.logger.Debug(fmt.Sprintf("Checking if remote archive exists: %s", remoteDir))

	cmd := exec.Command("ssh",
		"-i", a.config.SSHPrivateKeyPath,
		"-o", "ConnectTimeout=30",
		"-o", "ServerAliveInterval=60",
		"-o", "ServerAliveCountMax=3",
		fmt.Sprintf("%s@%s", a.config.RemoteUser, a.config.RemoteHost),
		fmt.Sprintf("test -d %s && ls -la %s | grep -v '^total'", remoteDir, remoteDir),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Directory doesn't exist or is empty
		a.logger.Debug(fmt.Sprintf("Remote directory check output: %s", string(output)))
		return false, nil
	}

	// Check if directory has content (not just empty)
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return false, nil
	}

	a.logger.Debug(fmt.Sprintf("Remote directory exists with content: %s", remoteDir))
	return true, nil
}

func (a *Archiver) processDateLocally(targetDate time.Time, localRootDir string) error {
	dateStr := targetDate.Format("2006-01-02")
	a.logger.Info(fmt.Sprintf("Archiving messages for date: %s", dateStr))

	// Create local export directory with proper hierarchy (year/month/day)
	year := targetDate.Format("2006")
	month := targetDate.Format("01")
	day := targetDate.Format("02")
	localExportDir := filepath.Join(localRootDir, year, month, day)

	if err := os.MkdirAll(localExportDir, 0755); err != nil {
		return fmt.Errorf("failed to create local export directory: %w", err)
	}

	// Export messages for the target date
	if err := a.exportMessages(targetDate, localExportDir); err != nil {
		a.logger.Error(fmt.Sprintf("Failed to export messages: %v", err))
		return fmt.Errorf("message export failed: %w", err)
	}

	// Check if there are any messages to archive
	isEmpty, err := a.isDirectoryEmpty(localExportDir)
	if err != nil {
		return fmt.Errorf("failed to check export directory: %w", err)
	}

	if isEmpty {
		a.logger.Info(fmt.Sprintf("No messages found for date %s, skipping archive", dateStr))
		// Cleanup empty directory
		if err := os.RemoveAll(localExportDir); err != nil {
			a.logger.Warn(fmt.Sprintf("Failed to remove empty export directory %s: %v", localExportDir, err))
		}
		return nil
	}

	a.logger.Info(fmt.Sprintf("Successfully processed messages for %s locally", dateStr))
	return nil
}

func (a *Archiver) exportMessages(targetDate time.Time, exportDir string) error {
	startDateStr := targetDate.Format("2006-01-02")
	// End date is the next day to capture only the target date's messages
	endDateStr := targetDate.AddDate(0, 0, 1).Format("2006-01-02")

	a.logger.Debug(fmt.Sprintf("Exporting messages from %s to %s (exclusive) to %s", startDateStr, endDateStr, exportDir))

	// Use config values for format and copy method
	format := a.config.ExportFormat
	copyMethod := a.config.CopyMethod

	cmd := exec.Command("imessage-exporter",
		"--format", format,
		"--copy-method", copyMethod,
		"--export-path", exportDir,
		"--start-date", startDateStr,
		"--end-date", endDateStr,
		"--no-lazy",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		a.logger.Debug(fmt.Sprintf("imessage-exporter output: %s", string(output)))
		return fmt.Errorf("imessage-exporter failed: %w", err)
	}

	a.logger.Debug("Message export completed successfully")
	return nil
}

func (a *Archiver) isDirectoryEmpty(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}

	// If completely empty, it's empty
	if len(entries) == 0 {
		return true, nil
	}

	// Check if directory only contains "empty export" artifacts from imessage-exporter
	// When there are no messages, imessage-exporter typically creates:
	// - An empty "attachments" directory
	// - An "orphaned.html" file (containing no actual messages)
	hasOnlyEmptyArtifacts := true
	hasAttachments := false
	hasOrphaned := false

	for _, entry := range entries {
		switch entry.Name() {
		case "attachments":
			if entry.IsDir() {
				// Check if attachments directory is empty
				attachmentsPath := filepath.Join(dir, "attachments")
				attachmentEntries, err := os.ReadDir(attachmentsPath)
				if err != nil {
					a.logger.Debug(fmt.Sprintf("Error reading attachments directory: %v", err))
					hasOnlyEmptyArtifacts = false
					break
				}
				if len(attachmentEntries) > 0 {
					// Attachments directory has content, so this is not empty
					hasOnlyEmptyArtifacts = false
					break
				}
				hasAttachments = true
			} else {
				// attachments is not a directory, unexpected
				hasOnlyEmptyArtifacts = false
				break
			}
		case "orphaned.html":
			if !entry.IsDir() {
				// Check if orphaned.html is small (indicating no real content)
				orphanedPath := filepath.Join(dir, "orphaned.html")
				info, err := os.Stat(orphanedPath)
				if err != nil {
					a.logger.Debug(fmt.Sprintf("Error stating orphaned.html: %v", err))
					hasOnlyEmptyArtifacts = false
					break
				}
				// If orphaned.html is larger than 10KB, assume it has real content
				if info.Size() > 10240 {
					hasOnlyEmptyArtifacts = false
					break
				}
				hasOrphaned = true
			} else {
				// orphaned.html is a directory, unexpected
				hasOnlyEmptyArtifacts = false
				break
			}
		default:
			// Found other files/directories, so this is not just empty artifacts
			hasOnlyEmptyArtifacts = false
			break
		}
	}

	// Consider it empty if we only have the typical empty export artifacts
	isEmpty := hasOnlyEmptyArtifacts && (hasAttachments || hasOrphaned)

	if isEmpty {
		a.logger.Debug(fmt.Sprintf("Directory contains only empty export artifacts (attachments: %v, orphaned: %v)", hasAttachments, hasOrphaned))
	}

	return isEmpty, nil
}

func (a *Archiver) getRemoteArchivePath(date time.Time) string {
	year := date.Format("2006")
	month := date.Format("01")
	day := date.Format("02")

	return filepath.Join(a.config.RemoteArchivePath, year, month, day)
}

func (a *Archiver) createRemoteDirectory(remoteDir string) error {
	a.logger.Debug(fmt.Sprintf("Creating remote directory: %s", remoteDir))

	cmd := exec.Command("ssh",
		"-i", a.config.SSHPrivateKeyPath,
		"-o", "ConnectTimeout=30",
		"-o", "ServerAliveInterval=60",
		"-o", "ServerAliveCountMax=3",
		fmt.Sprintf("%s@%s", a.config.RemoteUser, a.config.RemoteHost),
		fmt.Sprintf("mkdir -p %s", remoteDir),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		a.logger.Debug(fmt.Sprintf("SSH mkdir output: %s", string(output)))
		return fmt.Errorf("failed to create remote directory via SSH: %w", err)
	}

	return nil
}

func (a *Archiver) batchSyncToRemote(localRootDir string) error {
	a.logger.Debug("Starting batch sync to remote server")

	cmd := exec.Command("rsync",
		"-avz",
		"--timeout=300",
		"-e", fmt.Sprintf("ssh -i %s -o ConnectTimeout=30 -o ServerAliveInterval=60 -o ServerAliveCountMax=3", a.config.SSHPrivateKeyPath),
		localRootDir+"/",
		fmt.Sprintf("%s@%s:%s/", a.config.RemoteUser, a.config.RemoteHost, a.config.RemoteArchivePath),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		a.logger.Debug(fmt.Sprintf("Rsync output: %s", string(output)))
		return fmt.Errorf("batch rsync failed: %w", err)
	}

	a.logger.Debug("Batch sync completed successfully")
	return nil
}

func (a *Archiver) cleanup(dir string) {
	if err := os.RemoveAll(dir); err != nil {
		a.logger.Warn(fmt.Sprintf("Failed to cleanup temporary directory %s: %v", dir, err))
	} else {
		a.logger.Debug(fmt.Sprintf("Cleaned up temporary directory: %s", dir))
	}
}
