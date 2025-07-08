# L8s Remote-Only Implementation Plan

## Overview

This document outlines the implementation plan for converting l8s to a **remote-only** container management system. After this implementation, l8s will ONLY support remote container management - local Podman connections will be explicitly disabled.

**Key Decision**: L8s will NOT support local containers. This is by design for security isolation, particularly for AI workloads.

## Architecture

```mermaid
graph LR
    subgraph "Developer Laptop"
        L8S[l8s CLI]
        SSH_AGENT[ssh-agent]
        CONFIG[~/.l8s/config.yaml]
    end
    
    subgraph "Remote Server"
        subgraph "LXC Container (Fedora)"
            SSHD[SSH Daemon]
            PODMAN[Podman<br/>(running as root)]
            SOCKET[/run/podman/podman.sock]
            subgraph "Dev Containers"
                C1[Container 1]
                C2[Container 2]
                C3[Container N]
            end
        end
    end
    
    L8S -->|SSH Tunnel| SSHD
    SSHD --> PODMAN
    PODMAN --> SOCKET
    PODMAN --> C1
    PODMAN --> C2
    PODMAN --> C3
    SSH_AGENT -.->|provides auth| L8S
    CONFIG -.->|configures| L8S
```

**Note**: This diagram should be included in the README.md to clearly show the remote-only architecture with Podman running as root inside an isolated LXC container.

## Implementation Tasks

### 1. Configuration System Updates

**File**: `pkg/config/config.go`

**Changes**:
- Add remote server configuration fields
- Make remote connection mandatory
- Add validation for required fields

```go
type Config struct {
    // Remote connection settings (required)
    RemoteHost   string `yaml:"remote_host" validate:"required"`
    RemoteUser   string `yaml:"remote_user" validate:"required"`
    RemoteSocket string `yaml:"remote_socket,omitempty"`
    
    // SSH authentication (ssh-agent required)
    SSHKeyPath string `yaml:"ssh_key_path,omitempty"`
    
    // Existing fields remain unchanged
    BaseImage      string            `yaml:"base_image,omitempty"`
    ContainerUser  string            `yaml:"container_user,omitempty"`
    SSHStartPort   int               `yaml:"ssh_start_port,omitempty"`
    Labels         map[string]string `yaml:"labels,omitempty"`
}

// Add defaults
func (c *Config) SetDefaults() {
    if c.RemoteSocket == "" {
        // Default to system socket for root Podman in LXC container
        c.RemoteSocket = "/run/podman/podman.sock"
    }
    if c.SSHKeyPath == "" {
        c.SSHKeyPath = filepath.Join(GetHomeDir(), ".ssh", "id_ed25519")
    }
    // ... existing defaults
}
```

### 2. Podman Client Updates

**File**: `pkg/container/podman_client.go`

**Changes**:
- Remove local socket fallback logic
- Enforce remote connection only
- Add connection validation

```go
func NewPodmanClient() (*RealPodmanClient, error) {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        return nil, fmt.Errorf("failed to load config: %w", err)
    }
    
    // Validate remote configuration
    if cfg.RemoteHost == "" || cfg.RemoteUser == "" {
        return nil, fmt.Errorf(`l8s requires remote server configuration.

Please configure your remote server in ~/.l8s/config.yaml:

  remote_host: your-server.example.com
  remote_user: your-username

Or run 'l8s init' to set up your configuration.

Note: l8s ONLY supports remote container management for security isolation.`)
    }
    
    // Build connection string
    connectionURI := fmt.Sprintf("ssh://%s@%s%s",
        cfg.RemoteUser,
        cfg.RemoteHost,
        cfg.RemoteSocket,
    )
    
    // Verify ssh-agent is running
    if _, exists := os.LookupEnv("SSH_AUTH_SOCK"); !exists {
        return nil, fmt.Errorf(`ssh-agent is required but not running.

Please start ssh-agent and add your key:
  eval $(ssh-agent)
  ssh-add %s

l8s requires ssh-agent for secure remote connections.`, cfg.SSHKeyPath)
    }
    
    // Set SSH key if specified
    if cfg.SSHKeyPath != "" {
        os.Setenv("CONTAINER_SSHKEY", cfg.SSHKeyPath)
    }
    
    // Create connection
    ctx := context.Background()
    conn, err := bindings.NewConnection(ctx, connectionURI)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to remote Podman at %s: %w", cfg.RemoteHost, err)
    }
    
    // Test connection
    _, err = system.Info(conn, nil)
    if err != nil {
        return nil, fmt.Errorf(`failed to connect to Podman on remote server.

Connection details:
  Host: %s
  User: %s
  Socket: %s

Error: %w

Troubleshooting:
1. Verify SSH access: ssh %s@%s
2. Check Podman socket is running: systemctl --user status podman.socket
3. Ensure user has Podman permissions
4. Check ~/.l8s/config.yaml settings

