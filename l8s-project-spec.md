# L8s (Lebowskis) - Project Specification

> "The container management system that really ties the room together"

## Overview

`l8s` (or "Lebowskis") is a Podman-based development container management tool that creates isolated, SSH-accessible development environments. Each container is a fully-featured Linux environment with development tools, accessible via SSH using key-based authentication.

## Core Philosophy

- **Simple**: The Dude abides by simplicity
- **Secure**: SSH key-only authentication (no passwords)
- **Stateful**: Containers persist between sessions
- **Git-Integrated**: Automatic repository cloning and remote management

## Technical Requirements

### Language & Dependencies
- **Language**: Go 1.21+
- **CLI Framework**: spf13/cobra
- **Container Runtime**: Podman 4.0+
- **Key Dependencies**:
  - `github.com/containers/podman/v4/pkg/bindings`
  - `github.com/spf13/cobra`
  - `github.com/spf13/viper` (configuration)
  - `golang.org/x/crypto/ssh` (SSH key handling)

### System Requirements
- Linux (primary target: Fedora LXC)
- Podman installed and configured
- SSH client
- Git

## Command Specification

### Core Commands

#### `l8s create <name> <git-url> [branch]`
Creates a new development container.

**Process**:
1. Validate container name doesn't exist
2. Find available SSH port (starting at 2200)
3. Create container from base image with:
   - Mapped SSH port
   - Persistent volumes (home, workspace)
   - Hostname set to container name
   - Metadata labels for state tracking
4. Copy dotfiles from l8s/dotfiles/ to container's /home/dev/
5. Inject user's SSH public key
6. Clone specified git repository
7. Add SSH config entry to ~/.ssh/config
8. Configure git remote using SSH config host
9. Display connection information

**SSH Config Management**:
Automatically append to `~/.ssh/config`:
```
Host dev-myproject
    HostName localhost
    Port 2200
    User dev
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
```

This enables:
- `ssh dev-myproject` instead of `ssh -p 2200 dev@localhost`
- `git push myproject` with remote `dev-myproject:/workspace/project`
- `scp file.txt dev-myproject:` for file transfers
- VS Code Remote SSH: `code --remote ssh-remote+dev-myproject /workspace/project`

**Example**:
```bash
$ l8s create myproject https://github.com/user/repo.git main
🎳 Creating container: dev-myproject
✓ SSH port: 2200
✓ Repository cloned
✓ SSH config entry added
✓ Git remote 'myproject' added (dev-myproject:/workspace/project)

Connection options:
- l8s ssh myproject
- ssh dev-myproject
- git push myproject
```

#### `l8s ssh <name>`
Connect to container via SSH.

**Example**:
```bash
$ l8s ssh myproject
"The Dude abides... connecting..."
[connects via SSH]
```

#### `l8s list`
List all l8s containers with status.

**Implementation**:
```go
// Query all containers with dev- prefix
// Extract metadata from labels
// Format output
```

**Output format**:
```
NAME        STATUS      SSH PORT    GIT REMOTE    CREATED
myproject   running     2200        ✓            2h ago
another     stopped     2201        ✗            3d ago
```

#### `l8s stop <name>`
Stop a running container.

#### `l8s start <name>`
Start a stopped container.

#### `l8s remove <name>`
Remove container and optionally its volumes.

**Process**:
1. Confirm removal
2. Remove git remote from host
3. Stop container if running
4. Remove container
5. Optionally remove volumes

**Example**:
```bash
$ l8s remove myproject
Remove container dev-myproject and volumes? (y/N): y
✓ Git remote removed
✓ Container removed
✓ Volumes removed
```

#### `l8s info <name>`
Show detailed container information.

**Output**:
- Container name and status
- SSH connection options (port-mapped and direct IP)
- Git repository information
- Volume mount points
- SSH config snippet

#### `l8s build`
Build or rebuild the base container image.

### Advanced Commands

#### `l8s remote add <name>`
Add git remote for existing container.

#### `l8s remote remove <name>`
Remove git remote for container.

#### `l8s exec <name> <command>`
Execute command in container (wrapper around podman exec).

## Container Specification

### Base Image (Containerfile)
The base image should include:
- Fedora latest
- OpenSSH server (configured for key-only auth)
- Development tools:
  - git, neovim, zsh, tmux
  - gcc, make, python3, nodejs, npm
  - ripgrep, fd, fzf, bat
  - @anthropic-ai/claude-code (npm global)
- Oh-my-zsh pre-installed
- User created based on config (not hardcoded to 'dev')
- Sudo access for configured user

### Container Configuration
- **Naming**: `dev-<name>`
- **User**: `dev` (uid 1000)
- **Volumes**:
  - `dev-<name>-home`: Mounted at `/home/dev`
  - `dev-<name>-workspace`: Mounted at `/workspace`
