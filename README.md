# iMessage Archiver

iMessage Archiver is a Go application designed to automate the archiving of iMessages from a macOS device to a remote server. It utilizes the `imessage-exporter` CLI utility to export messages and manages batch synchronization to a remote archive server for daily backups. Note: created heavily with AI assistance as an experimental project.

## Features

- **Automated Daily Archiving**: Configurable scheduling to run daily and archive messages from missed days
- **Intelligent Gap Detection**: Scans remote server to identify missing archive dates and processes only what's needed
- **Batch Synchronization**: Efficiently syncs multiple days of archives in a single operation to reduce network overhead
- **Organized Directory Structure**: Creates year/month/day hierarchy for easy retrieval and organization
- **macOS Integration**: Includes launchd plist and installation scripts for seamless automation
- **Smart Empty Detection**: Identifies and skips days with no actual message content

## Program Flow & Logic

### High-Level Architecture

1. **Configuration Loading**: Loads YAML configuration with remote server details, export preferences, and scheduling options
2. **Gap Analysis**: Queries remote server to identify missing archive dates within the configured lookback window
3. **Local Processing**: For each missing date:
   - Creates temporary local directory structure (year/month/day)
   - Exports messages using `imessage-exporter` with date filtering
   - Validates that exported content contains actual messages (not just empty artifacts)
4. **Batch Synchronization**: Uses `rsync` to efficiently transfer all processed dates to remote server in a single operation
5. **Cleanup**: Removes temporary local files and provides detailed logging

## Runtime Environment

### Requirements

- **macOS**: Required for access to iMessage database and `imessage-exporter`
- **Go 1.24+**: For building the application
- **imessage-exporter**: Must be installed and available in PATH
- **SSH Access**: Configured SSH key-based authentication to remote backup server
- **Network Connectivity**: Reliable connection to remote server for rsync operations

### Dependencies

- `imessage-exporter`: CLI tool for exporting iMessage data
- `rsync`: For efficient file synchronization (included with macOS)
- `ssh`: For remote server communication (included with macOS)
- `gopkg.in/yaml.v2`: Go YAML parsing library

## Installation

### 1. Prerequisites

First, install `imessage-exporter`:
```bash
# Install via Homebrew (recommended)
brew install imessage-exporter

# Or download from: https://github.com/ReagentX/imessage-exporter
```

### 2. Build the Application

```bash
# Clone the repository
git clone https://github.com/iwvelando/imessage-archiver.git
cd imessage-archiver

# Build the binary
make build

# Or build manually
go build -o bin/imessage-archiver ./cmd/imessage-archival
```

### 3. Configuration

Create your configuration file:
```bash
# Create config directory
mkdir -p ~/.config/imessage-archiver

# Copy and edit the example configuration
cp config.yaml ~/.config/imessage-archiver/config.yaml
```

Edit `~/.config/imessage-archiver/config.yaml` with your settings.

### 4. SSH Key Setup

Configure passwordless SSH access to your backup server:
```bash
# Generate SSH key if you don't have one
ssh-keygen -t ed25519 -f ~/.ssh/backup_server_key

# Copy public key to remote server
ssh-copy-id -i ~/.ssh/backup_server_key.pub your_backup_user@backup.example.com

# Test connection
ssh -i ~/.ssh/backup_server_key your_backup_user@backup.example.com
```

### 5. macOS Automation Setup

The project includes automated installation helpers for daily scheduling:

```bash
# Install binary to standard location
make install-local

# Run the macOS automation installer
chmod +x install_macos_automation.sh
./install_macos_automation.sh
```

The installer will:
- Customize the launchd plist template with your home directory
- Install the plist to `~/Library/LaunchAgents/`
- Load the agent to run daily at 4 PM

#### Manual Installation

If you prefer manual setup:
```bash
# Copy and customize the plist file
cp com.imessagearchiver.plist ~/Library/LaunchAgents/
# Edit the file to replace __HOME_DIR__ with your actual home directory

# Load the launch agent
launchctl load ~/Library/LaunchAgents/com.imessagearchiver.plist
```

### 6. Testing

Test the installation:
```bash
# Run manually to verify configuration
~/bin/imessage-archiver -config ~/.config/imessage-archiver/config.yaml

# Check launch agent status
launchctl list | grep com.imessagearchiver

# View logs
tail -f ~/Library/Logs/com.imessagearchiver.out.log
tail -f ~/Library/Logs/com.imessagearchiver.err.log
```

## Usage

### Manual Execution
```bash
# Run with default config location
imessage-archiver

# Run with custom config
imessage-archiver -config /path/to/config.yaml
```

### Scheduled Execution
Once installed with the macOS automation, the archiver will:
- Run daily at 4 PM (configurable in the plist file)
- Automatically catch up on missed days if the system was asleep
- Log all operations to `~/Library/Logs/com.imessagearchiver.*.log`

### Uninstallation
```bash
# Unload the launch agent
launchctl unload ~/Library/LaunchAgents/com.imessagearchiver.plist

# Remove files
rm ~/Library/LaunchAgents/com.imessagearchiver.plist
rm ~/bin/imessage-archiver
rm -r ~/.config/imessage-archiver
rm ~/Library/Logs/com.imessagearchiver.*.log
```