For setup instructions, see: https://github.com/l8s/l8s/docs/REMOTE_SETUP.md`,
            cfg.RemoteHost, cfg.RemoteUser, cfg.RemoteSocket, err,
            cfg.RemoteUser, cfg.RemoteHost)
    }
    
    return &RealPodmanClient{conn: conn}, nil
}
```

### 3. SSH Configuration Updates

**File**: `pkg/ssh/keys.go`

**Changes**:
- Update SSH config generation to use remote hostname
- Add remote host parameter
- Add SSH connection multiplexing for performance

**SSH Connection Multiplexing Explanation**:
The `ControlMaster`, `ControlPath`, and `ControlPersist` options enable SSH connection reuse:
- **ControlMaster auto**: Automatically creates a master connection on first SSH
- **ControlPath**: Specifies where to store the control socket for the connection
- **ControlPersist 10m**: Keeps the master connection alive for 10 minutes after the last session closes

This means when you run multiple l8s commands or SSH to containers, they all reuse the same SSH tunnel instead of creating new connections each time. This significantly improves performance, especially for rapid command sequences.

```go
func GenerateSSHConfigEntry(containerName string, sshPort int, containerUser, prefix, remoteHost string) string {
    hostAlias := containerName
    if strings.HasPrefix(containerName, prefix+"-") {
        hostAlias = containerName
    }
    
    return fmt.Sprintf(`Host %s
    HostName %s
    Port %d
    User %s
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    ForwardAgent yes
    ControlMaster auto
    ControlPath ~/.ssh/control-%r@%h:%p
    ControlPersist 10m
`, hostAlias, remoteHost, sshPort, containerUser)
}

// Update AddSSHConfig to use remote host
func AddSSHConfig(name, hostname string, port int, user string) error {
    cfg, err := config.Load()
    if err != nil {
        return err
    }
    
    sshConfigPath := filepath.Join(GetHomeDir(), ".ssh", "config")
    entry := GenerateSSHConfigEntry(
        fmt.Sprintf("dev-%s", name), 
        port, 
        user, 
        "dev",
        cfg.RemoteHost, // Use remote host instead of localhost
    )
    return AddSSHConfigEntry(sshConfigPath, entry)
}
```

### 4. Init Command Updates

**File**: `cmd/commands/init.go`

**Changes**:
- Prompt for remote server details during initialization
- Validate SSH connectivity
- Test Podman connection

```go
func runInit(cmd *cobra.Command, args []string) error {
    // ... existing code ...
    
    // Prompt for remote server configuration
    fmt.Println("\n=== Remote Server Configuration ===")
    
    remoteHost, err := promptWithDefault("Remote server hostname/IP", "")
    if err != nil || remoteHost == "" {
        return fmt.Errorf("remote server hostname is required")
    }
    
    remoteUser, err := promptWithDefault("Remote server username", "podman")
    if err != nil {
        return err
    }
    
    // Test SSH connectivity
    fmt.Printf("\nTesting SSH connection to %s@%s...\n", remoteUser, remoteHost)
    testCmd := exec.Command("ssh", "-o", "ConnectTimeout=5", 
        fmt.Sprintf("%s@%s", remoteUser, remoteHost), "echo", "OK")
    if err := testCmd.Run(); err != nil {
        return fmt.Errorf("failed to connect via SSH: %w\n\nPlease ensure:\n1. SSH key is configured (ssh-copy-id %s@%s)\n2. Server is accessible\n3. User has Podman access", 
            remoteUser, remoteHost)
    }
    
    // Update config with remote settings
    cfg.RemoteHost = remoteHost
    cfg.RemoteUser = remoteUser
    
    // ... rest of init code ...
}
```

### 5. Info Command Updates

**File**: `cmd/commands/info.go`

**Changes**:
- Display SSH command with remote host

```go
func printDetailedInfo(container *container.Container, cfg *config.Config) {
    // ... existing code ...
    
    // SSH command now uses remote host
    fmt.Printf("  SSH Command:    ssh dev-%s\n", container.Name)
    fmt.Printf("  Direct SSH:     ssh -p %d %s@%s\n", 
        container.SSHPort, cfg.ContainerUser, cfg.RemoteHost)
    
    // Note: SSH connections are reused for performance via ControlMaster
    
    // ... rest of info display ...
}
```

### 6. Documentation Updates

**Files to create/update**:
- `docs/REMOTE_SETUP.md` - Server setup guide
- `README.md` - Update with remote-only information
- `docs/SECURITY.md` - Security considerations for remote setup

### 7. Testing Updates

**Changes needed**:
- Update mocks to simulate remote connections
- Add integration tests for remote connectivity
- Add connection failure test cases

## Error Handling Strategy

### Key Error Scenarios

1. **No Configuration**
   ```
   Error: l8s requires remote server configuration.
   
   Please configure your remote server in ~/.l8s/config.yaml:
     remote_host: your-server.example.com
     remote_user: your-username
   
   Or run 'l8s init' to set up your configuration.
   
   Note: l8s ONLY supports remote container management for security isolation.
   ```

2. **SSH Connection Failed**
   ```
   Error: Cannot establish SSH connection to server.
   
   Failed to connect to: user@server.example.com
   
   Troubleshooting:
   1. Test SSH manually: ssh user@server.example.com
   2. Add SSH key: ssh-copy-id user@server.example.com
   3. Check server is accessible and SSH is running
   ```

3. **Podman Socket Not Available**
   ```
   Error: Podman socket not accessible on remote server.
   
   The Podman socket at /run/podman/podman.sock is not available.
   
   On the remote server, run:
     systemctl --user enable --now podman.socket
   
   Or if using system Podman:
     sudo systemctl enable --now podman.socket
   ```

4. **Local Connection Attempt**
   ```
   Error: Local Podman connections are not supported.
   
   l8s ONLY supports remote container management for security isolation.
   Please configure a remote server in ~/.l8s/config.yaml
   ```

### Error Message Guidelines

- Always explain that l8s is remote-only
- Provide specific troubleshooting steps
- Include example commands users can run
- Reference documentation for detailed setup

## Migration Notice

**BREAKING CHANGE**: After this implementation, l8s will ONLY support remote connections. Users currently using local containers must:

1. Set up a remote Podman server
2. Configure l8s with remote server details
3. Recreate any existing containers on the remote server

**There is NO backwards compatibility mode. This is intentional for security.**
**We don't care about this, this is still early alpha and we can break things**

## Security Considerations

1. **SSH Key Management**
   - ssh-agent is REQUIRED (no passphrase storage support)
   - Keys must be added to ssh-agent before using l8s

2. **Network Security**
   - All Podman API calls go through SSH (encrypted)
   - Consider using VPN for additional security
   - Firewall the server to restrict access

3. **Container Isolation**
   - Containers are fully isolated on remote server
   - No code execution happens on developer laptops
   - Perfect for AI/untrusted code execution

## Server Setup Script

Create `scripts/setup-server.sh`:

```bash
#!/bin/bash
# L8s Server Setup Script for Root Podman in LXC Container

