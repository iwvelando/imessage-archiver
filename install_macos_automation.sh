#!/bin/bash

# Installation script for iMessage Archiver macOS Automation

# --- Configuration ---
# !!! IMPORTANT: Update these paths if your setup differs !!!
# Path to the compiled imessage-archiver binary
BINARY_PATH="$HOME/bin/imessage-archiver"
# Path to your imessage-archiver configuration file
CONFIG_PATH="$HOME/.config/imessage-archiver/config.yaml"
# Name of the plist file (should match the one in this repository)
PLIST_NAME="com.imessagearchiver.plist"
# Destination for the plist file
LAUNCH_AGENTS_DIR="$HOME/Library/LaunchAgents"
# Full path to the plist file in the LaunchAgents directory
DEST_PLIST_PATH="$LAUNCH_AGENTS_DIR/$PLIST_NAME"
# Source plist file (assuming this script is run from the project root)
SOURCE_PLIST_PATH="./$PLIST_NAME"

# --- Helper Functions ---
echo_info() {
    echo "[INFO] $1"
}

echo_warn() {
    echo "[WARN] $1"
}

echo_error() {
    echo "[ERROR] $1"
    exit 1
}

# --- Pre-flight Checks ---
echo_info "Starting iMessage Archiver macOS automation setup..."

# 1. Check if binary exists
if [ ! -f "$BINARY_PATH" ]; then
    echo_error "iMessage Archiver binary not found at $BINARY_PATH. Please build the project (e.g., using 'make build' or 'go build') and ensure the binary is in the correct location (or update BINARY_PATH in this script)."
fi
echo_info "Binary found at $BINARY_PATH."

# 2. Check if config file exists
if [ ! -f "$CONFIG_PATH" ]; then
    echo_warn "iMessage Archiver config file not found at $CONFIG_PATH."
    echo_warn "The launch agent will likely fail until this file is created and configured."
    echo_warn "Please create it based on the example 'config.yaml' in the repository."
    # To make the script proceed, we'll continue, but the agent won't work without the config.
fi
echo_info "Config path check complete (expected at $CONFIG_PATH)."

# 3. Check if source plist file exists
if [ ! -f "$SOURCE_PLIST_PATH" ]; then
    echo_error "Source plist file '$SOURCE_PLIST_PATH' not found. Make sure this script is run from the project root directory where '$PLIST_NAME' is located."
fi
echo_info "Source plist file found at $SOURCE_PLIST_PATH."

# --- Installation ---
echo_info "Installing launch agent..."

# 1. Create LaunchAgents directory if it doesn't exist
if [ ! -d "$LAUNCH_AGENTS_DIR" ]; then
    echo_info "Creating LaunchAgents directory: $LAUNCH_AGENTS_DIR"
    mkdir -p "$LAUNCH_AGENTS_DIR"
    if [ $? -ne 0 ]; then
        echo_error "Failed to create directory $LAUNCH_AGENTS_DIR."
    fi
fi

# 2. Copy the plist file
echo_info "Copying $PLIST_NAME to $DEST_PLIST_PATH..."
cp "$SOURCE_PLIST_PATH" "$DEST_PLIST_PATH"
if [ $? -ne 0 ]; then
    echo_error "Failed to copy plist file."
fi

# 3. Replace placeholders in the copied plist file
echo_info "Customizing plist file for user $USER ($HOME)..."
# Using a temporary file for sed to avoid issues with in-place editing on macOS
TMP_PLIST_PATH="$DEST_PLIST_PATH.tmp"
sed "s|__HOME_DIR__|$HOME|g" "$DEST_PLIST_PATH" > "$TMP_PLIST_PATH"
if [ $? -ne 0 ]; then
    echo_error "Failed to customize plist file with user's home directory."
fi
mv "$TMP_PLIST_PATH" "$DEST_PLIST_PATH"
if [ $? -ne 0 ]; then
    echo_error "Failed to move customized plist file into place."
fi

# 4. Set correct permissions for the plist file
chmod 644 "$DEST_PLIST_PATH"
if [ $? -ne 0 ]; then
    echo_warn "Failed to set permissions for $DEST_PLIST_PATH. This might cause issues."
fi

# 5. Unload the agent if it's already loaded (to ensure changes are picked up)
echo_info "Attempting to unload existing agent (if any)..."
launchctl unload "$DEST_PLIST_PATH" 2>/dev/null # Errors are fine if it wasn't loaded

# 6. Load the launch agent
echo_info "Loading launch agent: $DEST_PLIST_PATH"
launchctl load "$DEST_PLIST_PATH"
if [ $? -ne 0 ]; then
    echo_error "Failed to load launch agent. Check system logs for more details (Console.app)."
fi

echo_info "iMessage Archiver macOS automation has been installed and loaded."
echo_info "It is scheduled to run daily at the time specified in $PLIST_NAME."
echo_info "Output logs will be stored in $HOME/Library/Logs/"

# --- Uninstallation Instructions ---
echo_info ""
echo_info "To uninstall this automation:"
echo_info "1. Unload the agent: launchctl unload $DEST_PLIST_PATH"
echo_info "2. Remove the plist file: rm $DEST_PLIST_PATH"
echo_info "3. (Optional) Remove the binary: rm $BINARY_PATH"
echo_info "4. (Optional) Remove the config: rm -r $(dirname "$CONFIG_PATH")"
echo_info "5. (Optional) Remove the logs: rm $HOME/Library/Logs/com.imessagearchiver.*.log"

exit 0
