# 🎳 Lebowskis (aka `l8s`)

> "The container management system that really ties the room together"

l8s is a **remote-only** Podman-based development container management tool that creates isolated, SSH-accessible development environments on dedicated servers. Each container is a fully-featured Linux environment with development tools, accessible via SSH using key-based authentication.

**Security Note**: L8s ONLY supports remote container management. This design ensures all code execution happens on dedicated servers, providing complete isolation from developer laptops - perfect for AI workloads and untrusted code.

## Features

- **🔒 Secure**: SSH key-only authentication (no passwords)
- **💾 Stateful**: Containers persist between sessions with dedicated volumes
- **🔄 Git-Integrated**: Automatic repository cloning and remote management
- **🚀 Fast**: Containers ready in seconds
- **🛠️ Developer-Friendly**: Pre-configured with modern development tools

## Architecture

```mermaid
graph LR
    subgraph "Developer Laptop"
        L8S[l8s CLI]
        SSH_AGENT[ssh-agent]
        CONFIG[~/.config/l8s/config.yaml]
    end
    
    subgraph "Remote Server"
        subgraph "LXC Container (Fedora)"
            SSHD[SSH Daemon]
            PODMAN[Podman - running as root]
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

## Requirements

### On Your Laptop
- SSH client with ssh-agent
- Go 1.21+ (for building from source)
- Git

### On Remote Server (LXC Container Recommended)
- Linux (tested on Fedora in LXC)
- Podman 4.0+ (running as root)
- SSH server
- libgpgme-dev (or gpgme-devel on Fedora/RHEL)
- Proper permissions setup - see [Remote Server Setup Guide](docs/REMOTE_SERVER_SETUP.md)

## Installation

### Prerequisites

Install system dependencies:

```bash
# Fedora/RHEL/CentOS
sudo dnf install -y gpgme-devel

# Ubuntu/Debian
sudo apt-get update && sudo apt-get install -y libgpgme-dev

# macOS (using Homebrew)
brew install gpgme
```

### From Source

```bash
# Clone the repository
git clone https://github.com/l8s/l8s.git
cd l8s

# Build the binary
make build

# Install to /usr/local/bin (requires sudo)
sudo make install

# Or install to custom location
make install PREFIX=$HOME/.local
```

### Pre-built Binary

Coming soon!

## Quick Start

### 1. Set Up Remote Server

**Important**: Your remote server needs proper permissions configured for l8s to work correctly.

See the [Remote Server Setup Guide](docs/REMOTE_SERVER_SETUP.md) for detailed instructions on:
- Installing Podman
- Creating a podman group for socket access
- Configuring proper permissions
- Setting up SSH access
- Troubleshooting common issues

Quick setup for Fedora/RHEL:
```bash
# Install Podman
sudo dnf install -y podman

# Follow the full setup guide for permissions configuration
# See: docs/REMOTE_SERVER_SETUP.md
```

### 2. Configure L8s

On your laptop:

```bash
# Initialize l8s with your remote server
l8s init

# You'll be prompted for:
# - Remote server hostname/IP
# - Remote username (typically 'root' in LXC)
# - SSH key configuration
```

### 3. Build the Container Image

Build the base container image on the remote server:

```bash
l8s build
```

### 4. Create a Development Container

Create a new container with your Git repository:

```bash
l8s create myproject https://github.com/user/repo.git main
```

This will:
- Create a container named `dev-myproject`
- Clone your repository into `/workspace/project`
- Set up SSH access on port 2200
- Add SSH config entry for easy access
- Configure git remote for seamless pushing

### 3. Connect to Your Container

Three ways to connect:

```bash
# Using l8s
l8s ssh myproject

# Using SSH directly
ssh dev-myproject

# Using VS Code
code --remote ssh-remote+dev-myproject /workspace/project
```

## Core Commands

### Container Management

```bash
# Create a new container
l8s create <name> <git-url> [branch]

# List all containers
l8s list

# Start a stopped container
l8s start <name>

# Stop a running container
l8s stop <name>

# Remove a container
l8s remove <name>

# Get detailed container info
l8s info <name>
```

### Git Integration

```bash
# Add git remote for existing container
l8s remote add <name>

# Remove git remote
l8s remote remove <name>

# Work with git remotes
git push myproject  # Push to container
git pull myproject  # Pull from container
```

### Other Commands

```bash
# Execute command in container
l8s exec <name> <command>