## Configuration Reference

| Setting | Description | Default | Required |
|---------|-------------|---------|----------|
| `remote_user` | SSH username for backup server | - | Yes |
| `ssh_private_key_path` | Path to SSH private key | - | Yes |
| `remote_host` | Backup server hostname/IP | - | Yes |
| `remote_archive_path` | Remote directory for archives | - | Yes |
| `logging_level` | Log verbosity level | "info" | No |
| `export_format` | Export format (txt/html) | "txt" | No |
| `copy_method` | File copy method | "basic" | No |
| `days_to_check` | Lookback window for missed archives | 7 | No |

## Troubleshooting

### Common Issues

1. **Full Disk Access Required** (Most Common Issue)

   The iMessage Archiver needs access to your Messages database to export conversations. If you see this error:
   ```
   Invalid configuration: Unable to read from chat database: unable to open database file: /Users/$USER/Library/Messages/chat.db
   Ensure full disk access is enabled for your terminal emulator in System Settings > Privacy & Security > Full Disk Access
   ```

   **Solution:**
   - Open **System Settings** (or **System Preferences** on older macOS)
   - Navigate to **Privacy & Security** → **Full Disk Access**
   - Click the **+** button to add a new application
   - Navigate to `/Users/isaac/go/src/github.com/iwvelando/imessage-archiver/bin/imessage-archiver`
   - Select the `imessage-archiver` binary and click **Open**
   - Ensure the toggle next to `imessage-archiver` is **enabled**

   **Important Notes:**
   - You must grant access to the compiled binary, not just your terminal
   - Each time you rebuild/reinstall the binary, you may need to re-grant Full Disk Access
   - After granting access, restart any running launch agents: `launchctl stop com.user.imessage-archiver && launchctl start com.user.imessage-archiver`

2. **Permission Denied (SSH)**: Ensure SSH key has correct permissions (600)
   ```bash
   chmod 600 ~/.ssh/your-private-key
   ```

3. **imessage-exporter Not Found**: Install via Homebrew or add to PATH
   ```bash
   brew install imessage-exporter
   # or ensure it's in your PATH
   ```

4. **Network Issues**: Check SSH connectivity and rsync availability
   ```bash
   # Test SSH connection
   ssh -i ~/.ssh/your-key user@your-server

   # Test rsync availability
   which rsync
   ```

5. **Launch Agent Not Running**: Check system logs and service status
   ```bash
   # Check if service is loaded
   launchctl list | grep imessage-archiver

   # View logs
   log show --predicate 'subsystem == "com.user.imessage-archiver"' --last 1h
   ```

### Debug Mode

To get detailed logging output:

1. **Set debug logging in config:**
   ```yaml
   logging_level: "debug"
   ```

2. **Run manually to see output:**
   ```bash
   ./bin/imessage-archiver -config config.yaml
   ```

3. **Check launch agent logs:**
   ```bash
   # View recent logs
   log show --predicate 'subsystem contains "imessage"' --last 30m

   # Or check the launch agent log files directly
   tail -f ~/Library/Logs/imessage-archiver.log
   ```

### Full Disk Access via CLI (Advanced)

While Full Disk Access typically requires manual GUI steps, you can check current permissions:

```bash
# Check if the binary has Full Disk Access
sqlite3 ~/Library/Messages/chat.db "SELECT COUNT(*) FROM message LIMIT 1" 2>/dev/null && echo "Access granted" || echo "Access denied"
```

**Note:** macOS security requires manual approval through System Settings for Full Disk Access. There is no reliable CLI method to programmatically grant this permission.

### Rebuild/Reinstall Process

When rebuilding the binary, Full Disk Access must be re-granted:

```bash
# Build new binary
make build

# Important: Re-grant Full Disk Access in System Settings
# System Settings > Privacy & Security > Full Disk Access
# Remove old entry and add the newly built binary
```

### Getting Help

When reporting issues, please include:
1. Operating system version
2. Error messages from logs (`~/Library/Logs/com.imessagearchiver.*.log`)
3. Your configuration file (with sensitive data redacted)
4. Output from manual test run with debug logging enabled

### System Logs

For deeper system-level issues, check Console.app:
1. Open **Console.app**
2. Search for "imessage-archiver" or "launchd"
3. Look for permission denied or other system errors

### Reinstallation After Binary Updates

**Important:** If you rebuild or update the binary, you may need to:

1. **Re-grant Full Disk Access** (due to code signing changes)
2. **Reload the launch agent:**
```bash
launchctl unload ~/Library/LaunchAgents/com.imessagearchiver.plist
launchctl load ~/Library/LaunchAgents/com.imessagearchiver.plist
```

### Getting Help

If issues persist:
1. Run with debug logging enabled
2. Check all log files mentioned above
3. Verify all permissions and file paths
4. Test SSH connectivity manually
5. Ensure `imessage-exporter` works independently

## License

This project is open source. Please check the license file for details.
