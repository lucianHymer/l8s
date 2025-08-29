package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CA represents an SSH Certificate Authority
type CA struct {
	PrivateKeyPath string
	PublicKeyPath  string
}

// NewCA creates a new CA instance with paths configured
func NewCA(configDir string) (*CA, error) {
	caDir := filepath.Join(configDir, "ca")
	return &CA{
		PrivateKeyPath: filepath.Join(caDir, "ca_key"),
		PublicKeyPath:  filepath.Join(caDir, "ca_key.pub"),
	}, nil
}

// Generate creates a new CA keypair
func (ca *CA) Generate() error {
	// Create CA directory with proper permissions
	caDir := filepath.Dir(ca.PrivateKeyPath)
	if err := os.MkdirAll(caDir, 0700); err != nil {
		return fmt.Errorf("failed to create CA directory: %w", err)
	}

	// Check if CA already exists
	if ca.Exists() {
		return fmt.Errorf("CA already exists at %s", ca.PrivateKeyPath)
	}

	// Get hostname for comment
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	// Generate CA keypair using ssh-keygen
	cmd := exec.Command("ssh-keygen",
		"-t", "ed25519",
		"-f", ca.PrivateKeyPath,
		"-C", fmt.Sprintf("l8s-ca@%s", hostname),
		"-N", "") // No passphrase for automation

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to generate CA keypair: %w\nOutput: %s", err, output)
	}

	// Set proper permissions on CA private key
	if err := os.Chmod(ca.PrivateKeyPath, 0600); err != nil {
		return fmt.Errorf("failed to set CA private key permissions: %w", err)
	}

	// Set proper permissions on CA public key
	if err := os.Chmod(ca.PublicKeyPath, 0644); err != nil {
		return fmt.Errorf("failed to set CA public key permissions: %w", err)
	}

	return nil
}

// SignHostKey signs an SSH host key with the CA certificate
func (ca *CA) SignHostKey(hostKeyPath, containerName, remoteHost string) error {
	// Check if CA exists
	if !ca.Exists() {
		return fmt.Errorf("CA key not found. Run 'l8s init' to generate")
	}

	// Prepare certificate output path
	certPath := hostKeyPath + "-cert.pub"

	// Prepare principals (valid hostnames for the certificate)
	// Include both the container name and the remote host with wildcard ports
	principals := fmt.Sprintf("%s-%s,%s", "dev", containerName, remoteHost)

	// Sign the host key with CA
	cmd := exec.Command("ssh-keygen",
		"-s", ca.PrivateKeyPath,           // Sign with CA private key
		"-I", fmt.Sprintf("dev-%s", containerName), // Certificate ID
		"-h",                               // Host certificate (not user)
		"-V", "+3650d",                    // Valid for 10 years
		"-n", principals,                   // Valid principals (hostnames)
		hostKeyPath+".pub")                 // Public key to sign

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to sign host key: %w\nOutput: %s", err, output)
	}

	// Verify certificate was created
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return fmt.Errorf("certificate was not created at %s", certPath)
	}

	return nil
}

// WriteKnownHostsEntry writes the CA public key to known_hosts format
func (ca *CA) WriteKnownHostsEntry(knownHostsPath, remoteHost string) error {
	// Read CA public key
	pubKeyData, err := os.ReadFile(ca.PublicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read CA public key: %w", err)
	}

	pubKey := strings.TrimSpace(string(pubKeyData))

	// Create known_hosts entry for CA
	// This tells SSH to trust any host key signed by this CA for l8s containers
	// Use bracketed format for non-standard ports (SSH requires [host]:port format)
	entry := fmt.Sprintf("@cert-authority dev-*,[%s]:* %s\n", remoteHost, pubKey)

	// Ensure directory exists
	dir := filepath.Dir(knownHostsPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create known_hosts directory: %w", err)
	}

	// Check if file exists and already contains the entry
	if data, err := os.ReadFile(knownHostsPath); err == nil {
		if strings.Contains(string(data), pubKey) {
			// CA entry already exists
			return nil
		}
	}

	// Append CA entry to known_hosts
	file, err := os.OpenFile(knownHostsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open known_hosts file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(entry); err != nil {
		return fmt.Errorf("failed to write known_hosts entry: %w", err)
	}

	return nil
}

// Exists checks if the CA keypair exists
func (ca *CA) Exists() bool {
	// Check both private and public keys exist
	if _, err := os.Stat(ca.PrivateKeyPath); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(ca.PublicKeyPath); os.IsNotExist(err) {
		return false
	}
	return true
}

// GetPublicKey reads and returns the CA public key
func (ca *CA) GetPublicKey() (string, error) {
	if !ca.Exists() {
		return "", fmt.Errorf("CA does not exist")
	}

	data, err := os.ReadFile(ca.PublicKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read CA public key: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}