# Rebuild container image
l8s build
```

## Configuration

L8s uses a YAML configuration file located at `~/.config/l8s/config.yaml`:

```yaml
# Remote server configuration (REQUIRED)
remote_host: "server.example.com"
remote_user: "root"
remote_socket: "/run/podman/podman.sock"
ssh_key_path: "~/.ssh/id_ed25519"

# Container configuration
ssh_port_start: 2200
base_image: "localhost/l8s-fedora:latest"
container_prefix: "dev"
container_user: "dev"
ssh_public_key: ""  # Auto-detected if empty
```

## Container Features

Each container includes:
- **Base OS**: Fedora latest
- **Shell**: Zsh with Oh-My-Zsh
- **Editor**: Neovim with modern config
- **Languages**: Python 3, Node.js, Go, GCC
- **Tools**: tmux, ripgrep, fd, fzf, bat, git
- **AI Assistant**: Claude Code (via npm)

## SSH Configuration

L8s automatically manages your `~/.ssh/config` file, adding entries like:

```
Host dev-myproject
    HostName server.example.com
    Port 2200
    User dev
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    ForwardAgent yes
    ControlMaster auto
    ControlPath ~/.ssh/control-%r@%h:%p
    ControlPersist 10m
```

**Performance Note**: SSH connections are multiplexed using ControlMaster, so subsequent connections reuse the existing tunnel for instant access.

This enables convenient access:
- `ssh dev-myproject` - Direct SSH access
- `scp file.txt dev-myproject:` - Easy file transfers
- VS Code Remote SSH integration

## Git Workflow

L8s integrates seamlessly with your Git workflow:

```bash
# Create container with your repo
l8s create myproject https://github.com/user/repo.git

# Work on your laptop
git commit -m "WIP: new feature"
git push myproject  # Push to container

# Test in container
l8s ssh myproject
# Your changes are already there!

# Push to GitHub when ready
git push origin
```

## Dotfiles

Place your dotfiles in the `dotfiles/` directory of the l8s installation. They will be automatically copied to new containers:

```
dotfiles/
├── .zshrc
├── .tmux.conf
├── .gitconfig
└── .config/
    └── nvim/
        └── init.lua
```

## Volumes

Each container has persistent volumes:
- `dev-<name>-home`: User home directory (`/home/dev`)
- `dev-<name>-workspace`: Project workspace (`/workspace`)

Data persists between container restarts.

## Migration from Local Containers

**BREAKING CHANGE**: L8s no longer supports local containers. To migrate:

1. Set up a remote server with Podman
2. Run `l8s init` to configure the remote connection
3. Recreate your containers on the remote server
4. Update any scripts that assumed local container access

## Security Considerations

1. **Remote-Only Design**: All containers run on dedicated servers, never locally
2. **LXC Isolation**: Run Podman inside LXC containers for additional isolation
3. **SSH Key Auth**: No password authentication, ssh-agent required
4. **Root Podman**: Runs as root inside isolated LXC container (not on host)

## Troubleshooting

### Initial Setup Issues
- Run `l8s init` to configure remote server connection
- Ensure SSH key is added: `ssh-add ~/.ssh/id_ed25519`
- Test SSH access: `ssh root@your-server`
- Verify Podman socket: `ssh root@your-server systemctl status podman.socket`

### Container Creation Fails
- Check remote connection: `ssh root@your-server podman version`
- Ensure base image exists: `ssh root@your-server podman images | grep l8s`
- Rebuild the image: `l8s build`

### SSH Connection Refused
- Check if container is running: `l8s list`
- Verify SSH port on remote: `ssh root@your-server ss -tlnp | grep 2200`
- Start the container: `l8s start <name>`

### Git Push/Pull Issues
- Ensure git remote exists: `git remote -v`
- Re-add remote: `l8s remote add <name>`
- Check SSH connectivity: `l8s ssh <name>`

## Development

### Project Structure
```
l8s/
├── cmd/               # CLI commands
├── pkg/               # Core packages
│   ├── container/     # Container management
│   ├── git/          # Git operations
│   ├── ssh/          # SSH handling
│   └── config/       # Configuration
├── containers/       # Dockerfiles
├── dotfiles/        # Default dotfiles
└── test/            # Integration tests
```

### Running Tests
```bash
# Unit tests
make test

# Integration tests (requires Podman)
make test-integration

# All tests with coverage
make test-coverage
```

### Building
```bash
# Build binary
make build

# Build container image
make build-image

# Clean build artifacts
make clean
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

MIT License - see LICENSE file for details

## Acknowledgments

- The Podman team for the excellent container runtime
- The Go team for the amazing language and tools
- The Dude, for abiding

---

*"Yeah, well, that's just, like, your container, man."* 🎳
