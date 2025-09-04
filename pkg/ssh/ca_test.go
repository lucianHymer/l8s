package ssh

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCA(t *testing.T) {
	tempDir := t.TempDir()
	
	ca, err := NewCA(tempDir)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}
	
	expectedPrivate := filepath.Join(tempDir, "ca", "ca_key")
	expectedPublic := filepath.Join(tempDir, "ca", "ca_key.pub")
	
	if ca.PrivateKeyPath != expectedPrivate {
		t.Errorf("Expected private key path %s, got %s", expectedPrivate, ca.PrivateKeyPath)
	}
	
	if ca.PublicKeyPath != expectedPublic {
		t.Errorf("Expected public key path %s, got %s", expectedPublic, ca.PublicKeyPath)
	}
}

func TestCAGenerate(t *testing.T) {
	tempDir := t.TempDir()
	
	ca, err := NewCA(tempDir)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}
	
	// Generate CA keypair
	if err := ca.Generate(); err != nil {
		t.Fatalf("Failed to generate CA: %v", err)
	}
	
	// Check that files exist
	if !ca.Exists() {
		t.Error("CA files do not exist after generation")
	}
	
	// Check file permissions
	info, err := os.Stat(ca.PrivateKeyPath)
	if err != nil {
		t.Fatalf("Failed to stat private key: %v", err)
	}
	
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("Private key has incorrect permissions: %o (expected 0600)", perm)
	}
	
	// Check public key format
	pubKey, err := ca.GetPublicKey()
	if err != nil {
		t.Fatalf("Failed to get public key: %v", err)
	}
	
	if !strings.HasPrefix(pubKey, "ssh-ed25519") {
		t.Errorf("Public key has unexpected format: %s", pubKey)
	}
	
	// Ensure comment contains hostname
	if !strings.Contains(pubKey, "l8s-ca@") {
		t.Error("Public key comment does not contain expected CA identifier")
	}
}

func TestCAGenerateAlreadyExists(t *testing.T) {
	tempDir := t.TempDir()
	
	ca, err := NewCA(tempDir)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}
	
	// Generate CA keypair
	if err := ca.Generate(); err != nil {
		t.Fatalf("Failed to generate CA: %v", err)
	}
	
	// Try to generate again - should fail
	if err := ca.Generate(); err == nil {
		t.Error("Expected error when generating CA that already exists")
	} else if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestCASignHostKey(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create CA
	ca, err := NewCA(tempDir)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}
	
	if err := ca.Generate(); err != nil {
		t.Fatalf("Failed to generate CA: %v", err)
	}
	
	// Generate a test host key
	hostKeyPath := filepath.Join(tempDir, "test_host_key")
	if err := generateTestHostKey(hostKeyPath); err != nil {
		t.Fatalf("Failed to generate test host key: %v", err)
	}
	
	// Sign the host key
	if err := ca.SignHostKey(hostKeyPath, "test-container", "test.example.com"); err != nil {
		t.Fatalf("Failed to sign host key: %v", err)
	}
	
	// Check that certificate was created
	certPath := hostKeyPath + "-cert.pub"
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		t.Error("Certificate file was not created")
	}
	
	// Read certificate and verify it's not empty
	certData, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("Failed to read certificate: %v", err)
	}
	
	if len(certData) == 0 {
		t.Error("Certificate file is empty")
	}
	
	// Certificate should contain the signed key type
	if !strings.Contains(string(certData), "ssh-ed25519-cert") {
		t.Error("Certificate does not contain expected certificate type")
	}
}

