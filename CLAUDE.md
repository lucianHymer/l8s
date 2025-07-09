# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

- Always run `make ci` when done with changes

## Project Overview

L8s (Lebowskis) is a remote-only Podman container management tool that creates isolated development environments on dedicated servers via SSH. Each container is a fully-featured Linux environment with persistent volumes for home and workspace directories.

Key architectural principles:
- **Remote-only**: Containers run exclusively on remote servers, never locally
- **SSH-based**: All communication happens over SSH with multiplexing for performance
- **Git-aware**: Automatic repository cloning and remote management
- **Security-focused**: Designed for isolated, non-production development environments

## Essential Development Commands

### Building and Installation
```bash
make build              # Build the l8s binary in ./build/
make install            # Install to /usr/local/bin (requires sudo)
make clean              # Clean build artifacts
```

### Testing
```bash
make test               # Run all tests (Go unit + ZSH plugin tests)
make test-go            # Run Go unit tests only
make test-integration   # Run integration tests (requires Podman)
make test-coverage      # Generate test coverage report
```

### Code Quality
```bash
make lint               # Run linters (golangci-lint or go vet)
make fmt                # Format Go code
make tidy               # Tidy go modules
```

### Development Workflow
```bash
make setup              # Initial setup - check dependencies
make dev                # Run l8s in development mode
make watch              # Watch files and run tests on changes (requires entr)
```

## Architecture and Code Structure

### Package Organization
- `cmd/l8s/` - CLI entry point and command definitions
- `pkg/cli/` - Command implementations using CommandFactory pattern
- `pkg/container/` - Podman container management logic
- `pkg/git/` - Git remote detection and management
- `pkg/ssh/` - SSH key generation and management
- `pkg/config/` - Configuration handling with Viper
- `pkg/embed/` - Embedded dotfiles for containers

### Key Design Patterns
1. **CommandFactory Pattern**: All CLI commands use dependency injection for testability. Commands are created through factories that accept dependencies.
2. **Interface-based Design**: Core components (container client, git operations) are defined as interfaces with mock implementations for testing.
3. **Remote-only Operations**: The container client always connects to remote Podman instances via SSH tunnels.

### Testing Approach
- Unit tests live alongside source files (`*_test.go`)
- Integration tests in `/test/integration/`
- Mock container client available for testing without Podman
- ZSH plugin has its own test suite with custom framework

## Development Guidelines

### When Adding New Commands
1. Create command in `cmd/l8s/cmd_*.go`
2. Implement logic in `pkg/cli/` using CommandFactory pattern
3. Add comprehensive unit tests with mocks
4. Update integration tests if needed
5. Run `make test` to ensure all tests pass

### Git Integration
- The system automatically detects git repositories in mounted directories
- Creates remotes like `l8s-<container>-<branch>` pointing to container workspaces
- Remote format: `ssh://dev@<host>:<port>/workspace/<repo>`

### Container Management
- Containers are named `<prefix>-<purpose>` (e.g., `dev-myproject`)
- Each container gets unique SSH port starting from 2200
- Persistent volumes: `/home/dev` and `/workspace`
- Base image: `localhost/l8s-fedora:latest`

### Configuration
Config file: `~/.config/l8s/config.yaml`
Key settings:
- `remote_host`: Target server hostname
- `ssh_key_path`: SSH key for authentication
- `container_prefix`: Prefix for container names
- `base_image`: Container image to use

## Common Development Tasks

### Running a Single Test
```bash
go test -v -run TestSpecificFunction ./pkg/cli/
```

### Building Container Images
```bash
make container-build      # Build main l8s container image
make container-build-test # Build test container image
```

### Debugging SSH Connections
The system uses SSH ControlMaster for connection multiplexing. Check `~/.ssh/` for control sockets if experiencing connection issues.

### Working with Embedded Dotfiles
Dotfiles in `pkg/embed/dotfiles/` are embedded in the binary and deployed to new containers. Update these files to change default container configurations.