- **Network**: Default podman network
- **SSH**: Port 22 mapped to host port 220X

## Configuration

### Config File Location
- `~/.config/l8s/config.yaml`

### Configurable Options
```yaml
# Default SSH port range
ssh_port_start: 2200

# Container image
base_image: "localhost/l8s-fedora:latest"

# Container prefix
container_prefix: "dev"

# SSH key location (auto-detected if not specified)
ssh_public_key: ""  # Empty means auto-detect

# Container username (IMPORTANT: configurable!)
container_user: "dev"  # Can be set to "lucian" or any preferred username
```

## Git Integration Features

### Automatic Remote Management
When creating a container:
1. Add remote: `git remote add <name> ssh://dev@localhost:PORT/workspace/project`
2. Set upstream: `git branch --set-upstream-to=<name>/<branch>`

When removing a container:
1. Change upstream back to origin
2. Remove remote

### Use Case
```bash
# Work on laptop
git commit -m "WIP: feature"
git push myproject  # Pushes to container

# SSH to container and test
l8s ssh myproject
# Changes are already there!
```

## Error Handling

### User-Friendly Errors
- **Invalid arguments**: "You're out of your element! Usage: l8s create <name> <git-url>"
- **Container exists**: "Container 'name' already exists"
- **Container not found**: "Container 'name' not found"
- **SSH key missing**: "No SSH public key found in ~/.ssh/"
- **Podman not found**: "Podman is not installed"
- **Build failed**: "Container build failed: [error details]"

## Testing Requirements

### Unit Tests ✅
- Command parsing and validation
- Port allocation logic
- Git remote management
- Configuration loading

### Integration Tests ✅
- Container lifecycle (create, start, stop, remove)
- SSH connectivity
- Git operations
- Volume persistence

### Test Container
Minimal test image for faster testing cycles.

## Implementation Status

### Test Suite (TDD) ✅
The following test files have been created following Test-Driven Development:

1. **Container Manager Tests** (`pkg/container/manager_test.go`)
   - Container creation with validation
   - Lifecycle operations (start, stop, remove)
   - Port allocation and SSH configuration
   - Comprehensive error handling

2. **SSH Key Handling Tests** (`pkg/ssh/keys_test.go`)
   - SSH public key reading and validation
   - SSH config file management (add/remove entries)
   - Authorized keys generation
   - Port availability checking

3. **Git Remote Management Tests** (`pkg/git/remote_test.go`)
   - Repository cloning with branch support
   - Remote add/remove operations
   - Upstream branch configuration
   - Git URL validation

4. **CLI Command Tests** (`cmd/commands/create_test.go`)
   - All command parsing and validation
   - Mock-based testing for CLI operations
   - Error handling and user feedback testing

5. **Integration Tests** (`test/integration/container_lifecycle_test.go`)
   - Full container lifecycle testing
   - SSH connectivity verification
   - Git operations with containers
   - Multiple container management
   - Port allocation testing

6. **Build System** (`Makefile`)
   - Comprehensive build targets
   - Test execution (unit and integration)
   - Coverage reporting
   - Linting and formatting

### Running Tests
```bash
# Download dependencies
go mod download

# Run unit tests
make test

# Run integration tests (requires Podman)
make test-integration

# Run all tests with coverage
make test-coverage

# Run linting
make lint
```

## Project Structure
```
l8s/
├── cmd/
│   ├── l8s/
│   │   └── main.go
│   └── commands/
│       ├── create.go
│       ├── ssh.go
│       ├── list.go
│       └── ...
├── pkg/
│   ├── container/
│   │   ├── manager.go
│   │   └── manager_test.go
│   ├── git/
│   │   ├── remote.go
│   │   └── remote_test.go
│   └── ssh/
│       ├── keys.go
│       └── keys_test.go
├── containers/
│   ├── Containerfile
│   └── Containerfile.test
├── dotfiles/                    # Static dotfiles copied to containers
│   ├── .zshrc
│   ├── .tmux.conf
│   ├── .gitconfig
│   └── .config/
│       ├── nvim/
│       │   ├── init.lua
│       │   └── lua/
│       │       ├── plugins.lua
│       │       └── settings.lua
│       └── claude/
│           └── config.yaml
├── scripts/
│   ├── install.sh
│   └── test.sh
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── LICENSE
```

## Future Enhancements (Post-MVP)

1. **Template System**: Language-specific container images
2. **MCP Server**: Expose container management via Model Context Protocol
3. **Dotfile Management**: Automatic dotfile synchronization
4. **Multi-Host**: Support for remote Podman hosts
5. **Web UI**: Optional web dashboard
6. **Backup/Restore**: Container state snapshots

## Success Metrics

- Single binary installation
- Container creation under 30 seconds
- Zero-password SSH access
- Git remotes "just work"
- Clear, helpful error messages

---

*"This container really ties the room together."* 🎳