func TestCASignHostKeyNoCA(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create CA but don't generate keys
	ca, err := NewCA(tempDir)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}
	
	// Generate a test host key
	hostKeyPath := filepath.Join(tempDir, "test_host_key")
	if err := generateTestHostKey(hostKeyPath); err != nil {
		t.Fatalf("Failed to generate test host key: %v", err)
	}
	
	// Try to sign without CA existing
	if err := ca.SignHostKey(hostKeyPath, "test-container", "test.example.com"); err == nil {
		t.Error("Expected error when signing with non-existent CA")
	} else if !strings.Contains(err.Error(), "CA key not found") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestCAWriteKnownHostsEntry(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create and generate CA
	ca, err := NewCA(tempDir)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}
	
	if err := ca.Generate(); err != nil {
		t.Fatalf("Failed to generate CA: %v", err)
	}
	
	// Write known_hosts entry
	knownHostsPath := filepath.Join(tempDir, "known_hosts")
	if err := ca.WriteKnownHostsEntry(knownHostsPath, "test.example.com"); err != nil {
		t.Fatalf("Failed to write known_hosts entry: %v", err)
	}
	
	// Read and verify known_hosts
	data, err := os.ReadFile(knownHostsPath)
	if err != nil {
		t.Fatalf("Failed to read known_hosts: %v", err)
	}
	
	content := string(data)
	
	// Check for cert-authority marker
	if !strings.Contains(content, "@cert-authority") {
		t.Error("known_hosts does not contain @cert-authority marker")
	}
	
	// Check for host pattern
	if !strings.Contains(content, "dev-*") {
		t.Error("known_hosts does not contain dev-* pattern")
	}
	
	// Check for remote host (bracketed format for non-standard ports)
	if !strings.Contains(content, "[test.example.com]:*") {
		t.Error("known_hosts does not contain remote host pattern")
	}
	
	// Check for public key
	pubKey, _ := ca.GetPublicKey()
	if !strings.Contains(content, pubKey) {
		t.Error("known_hosts does not contain CA public key")
	}
}

func TestCAWriteKnownHostsEntryIdempotent(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create and generate CA
	ca, err := NewCA(tempDir)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}
	
	if err := ca.Generate(); err != nil {
		t.Fatalf("Failed to generate CA: %v", err)
	}
	
	knownHostsPath := filepath.Join(tempDir, "known_hosts")
	
	// Write entry twice
	if err := ca.WriteKnownHostsEntry(knownHostsPath, "test.example.com"); err != nil {
		t.Fatalf("Failed to write known_hosts entry: %v", err)
	}
	
	if err := ca.WriteKnownHostsEntry(knownHostsPath, "test.example.com"); err != nil {
		t.Fatalf("Failed to write known_hosts entry second time: %v", err)
	}
	
	// Read and verify there's only one entry
	data, err := os.ReadFile(knownHostsPath)
	if err != nil {
		t.Fatalf("Failed to read known_hosts: %v", err)
	}
	
	// Count occurrences of the public key
	pubKey, _ := ca.GetPublicKey()
	count := strings.Count(string(data), pubKey)
	
	if count != 1 {
		t.Errorf("Expected 1 occurrence of public key in known_hosts, got %d", count)
	}
}

func TestCAExists(t *testing.T) {
	tempDir := t.TempDir()
	
	ca, err := NewCA(tempDir)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}
	
	// Should not exist initially
	if ca.Exists() {
		t.Error("CA should not exist before generation")
	}
	
	// Generate CA
	if err := ca.Generate(); err != nil {
		t.Fatalf("Failed to generate CA: %v", err)
	}
	
	// Should exist after generation
	if !ca.Exists() {
		t.Error("CA should exist after generation")
	}
	
	// Remove public key and check again
	os.Remove(ca.PublicKeyPath)
	if ca.Exists() {
		t.Error("CA should not exist when public key is missing")
	}
	
	// Restore public key, remove private key
	if err := ca.Generate(); err == nil {
		t.Error("Should not be able to regenerate when private key exists")
	}
	
	os.Remove(ca.PrivateKeyPath)
	if ca.Exists() {
		t.Error("CA should not exist when private key is missing")
	}
}

func TestCAGetPublicKey(t *testing.T) {
	tempDir := t.TempDir()
	
	ca, err := NewCA(tempDir)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}
	
	// Should fail when CA doesn't exist
	if _, err := ca.GetPublicKey(); err == nil {
		t.Error("Expected error when getting public key from non-existent CA")
	}
	
	// Generate CA
	if err := ca.Generate(); err != nil {
		t.Fatalf("Failed to generate CA: %v", err)
	}
	
	// Should succeed after generation
	pubKey, err := ca.GetPublicKey()
	if err != nil {
		t.Fatalf("Failed to get public key: %v", err)
	}
	
	// Verify format
	if !strings.HasPrefix(pubKey, "ssh-ed25519") {
		t.Errorf("Public key has unexpected format: %s", pubKey)
	}
	
	// Should not have trailing newline
	if strings.HasSuffix(pubKey, "\n") {
		t.Error("Public key should not have trailing newline")
	}
}

// Helper function to generate a test host key
func generateTestHostKey(path string) error {
	// Use os/exec to run ssh-keygen for test
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", path, "-N", "", "-C", "test")
	return cmd.Run()
}