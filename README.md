# 🎳 L8s (Lebowskis)

> "The container management system that really ties the room together"

L8s is a **git-native**, **remote-only** development container system that creates isolated, SSH-accessible environments on dedicated servers. Each git repository gets its own persistent container with full development tools.

## 🔒 Security Architecture

L8s uses a **defense-in-depth architecture** for secure remote development:
- **Unprivileged LXC container** provides strong isolation from the host
- **Root Podman** inside the LXC for optimal performance
- **SSH Certificate Authority** prevents MITM attacks with cryptographic container verification
- **Remote-only execution** - code never runs on your laptop

This architecture provides excellent isolation for development workloads while maintaining the performance needed for real work.

## Key Features

- **🔗 Git-Native**: Seamlessly extends your git workflow - containers are tied to repositories
- **🔒 Secure by Design**: SSH Certificate Authority prevents MITM attacks, remote-only execution
- **⚡ Fast**: Containers ready in seconds, SSH multiplexing for instant connections  
- **💾 Persistent**: Your work survives container restarts with dedicated volumes
- **🛠️ Fully Featured**: Modern dev tools, Neovim, tmux, GitHub CLI, and more

## Quick Install

```bash
# Clone and build
git clone https://github.com/yourusername/l8s.git
cd l8s
make build
sudo make install

# Initialize (one-time setup)
l8s init
```

That's it! L8s will prompt for your remote server details and GitHub token during init.

## Quick Start

**L8s requires you to be in a git repository** for most commands:

```bash
# From inside your git repo
cd ~/projects/my-app

# Create a development container for this repo
l8s create

# SSH into your container  
l8s ssh

# Your code is already there at /workspace/project!
```

## Core Workflow

L8s extends git with remote development containers:

```bash
# Work locally, test remotely
git commit -m "New feature"
l8s push              # Push changes to container
l8s ssh               # Test in isolated environment

# Or use your favorite editor
code --remote ssh-remote+dev-myapp-a3f2d1 /workspace/project
```

## Common Commands

All commands work from within a git repository:

```bash
l8s create            # Create container for current repo
l8s ssh               # SSH into container
l8s push              # Push current branch to container
l8s rebuild           # Rebuild container (preserves data)
l8s rm                # Remove container
l8s exec <command>    # Run command in container
```

Global commands (work anywhere):

```bash
l8s list              # List all containers
l8s build             # Build container base image
l8s init              # Initial setup
```

## Git-Native Design

L8s automatically:
- Names containers based on your repository and worktree
- Maintains the same branch in the container
- Creates git remotes for seamless push/pull
- Maps each worktree to its own container

For example, if you're in `/Users/you/projects/myapp`:
- Container name: `dev-myapp-<hash>`
- Automatic SSH config entry
- Git remote: `l8s-dev-myapp-<hash>`

## Container Environment

Each container includes:
- **OS**: Fedora latest with systemd
- **Shell**: Zsh with Oh-My-Zsh  
- **Editor**: Neovim with modern config
- **Languages**: Go, Python, Node.js, Rust
- **Tools**: GitHub CLI, tmux, ripgrep, fzf
- **Security**: SSH Certificate Authority managed

## SSH Access

Three ways to connect:

```bash
# Via l8s
l8s ssh

# Direct SSH (after l8s creates the config)
ssh dev-myapp-a3f2d1

# VS Code Remote SSH
code --remote ssh-remote+dev-myapp-a3f2d1 /workspace/project
```

## Architecture

L8s is **remote-only** - containers never run on your laptop:

```
Your Laptop                    Remote Server
-----------                    -------------
l8s CLI          SSH          Podman (rootful)
Git repo    ------------->    Dev containers
SSH agent                     Persistent volumes
```

## Documentation

- [Remote Server Setup](docs/REMOTE_SERVER_SETUP.md) - Configuring your server
- [Configuration Guide](docs/CONFIGURATION.md) - Detailed config options
- [Troubleshooting](docs/TROUBLESHOOTING.md) - Common issues and solutions
- [Development Guide](docs/DEVELOPMENT.md) - Building and contributing

## Requirements

**Your laptop**: SSH client, Git  
**Remote server**: Linux, Podman 4+, SSH server

## Security

- **Remote-only execution**: Code never runs on your laptop
- **SSH Certificate Authority**: Cryptographic verification of container identity
- **No passwords**: SSH key authentication only
- **Isolated environments**: Each container is fully separated

## License

MIT - See LICENSE file

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

*"That container really tied the room together, did it not?"*