set -e

echo "=== L8s Server Setup (Root Podman in LXC) ==="

# Note: This script is designed for Podman running as root inside an LXC container
# The LXC container provides isolation from the host system

# Install Podman if not present
if ! command -v podman &> /dev/null; then
    echo "Installing Podman..."
    if [ -f /etc/fedora-release ]; then
        dnf install -y podman
    else
        apt-get update
        apt-get install -y podman
    fi
fi

# Enable Podman socket at system level
echo "Enabling Podman socket..."
systemctl enable --now podman.socket

# Verify socket is available
if [ ! -S /run/podman/podman.sock ]; then
    echo "ERROR: Podman socket not found at /run/podman/podman.sock"
    exit 1
fi

# Test Podman
echo "Testing Podman..."
podman version

echo ""
echo "=== Setup Complete ==="
echo "Next steps:"
echo "1. Add your SSH key: ssh-copy-id root@$(hostname)"
echo "2. Configure l8s on your laptop with:"
echo "   Remote host: $(hostname)"
echo "   Remote user: root"
echo "   Remote socket: /run/podman/podman.sock"
echo ""
echo "Note: Podman is running as root inside this LXC container,"
echo "      which provides isolation from the host system."
```

## Implementation Priority

1. **Phase 1 (Critical)**
   - Configuration system updates
   - Podman client remote connection
   - Basic testing

2. **Phase 2 (Required)**
   - SSH config generation updates
   - Init command updates
   - Documentation

3. **Phase 3 (Nice to have)**
   - Connection health checks
   - Automatic server setup
   - Connection profiles (dev/staging/prod)

## Success Criteria

- [ ] Local Podman connections are completely disabled
- [ ] All l8s commands work ONLY with remote Podman
- [ ] SSH connections use remote hostname (never localhost)
- [ ] No local Podman required on laptop
- [ ] Clear, actionable error messages for all failure modes
- [ ] Error messages always mention remote-only design
- [ ] Documentation emphasizes remote-only architecture
- [ ] Integration tests verify local connections fail appropriately
- [ ] Init command refuses to proceed without remote config

## Estimated Timeline

- Configuration and Podman client updates: 2-3 hours
- SSH and command updates: 2-3 hours
- Testing and documentation: 3-4 hours
- Total: 1-2 days of development

## Notes

- The Podman bindings already support remote connections, so most complexity is handled
- SSH key management is critical for good UX
- Consider adding connection test command: `l8s test-connection`
- Future enhancement: Support multiple remote servers/profiles
