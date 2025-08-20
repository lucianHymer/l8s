# Multiple Podman Host Support Implementation

## Overview

Add support for multiple Podman host configurations to l8s, allowing users to switch between different network endpoints (e.g., local LAN vs VPN access) to reach the same Podman server.

## Problem Statement

Users need to access their Podman host from different networks:
- Direct access when on home/office LAN (e.g., `192.168.1.100`)
- VPN or public access when remote (e.g., `10.x.x.x` or `public.example.com`)

Currently, users must manually edit their l8s config and SSH config files when switching networks.

## Solution Design

### Configuration Structure

The config file (`~/.config/l8s/config.yaml`) will support multiple named host configurations with an active host selector:

```yaml
# Active host selector (required)
active_host: "default"

# Host configurations (required, at least "default")
hosts:
  default:
    remote_host: "192.168.1.100"
    remote_user: "root"
    remote_socket: "/run/podman/podman.sock"
    ssh_key_path: "~/.ssh/id_ed25519"
    description: "Default Podman host"
    
  vpn:
    remote_host: "10.x.x.x"
    remote_user: "root"
    remote_socket: "/run/podman/podman.sock"
    ssh_key_path: "~/.ssh/id_ed25519"
    description: "Podman host via VPN"

# Shared settings (apply to all hosts)
ssh_port_start: 2200
base_image: "localhost/l8s-fedora:latest"
container_prefix: "dev"
container_user: "dev"
ssh_public_key: ""
```

### New CLI Commands

#### `l8s host` - Manage Podman host configurations

Subcommands:

##### `l8s host list`
List all configured Podman hosts, marking the active one.

```bash
$ l8s host list
* default - Default Podman host (192.168.1.100) [active]
  vpn     - Podman host via VPN (10.x.x.x)
```

Output format:
- `*` prefix indicates active host
- Show name, description, and host address
- `[active]` suffix for clarity

##### `l8s host show`
Display details of the current active host.

```bash
$ l8s host show
Active Podman host: default
  Host: 192.168.1.100
  User: root
  Socket: /run/podman/podman.sock
  SSH Key: ~/.ssh/id_ed25519
  Description: Default Podman host
```

##### `l8s host switch <name>`
Switch to a different Podman host configuration.

```bash
$ l8s host switch vpn
Switching Podman host from 'default' to 'vpn'...
Updating SSH configurations for 3 containers:
  ✓ dev-project1: 192.168.1.100 → 10.x.x.x
  ✓ dev-project2: 192.168.1.100 → 10.x.x.x
  ✓ dev-project3: 192.168.1.100 → 10.x.x.x
Switched to Podman host: vpn
```

Actions performed:
1. Validate the target host exists in config
2. Update `active_host` in config file
3. Find all l8s-managed SSH config entries
4. Update the `HostName` field in each entry
5. Save the updated config file

Options:
- `--dry-run`: Show what would change without making changes

## Implementation Details

### 1. Config Package Updates (`pkg/config/config.go`)

#### Updated Config struct:

```go
type Config struct {
    // Active host selector
    ActiveHost string `yaml:"active_host"`
    
    // Host configurations
    Hosts map[string]HostConfig `yaml:"hosts"`
    
    // Shared settings (existing fields)
    SSHPortStart    int    `yaml:"ssh_port_start"`
    BaseImage       string `yaml:"base_image"`
    ContainerPrefix string `yaml:"container_prefix"`
    SSHPublicKey    string `yaml:"ssh_public_key"`
    ContainerUser   string `yaml:"container_user"`
    DotfilesPath    string `yaml:"dotfiles_path,omitempty"`
}

type HostConfig struct {
    RemoteHost   string `yaml:"remote_host"`
    RemoteUser   string `yaml:"remote_user"`
    RemoteSocket string `yaml:"remote_socket,omitempty"`
    SSHKeyPath   string `yaml:"ssh_key_path,omitempty"`
    Description  string `yaml:"description,omitempty"`
}
```

#### New methods:

```go
// GetActiveHost returns the active host configuration
func (c *Config) GetActiveHost() (*HostConfig, error) {
    if c.ActiveHost == "" {
        return nil, fmt.Errorf("no active host configured")
    }
    
    host, exists := c.Hosts[c.ActiveHost]
    if !exists {
        return nil, fmt.Errorf("active host '%s' not found in configuration", c.ActiveHost)
    }
    
    return &host, nil
}

// SetActiveHost updates the active host
func (c *Config) SetActiveHost(name string) error {
    if _, exists := c.Hosts[name]; !exists {
        return fmt.Errorf("host '%s' not found in configuration", name)
    }
    
    c.ActiveHost = name
    return c.Save(GetConfigPath())
}

// ListHosts returns all configured hosts
func (c *Config) ListHosts() map[string]HostConfig {
    return c.Hosts
}
```

#### Update Validate() method:

```go
func (c *Config) Validate() error {
    // Validate hosts configuration
    if len(c.Hosts) == 0 {
        return fmt.Errorf("at least one host must be configured")
    }
    
    if c.ActiveHost == "" {
        return fmt.Errorf("active_host must be specified")
    }
    
    activeHost, err := c.GetActiveHost()
    if err != nil {
        return err
    }
    
    // Validate active host configuration
    if activeHost.RemoteHost == "" {
        return fmt.Errorf("remote_host is required for host '%s'", c.ActiveHost)
    }
    if activeHost.RemoteUser == "" {
        return fmt.Errorf("remote_user is required for host '%s'", c.ActiveHost)
    }
    
    // Existing validation for shared settings...
    // (SSHPortStart, BaseImage, ContainerPrefix, etc.)
    
    return nil
}
```

### 2. Update Existing Code to Use Active Host

All code that currently uses `cfg.RemoteHost`, `cfg.RemoteUser`, etc. needs to be updated:

#### In `pkg/container/podman_client.go`:

```go
func NewPodmanClient() (*RealPodmanClient, error) {
    cfg, err := config.Load(config.GetConfigPath())
    if err != nil {
        return nil, fmt.Errorf("failed to load config: %w", err)
    }
    
    // Get active host configuration
    hostCfg, err := cfg.GetActiveHost()
    if err != nil {
        return nil, fmt.Errorf("failed to get active host: %w", err)
    }
    
    // Use hostCfg.RemoteHost, hostCfg.RemoteUser, etc.
    connectionURI := fmt.Sprintf("ssh://%s@%s%s",
        hostCfg.RemoteUser,
        hostCfg.RemoteHost,
        hostCfg.RemoteSocket,
    )
    
    // ... rest of the function
}
```

#### In `pkg/ssh/keys.go`:

```go
func AddSSHConfig(name, hostname string, port int, user string) error {
    cfg, err := config.Load(config.GetConfigPath())
    if err != nil {
        return err
    }
    
    hostCfg, err := cfg.GetActiveHost()
    if err != nil {
        return err
    }
    
    sshConfigPath := filepath.Join(GetHomeDir(), ".ssh", "config")
    entry := GenerateSSHConfigEntry(
        fmt.Sprintf("dev-%s", name), 
        port, 
        user, 
        "dev",
        hostCfg.RemoteHost, // Use active host's remote_host
    )
    return AddSSHConfigEntry(sshConfigPath, entry)
}
```

### 3. New Host Command Implementation

Create `cmd/l8s/cmd_host.go`:

```go
package main

import (
    "fmt"
    "github.com/spf13/cobra"
    "l8s/pkg/cli"
)

func newHostCommand(factory *cli.CommandFactory) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "host",
        Short: "Manage Podman host configurations",
        Long:  "Manage multiple Podman host configurations for different network access scenarios",
    }
    
    cmd.AddCommand(newHostListCommand(factory))
    cmd.AddCommand(newHostShowCommand(factory))
    cmd.AddCommand(newHostSwitchCommand(factory))
    
    return cmd
}

func newHostListCommand(factory *cli.CommandFactory) *cobra.Command {
    return &cobra.Command{
        Use:   "list",
        Short: "List all configured Podman hosts",
        RunE: func(cmd *cobra.Command, args []string) error {
            return factory.CreateHostListCommand().Execute(cmd.Context())
        },
    }
}

func newHostShowCommand(factory *cli.CommandFactory) *cobra.Command {
    return &cobra.Command{
        Use:   "show",
        Short: "Show current Podman host details",
        RunE: func(cmd *cobra.Command, args []string) error {
            return factory.CreateHostShowCommand().Execute(cmd.Context())
        },
    }
}

func newHostSwitchCommand(factory *cli.CommandFactory) *cobra.Command {
    var dryRun bool
    
    cmd := &cobra.Command{
        Use:   "switch <name>",
        Short: "Switch to a different Podman host",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return factory.CreateHostSwitchCommand(args[0], dryRun).Execute(cmd.Context())
        },
    }
    
    cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would change without making changes")
    
    return cmd
}
```

### 4. CLI Handlers Implementation

Create `pkg/cli/host_handlers.go`:

```go
package cli

import (
    "context"
    "fmt"
    "path/filepath"
    "strings"
    
    "l8s/pkg/config"
    "l8s/pkg/ssh"
)

type HostListCommand struct {
    config *config.Config
}

func (c *HostListCommand) Execute(ctx context.Context) error {
    hosts := c.config.ListHosts()
    
    for name, host := range hosts {
        marker := " "
        active := ""
        if name == c.config.ActiveHost {
            marker = "*"
            active = " [active]"
        }
        
        fmt.Printf("%s %s - %s (%s)%s\n", 
            marker, name, host.Description, host.RemoteHost, active)
    }
    
    return nil
}

type HostShowCommand struct {
    config *config.Config
}

func (c *HostShowCommand) Execute(ctx context.Context) error {
    host, err := c.config.GetActiveHost()
    if err != nil {
        return err
    }
    
    fmt.Printf("Active Podman host: %s\n", c.config.ActiveHost)
    fmt.Printf("  Host: %s\n", host.RemoteHost)
    fmt.Printf("  User: %s\n", host.RemoteUser)
    fmt.Printf("  Socket: %s\n", host.RemoteSocket)
    if host.SSHKeyPath != "" {
        fmt.Printf("  SSH Key: %s\n", host.SSHKeyPath)
    }
    if host.Description != "" {
        fmt.Printf("  Description: %s\n", host.Description)
    }
    
    return nil
}

type HostSwitchCommand struct {
    config     *config.Config
    targetHost string
    dryRun     bool
}

func (c *HostSwitchCommand) Execute(ctx context.Context) error {
    // Validate target host exists
    newHost, exists := c.config.Hosts[c.targetHost]
    if !exists {
        return fmt.Errorf("host '%s' not found in configuration", c.targetHost)
    }
    
    // Get current host for comparison
    currentHost, err := c.config.GetActiveHost()
    if err != nil {
        return err
    }
    
    if c.config.ActiveHost == c.targetHost {
        fmt.Printf("Already using Podman host: %s\n", c.targetHost)
        return nil
    }
    
    fmt.Printf("Switching Podman host from '%s' to '%s'...\n", 
        c.config.ActiveHost, c.targetHost)
    
    // Find and update all SSH configs
    sshConfigPath := filepath.Join(ssh.GetHomeDir(), ".ssh", "config")
    updates, err := c.findSSHConfigUpdates(sshConfigPath, currentHost.RemoteHost, newHost.RemoteHost)
    if err != nil {
        return fmt.Errorf("failed to read SSH config: %w", err)
    }
    
    if len(updates) > 0 {
        fmt.Printf("Updating SSH configurations for %d containers:\n", len(updates))
        
        if !c.dryRun {
            for _, container := range updates {
                err := c.updateSSHConfigEntry(sshConfigPath, container, newHost.RemoteHost)
                if err != nil {
                    fmt.Printf("  ✗ %s: %v\n", container, err)
                } else {
                    fmt.Printf("  ✓ %s: %s → %s\n", 
                        container, currentHost.RemoteHost, newHost.RemoteHost)
                }
            }
        } else {
            for _, container := range updates {
                fmt.Printf("  Would update %s: %s → %s\n", 
                    container, currentHost.RemoteHost, newHost.RemoteHost)
            }
        }
    }
    
    if !c.dryRun {
        // Update active host in config
        if err := c.config.SetActiveHost(c.targetHost); err != nil {
            return fmt.Errorf("failed to update config: %w", err)
        }
        
        fmt.Printf("Switched to Podman host: %s\n", c.targetHost)
    } else {
        fmt.Printf("Would switch to Podman host: %s\n", c.targetHost)
    }
    
    return nil
}

func (c *HostSwitchCommand) findSSHConfigUpdates(configPath, oldHost, newHost string) ([]string, error) {
    // Parse SSH config to find all l8s-managed entries
    // Look for entries with pattern "dev-*" and matching HostName
    // Return list of container names that need updating
    
    // Implementation would parse ~/.ssh/config and find relevant entries
    // This is simplified - actual implementation needs proper SSH config parsing
    
    var containers []string
    // ... parse SSH config and populate containers list
    
    return containers, nil
}

func (c *HostSwitchCommand) updateSSHConfigEntry(configPath, container, newHost string) error {
    // Update the HostName field for the given container in SSH config
    // Preserve all other settings
    
    // Implementation would:
    // 1. Read the SSH config
    // 2. Find the Host block for the container
    // 3. Update only the HostName line
    // 4. Write back the config
    
    return nil
}
```

