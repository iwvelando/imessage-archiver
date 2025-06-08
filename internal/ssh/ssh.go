package ssh

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// SSHConfig holds the configuration for SSH connections.
type SSHConfig struct {
	User       string
	PrivateKey string
	RemoteHost string
	RemotePath string
}

// NewSSHConfig creates a new SSHConfig instance.
func NewSSHConfig(user, privateKey, remoteHost, remotePath string) *SSHConfig {
	return &SSHConfig{
		User:       user,
		PrivateKey: privateKey,
		RemoteHost: remoteHost,
		RemotePath: remotePath,
	}
}

// ExecuteCommand executes a command on the remote server via SSH.
func (config *SSHConfig) ExecuteCommand(command string) error {
	cmd := exec.Command("ssh", "-i", config.PrivateKey, fmt.Sprintf("%s@%s", config.User, config.RemoteHost), command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute command: %s, output: %s", err, output)
	}
	return nil
}

// Rsync transfers files to the remote server using rsync.
func (config *SSHConfig) Rsync(localPath string) error {
	remotePath := filepath.Join(config.RemotePath, localPath)
	cmd := exec.Command("rsync", "-avz", "-e", fmt.Sprintf("ssh -i %s", config.PrivateKey), localPath, fmt.Sprintf("%s@%s:%s", config.User, config.RemoteHost, remotePath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to rsync files: %s, output: %s", err, output)
	}
	return nil
}
