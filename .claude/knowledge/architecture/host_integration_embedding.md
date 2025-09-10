# Host Integration Embedding System

L8s provides a sophisticated embedding system for host integration files, separate from container dotfiles.

## Architecture Overview

The host integration embedding system allows L8s to bundle host-side tools (like the ZSH plugin) within the binary for easy installation.

## Core Components

### Embedding Module
**Location**: `pkg/embed/host_integration.go`

This module:
- Embeds the entire `host-integration` directory
- Provides `ExtractZSHPlugin()` function for plugin installation
- Manages extraction to user's Oh My Zsh installation

### Installation Command
**Command**: `l8s install-zsh-plugin`

This command:
- Extracts the embedded ZSH completion plugin
- Installs to `~/.oh-my-zsh/custom/plugins/l8s`
- Replaces the broken Makefile approach

## Design Benefits

1. **Binary Distribution**: Host tools bundled within l8s binary
2. **Separation of Concerns**: Host integration clearly separated from container dotfiles
3. **Easy Installation**: Single command to install host-side tools
4. **Self-Contained**: No external files needed for host setup

## File Organization

The system maintains clear separation:
- **Container dotfiles**: `pkg/embed/dotfiles/` - deployed to containers
- **Host integration**: `pkg/embed/host_integration/` - installed on host
- Files embedded in binary for host installation but not included in containers

## Related Files
- `pkg/embed/host_integration.go` - Embedding and extraction logic
- `pkg/cli/factory_lazy.go` - Command factory with install command
- `pkg/cli/handlers.go` - Handler for install-zsh-plugin
- `cmd/l8s/main.go` - Command registration