### 5. Update CommandFactory

Add new methods to `pkg/cli/factory.go`:

```go
func (f *CommandFactory) CreateHostListCommand() *HostListCommand {
    return &HostListCommand{
        config: f.config,
    }
}

func (f *CommandFactory) CreateHostShowCommand() *HostShowCommand {
    return &HostShowCommand{
        config: f.config,
    }
}

func (f *CommandFactory) CreateHostSwitchCommand(targetHost string, dryRun bool) *HostSwitchCommand {
    return &HostSwitchCommand{
        config:     f.config,
        targetHost: targetHost,
        dryRun:     dryRun,
    }
}
```

## Testing Requirements

### Unit Tests

1. **Config package tests** (`pkg/config/config_test.go`):
   - Test loading config with multiple hosts
   - Test GetActiveHost() with valid/invalid hosts
   - Test SetActiveHost() 
   - Test validation with missing/invalid host configurations

2. **Host command tests** (`pkg/cli/host_handlers_test.go`):
   - Test list command output formatting
   - Test show command with different configurations
   - Test switch command with dry-run
   - Test SSH config update logic

### Integration Tests

1. Test full workflow:
   - Create containers with default host
   - Switch to different host
   - Verify SSH configs are updated
   - Verify can still connect to containers

2. Test error cases:
   - Switching to non-existent host
   - Invalid host configuration
   - SSH config update failures

## Migration Guide

Users need to manually update their config from:

```yaml
# Old format
remote_host: "192.168.1.100"
remote_user: "root"
remote_socket: "/run/podman/podman.sock"
ssh_key_path: "~/.ssh/id_ed25519"
ssh_port_start: 2200
# ... other settings
```

To:

```yaml
# New format
active_host: "default"

hosts:
  default:
    remote_host: "192.168.1.100"
    remote_user: "root"
    remote_socket: "/run/podman/podman.sock"
    ssh_key_path: "~/.ssh/id_ed25519"
    description: "Default Podman host"

# Shared settings remain at root level
ssh_port_start: 2200
# ... other settings
```

## Documentation Updates

1. Update README.md to document the new multi-host feature
2. Update configuration section to show new structure
3. Add examples of common multi-host scenarios
4. Update troubleshooting section for host-switching issues

## Implementation Order

1. Update config package with new structures and methods
2. Update all existing code to use GetActiveHost()
3. Implement host command handlers
4. Add host command to CLI
5. Implement SSH config update logic
6. Add tests
7. Update documentation